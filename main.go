// This is the main file main.go

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

func main() {

	var TrackHandler = &TrackHandler{
		CurrTrack:  0,
		TrackCount: 0,
		Packets:    make(chan *rtp.Packet, 60),
	}

	// Define ICE servers for the WebRTC configuration
	iceServers := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{
					"stun:stun.l.google.com:19302",  // First STUN Server
					"stun:stun1.l.google.com:19302", // Second STUN Server
					"stun:stun2.l.google.com:19302", // Third STUN Server
				},
			},
			{
				URLs: []string{
					"turn:turn.example.com:3478", // Fallback TURN Server
				},
				Username:   "user",
				Credential: "pass",
			},
		},
	}

	fmt.Printf("Using ICE Servers: %+v\n", iceServers)

	// Create a new PeerConnection with the defined ICE servers (P2P)
	peerConnection, err := webrtc.NewPeerConnection(iceServers)
	if err != nil {
		panic(err)
	}

	defer func() {
		err := peerConnection.Close()
		if err != nil {
			fmt.Printf("Cannot close PeerConnection: %v\n", err)
		}
	}()

	fmt.Printf("PeerConnection created successfully with ICE servers.%+v\n", peerConnection)

	// =====================================================================================
	// ================================ Create Video Track =================================
	// =====================================================================================

	// Create Track to send video
	trackVideo, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{
		MimeType: webrtc.MimeTypeVP8,
	}, "video", "pion-video")

	if err != nil {
		panic(err)
	}

	fmt.Printf("Video Track created: %+v\n", trackVideo)

	// Add the video track to the PeerConnection (RTP Sender)
	sendVideo, err := peerConnection.AddTrack(trackVideo)
	if err != nil {
		fmt.Printf("Failed to add video track: %v\n", err)
		return
	}

	fmt.Printf("Send Video: %+v\n", sendVideo)

	// =====================================================================================
	// ================================ Create Screen Track ================================
	// =====================================================================================

	// Create Track to send screen
	trackScreen, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{
		MimeType: webrtc.MimeTypeVP8,
	}, "screen", "pion-screen")

	if err != nil {
		panic(err)
	}

	fmt.Printf("Screen Track created: %+v\n", trackScreen)

	// Add the screen track to the PeerConnection (RTP Sender)
	sendScreen, err := peerConnection.AddTrack(trackScreen)
	if err != nil {
		fmt.Printf("Failed to add screen track: %v\n", err)
		return
	}

	fmt.Printf("Send Screen: %+v\n", sendScreen)

	// =====================================================================================
	// ================================ Handle RTCP Packets ================================
	// =====================================================================================

	// Handle incoming RTCP packets for the video track
	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			_, _, rtcpErr := sendVideo.Read(rtcpBuf)
			if rtcpErr != nil {
				fmt.Printf("RTCP read error: %v\n", rtcpErr)
				return
			}
		}
	}()

	// Wait for the offer/answer exchange to complete -> Process Signaling
	offer := webrtc.SessionDescription{}
	input := readUntilNewLine()

	if err := decode(input, &offer); err != nil {
		fmt.Printf("Failed to decode offer: %v\n", err)
		return
	}

	fmt.Printf("Decoded offer SDP: %+v\n", offer)

	peerConnection.OnTrack(
		func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
			fmt.Printf("Track has started, of type %d: %s \n", track.PayloadType(), track.Codec().MimeType)
			trackNum := TrackHandler.TrackCount
			TrackHandler.TrackCount++

			// The last timestamp received
			var lastTimeStamp uint32

			// Check if the receiver is taking new video or updating existing video
			var isCurrTrack bool

			for {
				// Read RTP packets being sent to Pion
				rtp, _, readErr := track.ReadRTP()
				if readErr != nil {
					fmt.Printf("RTP read error: %v\n", readErr)
					return
				}

				// Change the timestamp to only be the delta
				oldTimeStamp := lastTimeStamp
				if lastTimeStamp == 0 {
					rtp.Timestamp = 0
				} else {
					rtp.Timestamp -= lastTimeStamp
				}
				lastTimeStamp = oldTimeStamp

				// Check if this is current track
				// If active, then send RTP packets to the channel
				if TrackHandler.CurrTrack == trackNum {
					if !isCurrTrack {
						isCurrTrack = true
						if track.Kind() == webrtc.RTPCodecTypeVideo {
							if writeErr := peerConnection.WriteRTCP([]rtcp.Packet{
								&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())},
							}); writeErr != nil {
								fmt.Printf("Failed to send PLI: %v\n", writeErr)
							}
						}
						TrackHandler.Packets <- rtp
					}
				} else {
					isCurrTrack = false
				}
			}
		})

	ctx, done := context.WithCancel(context.Background())

	// Set the handler for Peer connection state
	// This will notify you when the peer has connected/disconnected

	peerConnection.OnConnectionStateChange(
		func(state webrtc.PeerConnectionState) {
			fmt.Printf("Peer connection has changed: %s\n", state.String())

			if state == webrtc.PeerConnectionStateFailed {
				fmt.Printf("Timeout Countdown 30s Started \n")
				done()
			}

			if state == webrtc.PeerConnectionStateClosed {
				fmt.Printf("Peer Connection Closed, caused by %s\n", state.String())
				done()
			}
		})

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		fmt.Printf("Failed to create answer: %v\n", err)
		return
	}

	fmt.Printf("Created answer SDP: %+v\n", answer)

	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		fmt.Printf("Failed to set local description: %v\n", err)
		return
	}

	<-gatherComplete

	fmt.Println(encode(peerConnection.CurrentLocalDescription()))

	// Asynchronously take all packets in the channel and write them out to our
	// track

	go func() {
		var currTimestamp uint32
		for i := uint16(0); ; i++ {
			packet := <-TrackHandler.Packets
			// Timestamp on the RTP packet is really a diff, so we need to add it to current
			currTimestamp += packet.Timestamp
			packet.Timestamp = currTimestamp
			// Keep increasing sequence number
			packet.SequenceNumber = i
			// Write packet to track, with ignoring pipe if nobody is listening
			if err := trackVideo.WriteRTP(packet); err != nil {
				fmt.Printf("Failed to write RTP packet: %v\n", err)
			}

			panic(err)
		}
	}()

	// Wait for connection, then rotate every 5 seconds
	fmt.Printf("Waiting for connection to rotate tracks...\n")
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("Exiting track rotation due to context done.\n")
			return
		default:
		}

		// If there are no tracks, continue waiting
		if TrackHandler.TrackCount == 0 {
			continue
		}

		fmt.Printf("Waiting 5 seconds before switching to next track...\n")
		time.Sleep(5 * time.Second)
		if TrackHandler.CurrTrack == TrackHandler.TrackCount-1 {
			TrackHandler.CurrTrack = 0
		} else {
			TrackHandler.CurrTrack++
		}
		fmt.Printf("Switched to track #%v\n", TrackHandler.CurrTrack+1)
	}
}

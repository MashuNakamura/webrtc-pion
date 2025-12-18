// This is the file WebRTC implementation logic

package webrtc

import (
	"context"
	"fmt"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

func StartPeerConnecion() {

	// Load from TrackHandler Struct (struct.go)
	var TrackHandler = &TrackHandler{
		CurrTrack:     0,                          // Start from first track
		TrackCount:    0,                          // No tracks yet
		videoPackets:  make(chan *rtp.Packet, 60), // Buffer for video packets
		screenPackets: make(chan *rtp.Packet, 60), // Buffer for screen packets
		screenInUse:   false,                      // Screen track usage flag
	}

	// Define ICE servers for the WebRTC configuration
	// Total 3 STUN servers and 1 TURN server as fallback
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

	// Print ICE servers being used
	fmt.Printf("Using ICE Servers: %+v\n", iceServers)

	// Create a new PeerConnection with the defined ICE servers (P2P)
	peerConnection, err := webrtc.NewPeerConnection(iceServers)
	if err != nil {
		panic(err)
	}

	// Close the PeerConnection when done and send error if any
	defer func() {
		err := peerConnection.Close()
		if err != nil {
			fmt.Printf("Cannot close PeerConnection: %v\n", err)
		}
	}()

	// Print successful creation of PeerConnection
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

	// Print successful creation of Video Track
	fmt.Printf("Video Track created: %+v\n", trackVideo)

	// Add the video track to the PeerConnection (RTP Sender)
	sendVideo, err := peerConnection.AddTrack(trackVideo)
	if err != nil {
		fmt.Printf("Failed to add video track: %v\n", err)
		return
	}

	// Print successful addition of Video Track
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

	// Print successful creation of Screen Track
	fmt.Printf("Screen Track created: %+v\n", trackScreen)

	// Add the screen track to the PeerConnection (RTP Sender)
	sendScreen, err := peerConnection.AddTrack(trackScreen)
	if err != nil {
		fmt.Printf("Failed to add screen track: %v\n", err)
		return
	}

	// Print successful addition of Screen Track
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

	// Handle incoming RTCP packets for the screen track
	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			_, _, rtcpErr := sendScreen.Read(rtcpBuf)
			if rtcpErr != nil {
				fmt.Printf("RTCP read error: %v\n", rtcpErr)
				return
			}
		}
	}()

	// =====================================================================================
	// ================================ Handle Incoming Track =============================
	// =====================================================================================

	// Wait for the offer/answer exchange to complete -> Process Signaling
	offer := webrtc.SessionDescription{}
	input := readUntilNewLine()

	if err := decode(input, &offer); err != nil {
		fmt.Printf("Failed to decode offer: %v\n", err)
		return
	}

	// Print decoded offer SDP
	fmt.Printf("Decoded offer SDP: %+v\n", offer)

	// Set the remote SessionDescription
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
					if track.StreamID() == "screen" {
						TrackHandler.screenInUse = false
					}
					return
				}

				// Change the timestamp to only be the delta
				if lastTimeStamp == 0 {
					lastTimeStamp = rtp.Timestamp
					rtp.Timestamp = 0
				} else {
					rtp.Timestamp -= lastTimeStamp
					lastTimeStamp += rtp.Timestamp
				}

				// Check if this is current track
				// If active, then send RTP packets to the channel
				if TrackHandler.CurrTrack == trackNum {
					if !isCurrTrack {
						isCurrTrack = true
						// Check if the channel is existing
						if TrackHandler.videoPackets != nil {
							if track.Kind() == webrtc.RTPCodecTypeVideo {
								if track.StreamID() == "video" {
									TrackHandler.videoPackets <- rtp
								} else if track.StreamID() == "screen" {
									TrackHandler.screenInUse = true
									TrackHandler.screenPackets <- rtp
								}
							}
						}
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
			// If peer connection is closed or failed, exit the program
			fmt.Printf("Peer connection has changed: %s\n", state.String())

			// Handle failed connection
			if state == webrtc.PeerConnectionStateFailed {
				fmt.Printf("Timeout Countdown 30s Started \n")
				done()
			}

			// Handle closed connection
			if state == webrtc.PeerConnectionStateClosed {
				fmt.Printf("Peer Connection Closed, caused by %s\n", state.String())
				done()
			}
		})

	// Create answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		fmt.Printf("Failed to create answer: %v\n", err)
		return
	}

	// Print created answer SDP
	fmt.Printf("Created answer SDP: %+v\n", answer)

	// Sets the LocalDescription, and starts our UDP listeners
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	// Error handling for setting local description
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		fmt.Printf("Failed to set local description: %v\n", err)
		return
	}

	// Send packet gathering complete signal
	<-gatherComplete

	// Output the answer in base64 so we can paste it in browser
	fmt.Println(encode(peerConnection.CurrentLocalDescription()))

	// Asynchronously take all packets in the channel and write them out to our
	// Video Track
	go func() {
		// Keep track of current timestamp for video
		var currTimestampVideo uint32
		for i := uint16(0); ; i++ {
			packet := <-TrackHandler.videoPackets
			// Timestamp on the RTP packet is really a diff, so we need to add it to current
			currTimestampVideo += packet.Timestamp
			packet.Timestamp = currTimestampVideo
			// Keep increasing sequence number
			packet.SequenceNumber = i
			// Write packet to track, with ignoring pipe if nobody is listening
			if err := trackVideo.WriteRTP(packet); err != nil {
				fmt.Printf("Failed to write RTP packet: %v\n", err)
			}
		}
	}()

	// Screen Track
	go func() {
		// Keep track of current timestamp for screen
		var currTimestampScreen uint32
		for i := uint16(0); ; i++ {
			packet := <-TrackHandler.screenPackets
			// Timestamp on the RTP packet is really a diff, so we need to add it to current
			currTimestampScreen += packet.Timestamp
			packet.Timestamp = currTimestampScreen
			// Keep increasing sequence number
			packet.SequenceNumber = i
			// Write packet to track, with ignoring pipe if nobody is listening
			if err := trackScreen.WriteRTP(packet); err != nil {
				fmt.Printf("Failed to write RTP packet: %v\n", err)
			}
		}
	}()

	// Wait for connection, then rotate every 5 seconds
	fmt.Printf("Waiting for connection to rotate tracks...\n")
	for {
		// Case to exit goroutine if context is done
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

		// Will be used if the track more than 1 is needed in future

		// fmt.Printf("Waiting 5 seconds before switching to next track...\n")
		// time.Sleep(5 * time.Second)
		// if TrackHandler.CurrTrack == TrackHandler.TrackCount-1 {
		// 	TrackHandler.CurrTrack = 0
		// } else {
		// 	TrackHandler.CurrTrack++
		// }
		// fmt.Printf("Switched to track #%v\n", TrackHandler.CurrTrack+1)
	}
}

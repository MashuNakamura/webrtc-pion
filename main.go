// This is the main file main.go

package main

import (
	"fmt"

	"github.com/pion/webrtc/v4"
)

func main() {
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

	// Create Track to send video
	trackVideo, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{
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
}

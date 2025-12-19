// This is the file WebRTC implementation logic (webrtc.go)

package webrtc

import (
	"context"
	"fmt"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

// GlobalTrackHandler to manage tracks and packets
var GlobalTrackHandler *TrackHandler

// StartPeerConnection initializes and starts the WebRTC PeerConnection
func StartPeerConnection(offerChan chan string, answerChan chan string) {

	// Add infinite loop, So that after one session ends, it can start a new session
	for {
		fmt.Println("\n[SYSTEM] Waiting for new connection session...")

		// Anonymous function to encapsulate each session
		func() {
			// Load from TrackHandler Struct (struct.go)
			GlobalTrackHandler = &TrackHandler{
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
						Username:   "user", // TURN Dummy Credential
						Credential: "pass",
					},
				},
			}

			// Print ICE servers being used
			fmt.Println("\n[INIT] WebRTC Configuration Loaded")
			fmt.Println("[INIT] Using ICE Servers: STUN (Google) + TURN (Fallback)")

			// Create a new PeerConnection with the defined ICE servers (P2P)
			peerConnection, err := webrtc.NewPeerConnection(iceServers)
			if err != nil {
				fmt.Printf("[ERROR] Failed to create PeerConnection: %v\n", err)
				return // Lanjut ke loop berikutnya
			}

			// Close the PeerConnection when done (Session Selesai)
			defer func() {
				err := peerConnection.Close()
				if err != nil {
					fmt.Printf("[ERROR] Cannot close PeerConnection:  %v\n", err)
				}
				fmt.Println("[SYSTEM] Session Ended. Cleaning up...")
			}()

			// Print successful creation of PeerConnection
			fmt.Printf("[INIT] PeerConnection Created Successfully\n")

			// =====================================================================================
			// ================================ Create Video Track =================================
			// =====================================================================================

			// Create Track to send video
			trackVideo, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{
				MimeType: webrtc.MimeTypeVP8,
			}, "video", "pion-video")

			// Check for error during track creation
			if err != nil {
				panic(err)
			}

			// Add the video track to the PeerConnection (RTP Sender)
			sendVideo, err := peerConnection.AddTrack(trackVideo)
			if err != nil {
				fmt.Printf("[ERROR] Failed to add video track:  %v\n", err)
				return
			}

			// Print successful addition of Video Track
			fmt.Printf("[TRACK] Video Track Created & Added (ID: pion-video)\n")

			// =====================================================================================
			// ================================ Create Screen Track ================================
			// =====================================================================================

			// Create Track to send screen
			trackScreen, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{
				MimeType: webrtc.MimeTypeVP8,
			}, "screen", "pion-screen")

			// Check for error during track creation
			if err != nil {
				panic(err)
			}

			// Add the screen track to the PeerConnection (RTP Sender)
			sendScreen, err := peerConnection.AddTrack(trackScreen)
			if err != nil {
				fmt.Printf("[ERROR] Failed to add screen track: %v\n", err)
				return
			}

			// Print successful addition of Screen Track
			fmt.Printf("[TRACK] Screen Track Created & Added (ID: pion-screen)\n")

			// =====================================================================================
			// ================================ Handle RTCP Packets ================================
			// =====================================================================================

			// Handle incoming RTCP packets for the video track
			go func() {
				rtcpBuf := make([]byte, 1500)
				for {
					_, _, rtcpErr := sendVideo.Read(rtcpBuf)
					if rtcpErr != nil {
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
						return
					}
				}
			}()

			// =====================================================================================
			// ================================ Handle Incoming Track ==============================
			// =====================================================================================

			peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
				// Assign track number based on current track count
				trackNum := GlobalTrackHandler.TrackCount
				GlobalTrackHandler.TrackCount++

				// Print track information
				fmt.Printf("\n[STREAM-IN] Track Detected! Type: %s | ID: %s | Order: %d\n", track.Kind(), track.ID(), trackNum)

				for {
					// Read RTP packets from the incoming track
					packet, _, readErr := track.ReadRTP()
					if readErr != nil {
						if trackNum > 0 {
							GlobalTrackHandler.screenInUse = false
							fmt.Println("[STREAM-END] Screen Sharing Stopped")
						}
						return
					}

					// Forward the same packet to the appropriate channel
					if trackNum == 0 {
						select {
						case GlobalTrackHandler.videoPackets <- packet:
							fmt.Printf("[STREAM-END] Camera Stopped (ID: %s)\n", track.ID())
						default:
						}
					} else {
						GlobalTrackHandler.screenInUse = true
						select {
						case GlobalTrackHandler.screenPackets <- packet:
							fmt.Printf("[STREAM-END] Screen Sharing Stopped (ID: %s)\n", track.ID())
						default:
						}
					}
				}
			})

			// =====================================================================================
			// ================================ Connection State ===================================
			// =====================================================================================

			// Create a context to manage the lifecycle of the signaling loop
			ctx, done := context.WithCancel(context.Background())

			// Set the handler for Peer connection state
			peerConnection.OnConnectionStateChange(
				func(state webrtc.PeerConnectionState) {
					fmt.Printf("[STATE] Peer Connection Changed:  %s\n", state.String())

					if state == webrtc.PeerConnectionStateFailed {
						fmt.Printf("[STATE] Connection Failed - Timeout started \n")
						done() // Trigger exit signal
					}

					if state == webrtc.PeerConnectionStateClosed {
						fmt.Printf("[STATE] Peer Connection Closed\n")
						done() // Trigger exit signal
					}
				})

			// =====================================================================================
			// ================================ Signaling Loop =====================================
			// =====================================================================================

			go forwardVideoPackets(trackVideo)   // Forward video packets to the track
			go forwardScreenPackets(trackScreen) // Forward screen packets to the track

			// Print ready status
			fmt.Printf("[READY] WebRTC Setup Complete. Awaiting signaling...\n")

			for {
				select {
				// Exit the loop if context is done (Connection Closed/Failed)
				case <-ctx.Done():
					fmt.Printf("Exiting signaling loop due to context done.\n")
					return // Exit the anonymous function to reset for new session
				// Handle incoming offer from the channel
				case offerSDP := <-offerChan:
					// Process the received offer SDP
					answerSDP := processOffer(peerConnection, offerSDP)
					if answerSDP != "" {
						select {
						case answerChan <- answerSDP:
							fmt.Printf("Answer SDP sent successfully.\n")
						default:
							fmt.Printf("[WARN] Answer channel is full, unable to send answer SDP.\n")
						}
					}
				}
			}
		}() // Execute Anonymous Function
	} // End of Infinite Loop
}

// =====================================================================================
// ================================ Helper Functions ===================================
// =====================================================================================

// Process the received offer SDP and return the encoded answer SDP
func processOffer(peerConnection *webrtc.PeerConnection, offerSDP string) string {
	fmt.Printf("   > [WEBRTC] Processing Offer SDP... \n")

	// Decode the offer SDP
	offer := webrtc.SessionDescription{}
	if err := Decode(offerSDP, &offer); err != nil {
		fmt.Printf("[ERROR] Failed to decode offer: %v\n", err)
		return ""
	}

	// Set the remote SessionDescription
	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		fmt.Printf("[ERROR] Failed to set remote description: %v\n", err)
		return ""
	}

	// Create answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		fmt.Printf("[ERROR] Failed to create answer: %v\n", err)
		return ""
	}

	// Sets the LocalDescription, and starts our UDP listeners
	if err := peerConnection.SetLocalDescription(answer); err != nil {
		fmt.Printf("[ERROR] Failed to set local description: %v\n", err)
		return ""
	}

	// Wait for ICE gathering to complete
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)
	<-gatherComplete

	// Encode and return the answer SDP
	encodedAnswer, err := Encode(peerConnection.CurrentLocalDescription())
	if err != nil {
		fmt.Printf("[ERROR] Failed to encode answer SDP: %v\n", err)
		return ""
	}

	fmt.Printf("   > [WEBRTC] Answer Created Successfully\n")
	return encodedAnswer
}

// Forward video RTP packets from the channel to the video track
func forwardVideoPackets(trackVideo *webrtc.TrackLocalStaticRTP) {
	for {
		// Take packet from videoPackets channel
		packet := <-GlobalTrackHandler.videoPackets

		// Send it directly without modifying timestamp/sequence
		if err := trackVideo.WriteRTP(packet); err != nil {
			continue
		}
	}
}

// Forward screen RTP packets from the channel to the screen track
func forwardScreenPackets(trackScreen *webrtc.TrackLocalStaticRTP) {
	for {
		// Take packet from screenPackets channel
		packet := <-GlobalTrackHandler.screenPackets

		// Send it directly without modifying timestamp/sequence
		if err := trackScreen.WriteRTP(packet); err != nil {
			continue
		}
	}
}

// GetTrackStatus returns the current status of tracks (How much Screen and Camera are in use) but for now is not really needed
func GetTrackStatus() map[string]interface{} {
	// If no one using TrackHandler, return default values or default map
	if GlobalTrackHandler == nil {
		return map[string]interface{}{
			"screenInUse": false,
			"cameraInUse": false,
			"trackCount":  0,
		}
	}
	// Return the current status of tracks
	return map[string]interface{}{
		"screenInUse": GlobalTrackHandler.screenInUse,
		"cameraInUse": !GlobalTrackHandler.screenInUse,
		"trackCount":  GlobalTrackHandler.TrackCount,
	}
}

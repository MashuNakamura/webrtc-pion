// This is the main file to start HTTP server and WebRTC (main.go)

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"webrtc-demo/webrtc"

	pion "github.com/pion/webrtc/v4"
)

// Channels for communication between HTTP handlers and WebRTC goroutine
var (
	offerChan  = make(chan string, 1)
	answerChan = make(chan string, 1)
)

func main() {
	// Start WebRTC
	go webrtc.StartPeerConnection(offerChan, answerChan)

	// Implement API Endpoints
	http.HandleFunc("/offer", handleOffer)
	http.HandleFunc("/track-status", handleTrackStatus)

	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// Start the server
	log.Println("Server running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

// API Endpoint to handle WebRTC offer from frontend and return answer
func handleOffer(w http.ResponseWriter, r *http.Request) {
	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle preflight OPTIONS request for CORS
	if r.Method == "OPTIONS" {
		return
	}

	fmt.Println("\n========================================================")
	fmt.Println("[HTTP] Received Offer Request from Browser")

	// Parse offer from request body as pion.SessionDescription object (Frontend send JSON Request)
	var offer pion.SessionDescription
	if err := json.NewDecoder(r.Body).Decode(&offer); err != nil {
		http.Error(w, "Wrong format JSON", 400)
		return
	}

	fmt.Printf("   > Type: %s | SDP Size: %d chars\n", offer.Type, len(offer.SDP))

	// Encode offer to base64 string
	offerSDP, err := webrtc.Encode(&offer)
	if err != nil {
		http.Error(w, "Error encoding offer SDP", 500)
		return
	}

	fmt.Printf("   > Encoded SDP (Base64) ready to send to channel\n")

	// Send offer
	offerChan <- offerSDP

	select {
	case answerB64 := <-answerChan:
		// Generate answer from base64 string
		var sendAnswer pion.SessionDescription

		// Decode answer from base64
		if err := webrtc.Decode(answerB64, &sendAnswer); err != nil {
			http.Error(w, "Error decoding answer SDP", 500)
			return
		}

		fmt.Printf("[HTTP] Generated Answer SDP (Type: %s)\n", sendAnswer.Type)

		// Send answer back to browser as JSON
		answerJSON, err := json.Marshal(sendAnswer)
		if err != nil {
			http.Error(w, "Error encoding answer JSON", 500)
			return
		}

		fmt.Printf("[HTTP] Sending JSON Answer to Browser (Size: %d bytes)\n", len(answerJSON))
		fmt.Println("========================================================")

		// Set response headers and write answer
		w.Header().Set("Content-Type", "application/json")
		w.Write(answerJSON) // Send JSON answer back to browser

	case <-time.After(time.Second * 10): // Case if Timeout for 10 seconds
		fmt.Println("[HTTP] Timeout waiting for WebRTC answer")
		http.Error(w, "Timeout waiting for WebRTC answer", 500)
	}
}

// API Endpoint for monitoring/debugging track status from frontend (But not really needed for now)
func handleTrackStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Get track status from webrtc package
	status := webrtc.GetTrackStatus()

	fmt.Printf("[DEBUG] Track Status Check: %+v\n", status)

	// Send status as JSON response
	response, _ := json.Marshal(status)
	w.Write(response)
}

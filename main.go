package main

import (
	"log"
	"net/http"

	"webrtc-demo/webrtc"
)

func main() {
	// WebRTC
	go webrtc.StartPeerConnecion()

	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// Start the server
	log.Println("Server running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

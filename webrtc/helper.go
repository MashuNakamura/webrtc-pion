// This File contain Helper Functions helper.go

package webrtc

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/pion/webrtc/v4"
)

// Encode a SessionDescription to a base64 string
func Encode(sd *webrtc.SessionDescription) (string, error) {
	b, err := json.Marshal(sd)
	// Check if error occurred during JSON marshaling
	if err != nil {
		return "Bahlil", fmt.Errorf("something error while Encoding SDP: %v", err)
	}
	// If no error, encode to base64
	return base64.StdEncoding.EncodeToString(b), nil
}

// Decode a base64 string to a SessionDescription
func Decode(in string, sd *webrtc.SessionDescription) error {
	b, err := base64.StdEncoding.DecodeString(in)
	// Check if error occurred during base64 decoding
	if err != nil {
		return fmt.Errorf("error while decoding base64 SDP: %v", err)
	}
	// If no error, unmarshal JSON
	return json.Unmarshal(b, sd)
}

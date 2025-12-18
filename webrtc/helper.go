// This File contain Helper Functions helper.go

package webrtc

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pion/webrtc/v4"
)

// Read SessionDescription from stdin until a non-empty line is found
func readUntilNewLine() (in string) {
	// Create a new buffered reader from standard input
	r := bufio.NewReader(os.Stdin)

	for {
		// Read every line until newline character
		in, err := r.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			panic(err)
		}

		// Delete First and Last whitespace characters for pretty printing
		in = strings.TrimSpace(in)
		if len(in) > 0 {
			return in
		}
	}
}

// Encode a SessionDescription to a base64 string
func encode(sd *webrtc.SessionDescription) (string, error) {
	b, err := json.Marshal(sd)
	// Check if error occurred during JSON marshaling
	if err != nil {
		return "Bahlil", fmt.Errorf("something error while Encoding SDP: %v", err)
	}
	// If no error, encode to base64
	return base64.StdEncoding.EncodeToString(b), nil
}

// Decode a base64 string to a SessionDescription
func decode(in string, sd *webrtc.SessionDescription) error {
	b, err := base64.StdEncoding.DecodeString(in)
	// Check if error occurred during base64 decoding
	if err != nil {
		return fmt.Errorf("error while decoding base64 SDP: %v", err)
	}
	// If no error, unmarshal JSON
	return json.Unmarshal(b, sd)
}

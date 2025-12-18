// This file contain Struct to define TrackHandler for Track Management (struct.go)

package webrtc

import "github.com/pion/rtp"

type TrackHandler struct {
	CurrTrack     int              // Current Active Track: 0 = Video, 1 = Screen
	TrackCount    int              // Total Track Count
	videoPackets  chan *rtp.Packet // Channel for Video RTP Packets
	screenPackets chan *rtp.Packet // Channel for Screen RTP Packets
	screenInUse   bool             // Flag to indicate if screen sharing is active
}

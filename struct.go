// This file contain Struct to define TrackHandler for Track Management (struct.go)

package main

import "github.com/pion/rtp"

type TrackHandler struct {
	CurrTrack  int
	TrackCount int
	Packets    chan *rtp.Packet
}

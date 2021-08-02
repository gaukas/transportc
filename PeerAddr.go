package transportc

import (
	"fmt"
	"net"
)

// Will be heavily rely on seed2sdp

// A net.Addr compliance struct
type PeerAddr struct {
	// WebRTCAddr
	NetworkType string // for now, UDP only
	IP          net.IP
	Port        uint16
}

func (a PeerAddr) Network() string {
	return a.NetworkType
}

func (a PeerAddr) String() string {
	return fmt.Sprintf("%s:%d", a.IP.String(), a.Port)
}

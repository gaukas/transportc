package transportc

import "github.com/pion/webrtc/v3"

// RFC 3264
type SDPRole = uint8

const (
	OFFERER SDPRole = iota
	ANSWERER
)

// RFC 4347
type DTLSRole = webrtc.DTLSRole

// From pion/webrtc
const (
	// DTLSRoleAuto defines the DTLS role is determined based on
	// the resolved ICE role: the ICE controlled role acts as the DTLS
	// client and the ICE controlling role acts as the DTLS server.
	DTLSRoleAuto DTLSRole = iota + 1

	// DTLSRoleClient defines the DTLS client role.
	DTLSRoleClient

	// DTLSRoleServer defines the DTLS server role.
	DTLSRoleServer
)

// type Status = uint8

// const (
// 	INITIALIZED Status = 1 << iota // Initialized.
// 	RW_READY
// 	CLOSED
// 	ERRORED
// 	LOCAL_SDP_AVAILABLE
// 	REMOTE_SDP_ACCEPTED

// 	NEW Status = 0 // No status, newly created.
// )

type NAT1To1IPs struct {
	IPs  []string
	Type webrtc.ICECandidateType
}

type PortRange struct {
	Min uint16
	Max uint16
}

type pionWrapper struct {
	peerConnection *webrtc.PeerConnection
	dataChannel    *webrtc.DataChannel
}

package transportc

import "github.com/pion/webrtc/v3"

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

// NAT1To1IPs consists of a slice of IP addresses and one single ICE Candidate Type.
// Use this struct to set the IPs to be used as ICE Candidates.
type NAT1To1IPs struct {
	IPs  []string
	Type webrtc.ICECandidateType
}

// PortRange specifies the range of ports to use for ICE Transports.
type PortRange struct {
	Min uint16
	Max uint16
}

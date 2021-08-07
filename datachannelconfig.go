package transportc

import "github.com/pion/webrtc/v3"

const (
	DataChannelBufferSizeDefault uint64 = 1024 * 1024 // Default Buffer Size: 1MB
	DataChannelBufferSizeMin     uint64 = 1024        // 1KB buffer could be too small...
)

type DataChannelConfig struct {
	Label       string // Name of DataChannel instance.
	SelfSDPType string // "offer", "answer"
	// PeerSDPType    string // "answer", "offer"
	SendBufferSize uint64 // send buffer max capacity. 0 for unlimited (at software level).

	// Optional
	IPAddr        []string
	CandidateType webrtc.ICECandidateType
	Port          uint16
}

func (dcc DataChannelConfig) PeerSDPType() string {
	if dcc.SelfSDPType == "offer" {
		return "answer"
	} else if dcc.SelfSDPType == "answer" {
		return "offer"
	} else {
		return "unknown"
	}
}

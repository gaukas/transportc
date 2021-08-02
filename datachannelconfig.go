package transportc

type DataChannelConfig struct {
	Label       string // Name of DataChannel instance.
	SelfSDPType string // "offer", "answer"
	// PeerSDPType    string // "answer", "offer"
	SendBufferSize uint64 // send buffer max capacity. 0 for unlimited (at software level).
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

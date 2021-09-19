package transportc

import "errors"

var (
	ErrDatachannelNotReady       = errors.New("transportc: data channel not ready")
	ErrDataChannelClosed         = errors.New("transportc: data channel already closed")
	ErrDataChannelAtCapacity     = errors.New("transportc: data channel at max capacity")
	ErrWebRTCUnsupportedNetwork  = errors.New("transportc: unsupported webrtconn network type")
	ErrWebRTConnReinit           = errors.New("transportc: double init")
	ErrWebRTConnReadIntegrity    = errors.New("transportc: webrtconn read integrity check failed")
	ErrWebRTConnOfferNotReceived = errors.New("transportc: offer not yet received by answerer")
)

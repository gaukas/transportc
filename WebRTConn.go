package wdt

// Will be heavily rely on seed2sdp

import (
	s2s "github.com/Gaukas/seed2sdp"
)

type WebRTConnRole uint8

const (
	OFFERER WebRTConnRole = iota
	ANSWERER
)

type WebRTConnStatus uint8

const (
	NEW                         WebRTConnStatus = 0 // NEWly created WebRTConn
	INIT                        WebRTConnStatus = 1
	READY                       WebRTConnStatus = 2 // Ready for send/recv
	CLOSED                      WebRTConnStatus = 4
	ERRORED                     WebRTConnStatus = 8
	LOCAL_DESCRIPTION_CREATED   WebRTConnStatus = 16
	REMOTE_DESCRIPTION_RECEIVED WebRTConnStatus = 32
)

type WebRTConn struct {
	dataChannel *s2s.DataChannel
	recvBuf     chan byte
	role        WebRTConnRole
	sendBuf     chan byte
	Status      WebRTConnStatus
}

// NewWebRTConn() creates a new WebRTConn instance and returns a pointer to it.
func NewWebRTConn(s2sconfig *s2s.DataChannelConfig) *WebRTConn {
	newDataChannel := s2s.DeclareDatachannel(s2sconfig)

	newRole := OFFERER
	if s2sconfig.SelfSDPType == "answer" {
		newRole = ANSWERER
	}

	return &WebRTConn{
		dataChannel: newDataChannel,
		recvBuf:     make(chan byte),
		role:        newRole,
		sendBuf:     make(chan byte),
		Status:      NEW,
	}
}

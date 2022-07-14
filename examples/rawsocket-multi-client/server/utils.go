package main

import (
	"sync"

	"github.com/Gaukas/transportc"
)

type SyncWebRTConn struct {
	Conn *transportc.WebRTConn
	Mux  *sync.Mutex
}

func NewSyncWebRTConn(conn *transportc.WebRTConn) *SyncWebRTConn {
	return &SyncWebRTConn{
		Conn: conn,
		Mux:  &sync.Mutex{},
	}
}

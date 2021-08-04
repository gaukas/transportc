package transportc

import (
	"net"
	"sync"
	"time"
)

// Will be heavily rely on seed2sdp

// A net.Conn compliance struct
type WebRTConn struct {
	// mutex
	lock *sync.RWMutex
	// states
	lasterr error
	role    WebRTCRole
	status  WebRTConnStatus

	// datachannel to net.Conn interface
	dataChannel *DataChannel
	recvBuf     *([]byte)
	// sendBuf     chan byte // Shouldn't be needed

	// net.Conn support, not meaningful at current phase
	localAddr  PeerAddr
	remoteAddr PeerAddr
}

func Dial(network, address string) (WebRTConn, error) {
	return WebRTConn{
		status: WebRTConnNew,
		localAddr: PeerAddr{
			NetworkType: "udp",
			IP:          net.ParseIP("0.0.0.0"),
			Port:        0,
		},
		remoteAddr: PeerAddr{
			NetworkType: "udp",
			IP:          net.ParseIP("0.0.0.0"),
			Port:        0,
		},
	}, nil
}

// Read() reads from recvBuf as the byte channel.
func (c WebRTConn) Read(b []byte) (n int, err error) {
	defer c.lock.Unlock()
	c.lock.Lock()
	n = len(*c.recvBuf)
	err = nil

	var i int
	for i = 0; i < n && i < len(b); i++ {
		nextbyte := (*c.recvBuf)[0]
		*c.recvBuf = (*c.recvBuf)[1:]
		// fmt.Printf("Byte read: %d\n", nextbyte)
		b[i] = nextbyte
	}
	// if i > 0 {
	// 	fmt.Printf("Returning i:%d", i)
	// }
	return i, err
}

// Write() send bytes over DataChannel.
func (c WebRTConn) Write(b []byte) (n int, err error) {
	// Won't implement timeout for now
	n = len(b)
	err = c.dataChannel.Send(b)
	if err != nil {
		return 0, nil
	}
	return n, err
}

func (c WebRTConn) Close() error {
	return c.dataChannel.Close()
}

func (c WebRTConn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c WebRTConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

// Unimplemented
func (c WebRTConn) SetDeadline(t time.Time) error {
	return nil
}

// Unimplemented
func (c WebRTConn) SetReadDeadline(t time.Time) error {
	return nil
}

// Unimplemented
func (c WebRTConn) SetWriteDeadline(t time.Time) error {
	return nil
}

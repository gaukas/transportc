package transportc

import (
	"context"
	"net"
	"sync"
	"time"
)

// *WebRTConn (vs. WebRTConn) implements net.Conn!
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

	readDeadline  time.Time
	writeDeadline time.Time
}

// Dial() creates the WebRTConn{} instance and assign a rwlock to it.
func Dial(_, _ string) (*WebRTConn, error) {
	return &WebRTConn{
		lock:   &sync.RWMutex{},
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

func (c *WebRTConn) _read(ctx context.Context, b []byte) (n int, err error) {
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

	if c.status&WebRTConnClosed > 0 {
		err = ErrDataChannelClosed
	} else if c.status&WebRTConnErrored > 0 {
		err = c.lasterr
		c.lasterr = nil
		c.status ^= WebRTConnErrored
	}

	return i, err
}

// Read() reads from recvBuf as the byte channel.
func (c *WebRTConn) Read(b []byte) (n int, err error) {
	if c.readDeadline.IsZero() {
		return c._read(context.Background(), b)
	} else {
		ctx, cancel := context.WithDeadline(context.Background(), c.readDeadline)
		defer cancel()
		return c._read(ctx, b)
	}
}

func (c *WebRTConn) _write(ctx context.Context, b []byte) (n int, err error) {
	n = len(b)
	err = c.dataChannel.Send(b)
	if err != nil {
		return 0, nil
	}
	return n, err
}

// Write() send bytes over DataChannel.
func (c *WebRTConn) Write(b []byte) (n int, err error) {
	if c.writeDeadline.IsZero() {
		return c._write(context.Background(), b)
	} else {
		ctx, cancel := context.WithDeadline(context.Background(), c.writeDeadline)
		defer cancel()
		return c._write(ctx, b)
	}
}

func (c *WebRTConn) Close() error {
	return c.dataChannel.Close()
}

func (c *WebRTConn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *WebRTConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *WebRTConn) SetDeadline(deadline time.Time) error {
	if deadline.Before(time.Now()) {
		return ErrDeadlinePast
	}
	c.readDeadline = deadline
	c.writeDeadline = deadline
	return nil
}

func (c *WebRTConn) SetReadDeadline(deadline time.Time) error {
	if deadline.Before(time.Now()) {
		return ErrDeadlinePast
	}
	c.readDeadline = deadline
	return nil
}

func (c *WebRTConn) SetWriteDeadline(deadline time.Time) error {
	if deadline.Before(time.Now()) {
		return ErrDeadlinePast
	}
	c.writeDeadline = deadline
	return nil
}

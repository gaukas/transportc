package transportc

import (
	"context"
	"errors"
	"net"
	"os"
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
	dataChannel    *DataChannel
	recvBuf        chan []byte
	unfinishedRecv []byte

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
	// defer c.lock.Unlock()
	// c.lock.Lock()
	var sizeReadMax int = len(b)

	if len(c.unfinishedRecv) == 0 {
		select {
		case <-ctx.Done():
			// if c.status&WebRTConnClosed > 0 {
			// 	return 0, ErrDataChannelClosed
			// } else if c.status&WebRTConnErrored > 0 {
			// 	err = c.lasterr
			// 	c.lasterr = nil
			// 	c.status &^= WebRTConnErrored
			// 	return 0, err
			// } else {
			return 0, ctx.Err()
			// }
		case c.unfinishedRecv = <-c.recvBuf:
			break
		}
	}

	for n = 0; n < sizeReadMax && len(c.unfinishedRecv) > 0; n++ {
		b[n] = c.unfinishedRecv[0]
		c.unfinishedRecv = c.unfinishedRecv[1:]
	}
	return n, nil
}

// Read() reads from recvBuf as the byte channel.
func (c *WebRTConn) Read(b []byte) (n int, err error) {
	if c.readDeadline.IsZero() {
		return c._read(context.Background(), b)
	} else {
		ctx, cancel := context.WithDeadline(context.Background(), c.readDeadline)
		defer cancel()
		n, err := c._read(ctx, b)
		if errors.Is(err, context.DeadlineExceeded) {
			return n, os.ErrDeadlineExceeded
		}
		return n, err
	}
}

func (c *WebRTConn) _write(ctx context.Context, b []byte) (n int, err error) {
	n = len(b)
	writeDone := make(chan error)

	go func() {
		err = c.dataChannel.Send(b)
		if err != nil {
			writeDone <- err
		}
		writeDone <- nil
	}()

	select {
	case err := <-writeDone:
		// fmt.Printf("Written: %d bytes\n", n)
		return n, err
	case <-ctx.Done():
		return 0, ctx.Err()
	}
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

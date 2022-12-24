package transportc

import (
	"context"
	"io"
	"net"
	"os"
	"sync/atomic"
	"time"
)

const (
	CONN_DEFAULT_MTU         = 65536
	CONN_IDLE_TIMEOUT        = 30 * time.Second
	CONN_DEFAULT_CONCURRENCY = 4
)

// Conn defines a connection based on a dedicated datachannel.
// Conn interfaces net.Conn.
type Conn struct {
	dataChannel io.ReadWriteCloser
	localAddr   net.Addr
	remoteAddr  net.Addr

	recvBuf    chan []byte // only readloop may write to or close this channel
	recvClosed atomic.Bool

	deadlineRd time.Time
	deadlineWr time.Time

	idle atomic.Bool
}

// BuildConningle builds a Conningle from an existing datachannel.
func NewConn(dataChannel io.ReadWriteCloser, maxConcurrency int) *Conn {
	return &Conn{
		dataChannel: dataChannel,
		recvBuf:     make(chan []byte, maxConcurrency),
	}
}

// Read reads data from the connection (underlying datachannel). It blocks until
// read deadline is reached, data is received in read buffer or error occurs.
func (c *Conn) Read(p []byte) (n int, err error) {
	if c.recvClosed.Load() {
		return 0, io.EOF
	}

	var ctxRead context.Context = context.Background()
	var cancelRead context.CancelFunc = func() {}
	if !c.deadlineRd.IsZero() {
		ctxRead, cancelRead = context.WithDeadline(ctxRead, c.deadlineRd)
	}
	defer cancelRead()

	// First select: check if anything readily available.
	select {
	case <-ctxRead.Done(): // if context is done, return error
		return 0, ctxRead.Err()
	case buf := <-c.recvBuf: // if anything is in the read buffer, read from it
		if buf == nil {
			return 0, io.EOF
		}
		n = copy(p, buf)
		if n < len(buf) {
			err = io.ErrShortBuffer
		}
		return n, err
	default: // nothing readily available, read from datachannel into recvBuf
		go func() {
			buf := make([]byte, CONN_DEFAULT_MTU)
			n, err := c.dataChannel.Read(buf)
			if err != nil {
				c.dataChannel.Close() // immediately close datachannel on error
				c.recvClosed.Store(true)
				close(c.recvBuf)
				return
			}
			if c.recvClosed.Load() {
				return
			}
			c.recvBuf <- buf[:n]
		}()
	}

	// Second select:
	select {
	case <-ctxRead.Done(): // if context is done, return error
		return 0, ctxRead.Err()
	case buf := <-c.recvBuf: // if anything is in the read buffer, read from it
		if buf == nil {
			return 0, io.EOF
		}
		n = copy(p, buf)
		if n < len(buf) {
			err = io.ErrShortBuffer
		}
		return n, err
	}
}

// Write writes data to the connection (underlying datachannel). It blocks until
// write deadline is reached, data is accepted by write buffer or error occurs.
func (c *Conn) Write(p []byte) (n int, err error) {
	if c.deadlineWr.IsZero() {
		n, err = c.dataChannel.Write(p)
		if err == nil || n > 0 {
			c.idle.Store(false)
		}
		return n, err
	}

	select {
	case <-time.After(time.Until(c.deadlineWr)):
		return 0, os.ErrDeadlineExceeded
	default:
		n, err = c.dataChannel.Write(p)
		if err == nil || n > 0 {
			c.idle.Store(false)
		}
		return n, err
	}
}

func (c *Conn) Close() error {
	return c.dataChannel.Close()
}

// LocalAddr returns the address of Local ICE Candidate
// selected for the datachannel
func (c *Conn) LocalAddr() net.Addr {
	return c.localAddr
}

// RemoteAddr returns the address of Remote ICE Candidate
// selected for the datachannel
func (c *Conn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

// SetDeadline sets the deadline for future Read and Write calls.
func (c *Conn) SetDeadline(t time.Time) error {
	c.deadlineRd = t
	c.deadlineWr = t
	return nil
}

// SetReadDeadline sets the deadline for future Read calls.
func (c *Conn) SetReadDeadline(t time.Time) error {
	c.deadlineRd = t
	return nil
}

// SetWriteDeadline sets the deadline for future Write calls.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	c.deadlineWr = t
	return nil
}

func (c *Conn) idleloop(t time.Duration) {
	if t == 0 {
		return // no idle timeout
	}

	for {
		if c.idle.Load() {
			c.Close()
			return
		}
		c.idle.Store(true)
		time.Sleep(t)
	}
}

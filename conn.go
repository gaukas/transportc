package transportc

import (
	"errors"
	"io"
	"net"
	"os"
	"time"

	"github.com/pion/datachannel"
	"golang.org/x/net/context"
)

// Conn is a net.Conn implementation for WebRTC DataChannels.
type Conn struct {
	dataChannel datachannel.ReadWriteCloser

	readDeadline      time.Time
	readMaxPacketSize int
	readBuf           chan []byte

	writeDeadline time.Time
}

// Read implements the net.Conn Read method.
func (c *Conn) Read(p []byte) (int, error) {
	if c.readDeadline.Before(time.Now()) && !c.readDeadline.IsZero() {
		return 0, os.ErrDeadlineExceeded
	}

	var ctx context.Context = context.Background()
	var cancel context.CancelFunc = func() {}
	if !c.readDeadline.IsZero() {
		ctx, cancel = context.WithDeadline(ctx, c.readDeadline)
	}
	defer cancel()

	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return 0, os.ErrDeadlineExceeded
		}
		return 0, ctx.Err()
	case b := <-c.readBuf:
		var err error = nil
		if len(b) > len(p) {
			err = io.ErrShortBuffer
		}
		return copy(p, b), err
	}
}

// Write implements the net.Conn Write method.
func (c *Conn) Write(p []byte) (int, error) {
	if c.writeDeadline.Before(time.Now()) && !c.writeDeadline.IsZero() {
		return 0, os.ErrDeadlineExceeded
	}
	return c.dataChannel.Write(p)
}

// Close implements the net.Conn Close method.
func (c *Conn) Close() error {
	return c.dataChannel.Close()
}

// LocalAddr implements the net.Conn LocalAddr method.
//
// It is hardcoded to return nil since WebRTC DataChannels are P2P
// and Local addresses are therefore trivial.
func (*Conn) LocalAddr() net.Addr {
	return nil
}

// RemoteAddr implements the net.Conn RemoteAddr method.
//
// It is hardcoded to return nil since WebRTC DataChannels are P2P
// and Remote addresses are therefore trivial.
func (*Conn) RemoteAddr() net.Addr {
	return nil
}

// SetDeadline implements the net.Conn SetDeadline method.
// It sets both read and write deadlines in a single call.
//
// See SetReadDeadline and SetWriteDeadline for the behavior of the deadlines.
func (c *Conn) SetDeadline(deadline time.Time) error {
	if deadline.Before(time.Now()) && !deadline.IsZero() {
		return errors.New("deadline is in the past")
	}
	c.readDeadline = deadline
	c.writeDeadline = deadline
	return nil
}

// SetReadDeadline sets the deadline for future Read calls.
//
// A ReadFrom call will fail and return os.ErrDeadlineExceeded
// before attempting to read from the buffer if the deadline has passed.
// And a ReadFrom call will block till no later than the set read deadline.
func (c *Conn) SetReadDeadline(deadline time.Time) error {
	if deadline.Before(time.Now()) && !deadline.IsZero() {
		return errors.New("deadline is in the past")
	}
	c.readDeadline = deadline
	return nil
}

// SetWriteDeadline sets the deadline for future Write calls.
//
// A WriteTo call will fail and return os.ErrDeadlineExceeded
// before attempting to write to the buffer if the deadline has passed.
// Otherwise the set write deadline will not affect the WriteTo call.
func (c *Conn) SetWriteDeadline(deadline time.Time) error {
	if deadline.Before(time.Now()) && !deadline.IsZero() {
		return errors.New("deadline is in the past")
	}
	c.writeDeadline = deadline
	return nil
}

// readLoop reads from the underlying DataChannel and writes to the readBuf channel.
//
// Start running in a goroutine once the datachannel is opened.
func (c *Conn) readLoop() {
	for {
		b := make([]byte, c.readMaxPacketSize)
		n, err := c.dataChannel.Read(b)
		if err != nil {
			break // Conn failed or closed
		}
		c.readBuf <- b[:n]
	}
}

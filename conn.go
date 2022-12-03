package transportc

import (
	"context"
	"errors"
	"io"
	"math"
	"net"
	"os"
	"time"

	"github.com/pion/datachannel"
)

// Conn is a net.Conn implementation for WebRTC DataChannels.
type Conn struct {
	dataChannel datachannel.ReadWriteCloser

	mtu          int // Max Transmission Unit for both recv and send
	readDeadline time.Time
	readBuf      chan []byte

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
// If the size of the buffer is greater than the MTU,
// the data could still be sent but will be fragmented (not recommended).
func (c *Conn) Write(p []byte) (int, error) {
	// count length of p as uint16 (65535 bytes max per message)
	var n uint32 = uint32(len(p))
	if n > math.MaxUint16 {
		return 0, errors.New("message too long, max 65535 bytes")
	}
	// build write buffer
	var wrBuf []byte = make([]byte, n+2) // 2 bytes for length, n bytes for data
	wrBuf[0] = byte(n >> 8)              // first byte of length (most significant)
	wrBuf[1] = byte(n)                   // second byte of length (least significant)
	copy(wrBuf[2:], p)                   // copy data to buffer

	if c.writeDeadline.IsZero() || c.writeDeadline.After(time.Now()) {
		var writtenTotal int
		// split into multiple packets then write
		for len(wrBuf) > c.mtu {
			var err error
			written, err := c.dataChannel.Write(wrBuf[:c.mtu])
			if err != nil {
				return writtenTotal - 2, err
			}
			writtenTotal += written
			wrBuf = wrBuf[c.mtu:]
		}

		// write the remaining bytes
		written, err := c.dataChannel.Write(wrBuf)
		writtenTotal += written

		return writtenTotal - 2, err
	}

	return 0, os.ErrDeadlineExceeded
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
// A Read call will fail and return os.ErrDeadlineExceeded
// before attempting to read from the buffer if the deadline has passed.
// And a Read call will block till no later than the set read deadline.
func (c *Conn) SetReadDeadline(deadline time.Time) error {
	if deadline.Before(time.Now()) && !deadline.IsZero() {
		return errors.New("deadline is in the past")
	}
	c.readDeadline = deadline
	return nil
}

// SetWriteDeadline sets the deadline for future Write calls.
//
// A Write call will fail and return os.ErrDeadlineExceeded
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
READLOOP:
	for {
		b := make([]byte, c.mtu)
		n, err := c.dataChannel.Read(b)
		if err != nil {
			break READLOOP // Conn failed or closed
		}

		// read length of message
		var msgLen uint16 = uint16(b[0])<<8 | uint16(b[1])
		// create read buffer
		var rdBuf []byte = make([]byte, 0)
		// copy data to read buffer
		rdBuf = append(rdBuf, b[2:n]...)
		// read remaining data
		for len(rdBuf) < int(msgLen) {
			n, err := c.dataChannel.Read(b)
			if err != nil {
				break READLOOP // Conn failed or closed
			}
			rdBuf = append(rdBuf, b[:n]...)
		}
		c.readBuf <- rdBuf[:msgLen]
	}
}

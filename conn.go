package transportc

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
)

// Conn implements the net.Conn interface.
type Conn struct {
	// config
	config *Config

	// lock protects recvBuf
	bufLock *sync.Mutex
	recvBuf []byte

	rDeadline time.Time
	wDeadline time.Time

	// pion/webrtc wrapper
	pion pionWrapper
}

// connWithConfig DOES NOT start the handshake!
func connWithConfig(config *Config) *Conn {
	return &Conn{
		config:  config,
		bufLock: &sync.Mutex{},
		recvBuf: make([]byte, 0),
		pion:    pionWrapper{},
	}
}

// Read implements the net.Conn Read method.
func (c *Conn) Read(b []byte) (n int, err error) {
	for c.rDeadline.After(time.Now()) || c.rDeadline.IsZero() {
		c.bufLock.Lock()
		if len(c.recvBuf) > 0 {
			n = copy(b, c.recvBuf)
			c.recvBuf = c.recvBuf[n:]
			c.bufLock.Unlock()
			return n, nil
		}
		c.bufLock.Unlock()
	}
	return 0, errors.New("read deadline exceeded")
}

// Write implements the net.Conn Write method.
func (c *Conn) Write(b []byte) (n int, err error) {
	for c.wDeadline.After(time.Now()) || c.wDeadline.IsZero() {
		if c.pion.dataChannel == nil {
			return 0, errors.New("data channel not found")
		}
		err := c.pion.dataChannel.Send(b)
		if err == nil {
			return len(b), nil
		} else {
			return 0, err
		}
	}
	return 0, errors.New("write deadline exceeded")
}

func (c *Conn) Close() error {
	c.bufLock.Lock()
	defer c.bufLock.Unlock()

	err := c.pion.dataChannel.Close()
	if err != nil {
		return err
	}
	return c.pion.peerConnection.Close()
}

func (c *Conn) LocalAddr() net.Addr {
	return LocalDummyAddr()
}

func (c *Conn) RemoteAddr() net.Addr {
	return RemoteDummyAddr()
}

func (c *Conn) SetDeadline(deadline time.Time) error {
	if deadline.Before(time.Now()) && !deadline.IsZero() {
		return errors.New("deadline is in the past")
	}
	c.rDeadline = deadline
	c.wDeadline = deadline
	return nil
}

func (c *Conn) SetReadDeadline(deadline time.Time) error {
	if deadline.Before(time.Now()) && !deadline.IsZero() {
		return errors.New("deadline is in the past")
	}
	c.rDeadline = deadline
	return nil
}

func (c *Conn) SetWriteDeadline(deadline time.Time) error {
	if deadline.Before(time.Now()) && !deadline.IsZero() {
		return errors.New("deadline is in the past")
	}
	c.wDeadline = deadline
	return nil
}

// CreateLocalDescription creates a local session description.
// It will block until ICE gathering is complete.
//
// Need to be called manually when using no SignalMethod.
func (c *Conn) CreateLocalDescription() error {
	var localDescription webrtc.SessionDescription
	var err error
	if c.config.SDPRole == OFFERER {
		localDescription, err = c.pion.peerConnection.CreateOffer(nil)
	} else if c.config.SDPRole == ANSWERER {
		localDescription, err = c.pion.peerConnection.CreateAnswer(nil)
	} else {
		err = errors.New("unknown SDP role")
	}

	if err != nil {
		return err
	}

	// Create channel that is blocked until ICE Gathering is complete
	gatherComplete := webrtc.GatheringCompletePromise(c.pion.peerConnection)

	// Sets the LocalDescription, and starts our UDP listeners
	err = c.pion.peerConnection.SetLocalDescription(localDescription)
	if err != nil {
		return err
	}

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	// TODO: use OnICECandidate callback instead
	<-gatherComplete

	return nil
}

func (c *Conn) GetLocalDescription() *webrtc.SessionDescription {
	return c.pion.peerConnection.LocalDescription()
}

func (c *Conn) SetRemoteDescription(remoteSDP *webrtc.SessionDescription) error {
	return c.pion.peerConnection.SetRemoteDescription(*remoteSDP)
}

func (c *Conn) setEventHandler(dataChannelStatus chan bool) {
	c.pion.peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		if connectionState > webrtc.ICEConnectionStateConnected {
			c.pion.peerConnection.Close() // TODO: should this be done?
		}
	})

	c.pion.dataChannel.OnOpen(func() {
		dataChannelStatus <- true
	})

	c.pion.dataChannel.OnClose(func() {
		dataChannelStatus <- false
	})

	c.pion.dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		// append to conn.recvBuf
		c.bufLock.Lock()
		c.recvBuf = append(c.recvBuf, msg.Data...)
		c.bufLock.Unlock()
	})

	c.pion.dataChannel.OnError(func(err error) {
		// TODO: Do something? Deal with it? Tear it down?
	})
}

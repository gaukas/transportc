package transportc

import (
	"encoding/json"
	"sync"

	"github.com/pion/webrtc/v3"
)

func (c *WebRTConn) Status() WebRTConnStatus {
	defer c.lock.RUnlock()
	c.lock.RLock()
	status := c.status
	return status
}

func (c *WebRTConn) LastError() error {
	defer c.lock.RUnlock()
	c.lock.RLock()
	err := c.lasterr
	return err
}

// setEvtHandler() can't be called until c.dataChannel.WebRTCDataChannel has been pointed to a valid webrtc.DataChannel
func (c *WebRTConn) setDataChannelEvtHandler() {
	c.dataChannel.WebRTCDataChannel.OnOpen(func() {
		// fmt.Printf("[Info] Successfully opened Data Channel '%s'-'%d'. \n", dataChannel.WebRTCDataChannel.Label(), dataChannel.WebRTCDataChannel.ID())
		defer c.lock.Unlock()
		c.lock.Lock()
		c.status |= WebRTConnReady
	})

	c.dataChannel.WebRTCDataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		for _, b := range msg.Data {
			c.recvBuf <- b // all into channel, assuming Thread-Safe
		}
		// fmt.Printf("[Comm] %s: '%s'\n", dataChannel.WebRTCDataChannel.Label(), string(msg.Data))
	})

	c.dataChannel.WebRTCDataChannel.OnClose(func() {
		defer c.lock.Unlock()
		c.lock.Lock()
		c.status |= WebRTConnClosed
		// fmt.Printf("[Warning] Data Channel %s closed\n", dataChannel.WebRTCDataChannel.Label())
		// fmt.Printf("[Info] Tearing down Peer Connection\n")
		c.dataChannel.WebRTCPeerConnection.Close()
	})

	c.dataChannel.WebRTCDataChannel.OnError(func(err error) {
		defer c.lock.Unlock()
		c.lock.Lock()
		c.lasterr = err
		c.status |= WebRTConnErrored
		// fmt.Printf("[Fatal] Data Channel %s errored: %v\n", dataChannel.WebRTCDataChannel.Label(), err)
		// fmt.Printf("[Info] Tearing down Peer Connection\n")
		c.dataChannel.WebRTCPeerConnection.Close()
	})
}

// Init() setup the underlying datachannel with everything defined in c.dataChannel.
// once this function returns, a remote description should be fed in as soon as possible.
func (c *WebRTConn) Init(dcconfig *DataChannelConfig, pionSettingEngine webrtc.SettingEngine, pionConfiguration webrtc.Configuration) error {
	if c.status != WebRTConnNew {
		return ErrWebRTConnReinit
	}

	c.lock = &sync.RWMutex{}
	c.lasterr = nil
	if dcconfig.SelfSDPType == "answer" {
		c.role = ANSWERER
	} else {
		c.role = OFFERER
	}

	c.dataChannel = DeclareDatachannel(dcconfig, pionSettingEngine, pionConfiguration)
	c.recvBuf = make(chan byte)

	// defer c.lock.Unlock()
	// c.lock.Lock()
	err := c.dataChannel.Initialize()
	if err != nil {
		return err
	}

	c.dataChannel.WebRTCPeerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		if connectionState >= webrtc.ICEConnectionStateCompleted {
			defer c.lock.Unlock()
			c.lock.Lock()
			// ICEConnectionStateCompleted, ICEConnectionStateDisconnected, ICEConnectionStateFailed, ICEConnectionStateClosed
			c.status |= WebRTConnClosed
		}
	})

	// Create LocalSDP(offer) for Offerer. Answerer shall wait for an offer before generating LocalSDP(answer)
	if c.role == OFFERER {
		c.setDataChannelEvtHandler() // Offerer, safe to set datachannel event handler before creating descriptions.
		err = c.dataChannel.CreateLocalDescription()
		if err != nil {
			return err
		}
		c.status |= WebRTConnLocalSDPReady
	} else if c.role == ANSWERER {
		// Answere, shall wait for offerer to create the channel.
		c.dataChannel.WebRTCPeerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
			c.dataChannel.WebRTCDataChannel = d
			c.setDataChannelEvtHandler()
		})
	}
	c.status |= WebRTConnInit
	return nil
}

func (c *WebRTConn) LocalSDP() *webrtc.SessionDescription {
	return c.dataChannel.GetLocalDescription()
}

func (c *WebRTConn) LocalSDPJsonString() (string, error) {
	sdp, err := json.Marshal(c.LocalSDP())
	if err != nil {
		return "", err
	}
	return string(sdp), nil
}

func (c *WebRTConn) SetRemoteSDP(sdp *webrtc.SessionDescription) error {
	return c.dataChannel.SetRemoteDescription(sdp)
}

func (c *WebRTConn) SetRemoteSDPJsonString(sdp string) error {
	rdesc := webrtc.SessionDescription{}
	err := json.Unmarshal([]byte(sdp), &rdesc)
	if err != nil {
		return err
	}
	return c.SetRemoteSDP(&rdesc)
}

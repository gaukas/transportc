package transportc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gaukas/logging"
	"github.com/pion/datachannel"
	"github.com/pion/webrtc/v3"
)

// Dialer can dial a remote peer.
//
// If the SignalMethod is set, the Offer/Answer exchange per new PeerConnection will be done automatically.
type Dialer struct {
	logger  logging.Logger
	signal  Signal
	timeout time.Duration

	// WebRTC configuration
	settingEngine webrtc.SettingEngine
	configuration webrtc.Configuration

	// WebRTC PeerConnection
	mutex               sync.Mutex // mutex makes peerConnection thread-safe
	peerConnection      *webrtc.PeerConnection
	reusePeerConnection bool
}

var (
	ErrBrokenDialer = errors.New("dialer need to be recreated")
)

// Dial connects to a remote peer with SDP-based negotiation.
//
// Internally calls DialContext with context.Background().
//
// The returned connection is backed by a DataChannel created by the caller
// with the SDP role as OFFERER as defined in RFC3264. If SignalMethod is set,
// the Offer/Answer exchange per new PeerConnection will be done automatically.
// Otherwise, it is recommended to call NewPeerConnection and exchange the SDP
// offer/answer manually before dialing.
func (d *Dialer) Dial(label string) (net.Conn, error) {
	return d.DialContext(context.Background(), label)
}

// DialContext connects to a remote peer with SDP-based negotiation
// using the provided context.
//
// The provided Context must be non-nil. If the context expires before
// the connection is complete, an error is returned. Once successfully
// connected, any expiration of the context will not affect the
// connection.
//
// The returned connection is backed by a DataChannel created by the caller
// with the SDP role as OFFERER as defined in RFC3264. If SignalMethod is set,
// the Offer/Answer exchange per new PeerConnection will be done automatically.
// Otherwise, it is recommended to call NewPeerConnection and exchange the SDP
// offer/answer manually before dialing.
func (d *Dialer) DialContext(ctx context.Context, label string) (net.Conn, error) {
	// check if context is done
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	dataChannel, err := d.nextDataChannel(ctx, label)
	if err != nil {
		return nil, err
	}

	conn := NewConn(nil, CONN_DEFAULT_CONCURRENCY)

	// set event handlers
	var detachChan chan datachannel.ReadWriteCloser = make(chan datachannel.ReadWriteCloser)
	dataChannel.OnOpen(func() {
		// detach from wrapper
		dc, err := dataChannel.Detach()
		if err != nil {
			close(detachChan)
		} else {
			detachChan <- dc
			close(detachChan)
		}
	})

	dataChannel.OnClose(func() {
		// TODO: possibly tear down the PeerConnection if it is the last DataChannel?
		conn.Close()
	})

	// OnError won't be used as pion's readLoop is ignored
	// dataChannel.OnError(func(err error) {
	// })

	// wait for datachannel
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case dataChannelDetach := <-detachChan:
		if dataChannelDetach == nil {
			return nil, errors.New("failed to receive datachannel")
		}
		conn.dataChannel = dataChannelDetach

		// Set LocalAddr and RemoteAddr
		if sctp := d.peerConnection.SCTP(); sctp != nil {
			if dtls := sctp.Transport(); dtls != nil {
				if ice := dtls.ICETransport(); ice != nil {
					icePair, err := ice.GetSelectedCandidatePair()
					if err != nil {
						return nil, fmt.Errorf("dialer: failed to get selected ICE Candidate pair: %w", err)
					}
					conn.localAddr = &Addr{
						Hostname: icePair.Local.Address,
						Port:     icePair.Local.Port,
					}
					conn.remoteAddr = &Addr{
						Hostname: icePair.Remote.Address,
						Port:     icePair.Remote.Port,
					}
				}
			}
		}
		go conn.idleloop(d.timeout) // start the read loop

		return conn, nil
	}
}

// Close closes the WebRTC PeerConnection and with it
// all the WebRTC DataChannels under it.
//
// SHOULD be called when done using the transport.
func (d *Dialer) Close() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.peerConnection != nil {
		return d.peerConnection.Close()
	}
	return nil
}

func (d *Dialer) nextDataChannel(ctx context.Context, label string) (*webrtc.DataChannel, error) {
	if d.peerConnection == nil || !d.reusePeerConnection {
		dc, err := d.startPeerConnection(ctx, label)
		if err != nil {
			return nil, err
		}
		return dc, nil
	}

	// try getting a new data channel from the existing peer connection
	dataChannel, err := d.peerConnection.CreateDataChannel(label, nil)
	if err != nil {
		// error: retry after getting a new peer connection.
		// if errors.Is(err, webrtc.ErrConnectionClosed) {
		d.peerConnection.Close()
		d.peerConnection = nil
		dataChannel, err = d.startPeerConnection(ctx, label)
		if err != nil {
			return nil, err
		}
		// } else { // error but not due to PC closed
		// 	return nil, err
		// }
	}
	return dataChannel, nil
}

// startPeerConnection creates a new PeerConnection that can be reused in following Dial calls.
// If Dialer.signal is set, the Offer/Answer exchange will be done automatically.
//
// It returns the first DataChannel created with the PeerConnection. Note: the returned DataChannel
// is not guaranteed to be open yet.It is caller's responsibility to check the DataChannel's state
// and handle the OnOpen event.
//
// Not thread-safe. Caller MUST hold the mutex before calling this function.
func (d *Dialer) startPeerConnection(ctx context.Context, dataChannelLabel string) (*webrtc.DataChannel, error) {
	api := webrtc.NewAPI(webrtc.WithSettingEngine(d.settingEngine))

	peerConnection, err := api.NewPeerConnection(d.configuration)
	if err != nil {
		return nil, err
	} else if peerConnection == nil {
		return nil, errors.New("dialer: created nil PeerConnection")
	}

	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		// TODO: handle this better
		if s > webrtc.PeerConnectionStateConnected {
			d.logger.Warnf("dialer: PeerConnection disconnected.")
			d.mutex.Lock()
			peerConnection.Close()
			if d.peerConnection == peerConnection {
				d.peerConnection = nil
			}
			d.mutex.Unlock()
		}
	})

	d.peerConnection = peerConnection

	dataChannel, err := d.peerConnection.CreateDataChannel(dataChannelLabel, nil)
	if err != nil {
		return nil, err
	}

	// Automatic Signalling when possible
	if d.signal != nil {
		offerID, err := d.SendOffer(ctx)
		if err != nil {
			return nil, fmt.Errorf("dialer: failed to send offer: %w", err)
		}

		err = d.SetAnswer(ctx, offerID)
		if err != nil {
			return nil, fmt.Errorf("dialer: failed to set answer: %w", err)
		}
	}

	return dataChannel, nil
}

// SendOffer creates a local offer and sets it as the local description,
// then signals the offer to the remote peer and return the offer ID.
//
// Automatically called by startPeerConnection when Dialer.signal is set.
func (d *Dialer) SendOffer(ctx context.Context) (uint64, error) {
	localDescription, err := d.peerConnection.CreateOffer(nil)
	if err != nil {
		return 0, fmt.Errorf("dialer: failed to create local offer: %w", err)
	}

	// Create channel that is blocked until ICE Gathering is complete
	gatherComplete := webrtc.GatheringCompletePromise(d.peerConnection)

	// Sets the LocalDescription, and starts our UDP listeners
	err = d.peerConnection.SetLocalDescription(localDescription)
	if err != nil {
		return 0, fmt.Errorf("dialer: failed to set local description: %w", err)
	}

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	// TODO: use OnICECandidate callback instead
	select {
	case <-ctx.Done():
		return 0, fmt.Errorf("dialer: context done before ICE gathering complete: %w", ctx.Err())
	case <-gatherComplete:
		offer := d.peerConnection.LocalDescription()
		offerByte, err := json.Marshal(offer)
		if err != nil {
			return 0, fmt.Errorf("dialer: failed to marshal local offer: %w", err)
		}

		offerID, err := d.signal.Offer(offerByte)
		if err != nil {
			return 0, fmt.Errorf("dialer: failed to signal local offer: %w", err)
		}

		return offerID, nil
	}
}

// SetAnswer reads the answer from the signaler and sets it as the remote description.
//
// Automatically called by startPeerConnection when Dialer.signal is set.
func (d *Dialer) SetAnswer(ctx context.Context, offerID uint64) error {
	var blockingChan chan error = make(chan error)
	var answerUnmarshal webrtc.SessionDescription

	go func(blockingChan chan error, webrtcAnswer *webrtc.SessionDescription) {
		defer close(blockingChan)
		answerBytes, err := d.signal.ReadAnswer(offerID)
		for err == ErrAnswerNotReady {
			time.Sleep(100 * time.Millisecond)
			answerBytes, err = d.signal.ReadAnswer(offerID)
		}

		if err != nil {
			blockingChan <- fmt.Errorf("dialer: failed to read answer: %w", err)
			return
		}

		err = json.Unmarshal(answerBytes, webrtcAnswer)
		if err != nil {
			blockingChan <- fmt.Errorf("dialer: failed to unmarshal answer: %w", err)
			return
		}
	}(blockingChan, &answerUnmarshal)

	select {
	case <-ctx.Done():
		return fmt.Errorf("dialer: context done before answer received: %w", ctx.Err())
	case remoteErr := <-blockingChan:
		if remoteErr != nil {
			return remoteErr
		}
	}
	err := d.peerConnection.SetRemoteDescription(answerUnmarshal)
	if err != nil {
		return fmt.Errorf("dialer: failed to set remote description: %w", err)
	}

	return nil
}

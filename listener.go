package transportc

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gaukas/logging"
	"github.com/gaukas/transportc/internal/utils"
	"github.com/pion/webrtc/v3"
)

type ListenerRunningStatus = uint32

const (
	LISTENER_NEW ListenerRunningStatus = iota
	LISTENER_RUNNING
	LISTENER_SUSPENDED
	LISTENER_STOPPED
)

const (
	DEFAULT_ACCEPT_TIMEOUT = 10 * time.Second
)

// Listener listens for new PeerConnections and saves all incoming datachannel from peers for later use.
type Listener struct {
	logger  logging.Logger
	signal  Signal
	timeout time.Duration

	runningStatus ListenerRunningStatus // Initialized at creation. Atomic. Access via sync/atomic methods only

	// WebRTC configuration
	settingEngine webrtc.SettingEngine
	configuration webrtc.Configuration

	// WebRTC PeerConnection
	mutex           sync.Mutex                        // mutex makes peerConnection thread-safe
	peerConnections map[uint64]*webrtc.PeerConnection // PCID:PeerConnection pair

	// chan Conn for Accept
	conns chan net.Conn // Initialized at creation
}

// Accept accepts a new connection from the listener.
//
// It does not establish new connections.
// These connections are from the pool filled automatically by acceptLoop.
func (l *Listener) Accept() (net.Conn, error) {
	// read next from conns
	conn := <-l.conns
	if conn == nil {
		return nil, errors.New("closed listener can't accept new connections")
	}
	return conn, nil
}

func (l *Listener) Start() error {
	if atomic.CompareAndSwapUint32(&l.runningStatus, LISTENER_NEW, LISTENER_RUNNING) || atomic.CompareAndSwapUint32(&l.runningStatus, LISTENER_SUSPENDED, LISTENER_RUNNING) || atomic.CompareAndSwapUint32(&l.runningStatus, LISTENER_STOPPED, LISTENER_RUNNING) {
		l.startAcceptLoop()
		return nil
	}
	return errors.New("listener already started")
}

// Stop the listener. Close existing PeerConnections.
//
// The listener can be stopped when it is running or suspended.
func (l *Listener) Stop() error {
	if atomic.CompareAndSwapUint32(&l.runningStatus, LISTENER_RUNNING, LISTENER_STOPPED) || atomic.CompareAndSwapUint32(&l.runningStatus, LISTENER_SUSPENDED, LISTENER_STOPPED) {
		l.mutex.Lock()
		defer l.mutex.Unlock()
		for _, pc := range l.peerConnections {
			pc.Close()
		}
		l.peerConnections = make(map[uint64]*webrtc.PeerConnection) // clear map

		return nil
	}
	return errors.New("listener already stopped")
}

// Suspend the listener. Don't close existing PeerConnections.
func (l *Listener) Suspend() error {
	if atomic.CompareAndSwapUint32(&l.runningStatus, LISTENER_RUNNING, LISTENER_SUSPENDED) {
		return nil
	}
	return errors.New("listener not in running state")
}

// startAcceptLoop() should be called before the first Accept() call.
func (l *Listener) startAcceptLoop() {
	if l.signal == nil {
		return // nothing to do for manual signaling (nil)
	}

	if l.timeout == 0 {
		l.timeout = DEFAULT_ACCEPT_TIMEOUT
	}

	// Loop: accept new Offers from signal and establish new PeerConnections
	go func() {
		for atomic.LoadUint32(&l.runningStatus) != LISTENER_STOPPED { // Don't return unless STOPPED
			for atomic.LoadUint32(&l.runningStatus) == LISTENER_RUNNING { // Only accept new Offers if RUNNING
				// Accept new Offer from signal
				offerID, offer, err := l.signal.ReadOffer()
				if err != nil {
					continue
				}
				// Create new PeerConnection in a goroutine
				go func() {
					ctxTimeout, cancel := context.WithTimeout(context.Background(), l.timeout)
					defer cancel()
					err := l.nextPeerConnection(ctxTimeout, offerID, offer)
					if err != nil {
						return // ignore errors
					}
				}()
			}
			// sleep for a little while if new/suspended
			time.Sleep(time.Second)
		}
	}()
}

func (l *Listener) nextPeerConnection(ctx context.Context, offerID uint64, offer []byte) error {
	api := webrtc.NewAPI(webrtc.WithSettingEngine(l.settingEngine))

	peerConnection, err := api.NewPeerConnection(l.configuration)
	if err != nil {
		return err
	}

	pcwg := &sync.WaitGroup{}

	// Get a random ID
	id := l.nextPCID()
	l.mutex.Lock()
	l.peerConnections[id] = peerConnection
	l.mutex.Unlock()

	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		// TODO: handle this better
		if s == webrtc.PeerConnectionStateFailed || s == webrtc.PeerConnectionStateClosed || s == webrtc.PeerConnectionStateDisconnected {
			l.mutex.Lock()
			peerConnection.Close()
			delete(l.peerConnections, id)
			l.logger.Warnf("User session closed, %d active sessions remain", len(l.peerConnections))
			l.mutex.Unlock()
		} else if s == webrtc.PeerConnectionStateConnected {
			l.mutex.Lock()
			l.logger.Warnf("User session created, %d active sessions in total", len(l.peerConnections))
			l.mutex.Unlock()
			go utils.DelayedExecution(l.timeout, func() {
				pcwg.Wait()
				l.mutex.Lock()
				peerConnection.Close()
				l.logger.Warnf("Closing user session due to idle... ")
				delete(l.peerConnections, id)
				l.mutex.Unlock()
			})
		}
	})

	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		conn := NewConnD(nil, CONN_D_MAX_CONC)

		d.OnOpen(func() {
			// detach from wrapper
			dc, err := d.Detach()
			if err != nil {
				return
			} else {
				conn.dataChannel = dc

				// Set LocalAddr and RemoteAddr
				if sctp := peerConnection.SCTP(); sctp != nil {
					if dtls := sctp.Transport(); dtls != nil {
						if ice := dtls.ICETransport(); ice != nil {
							icePair, err := ice.GetSelectedCandidatePair()
							if err != nil {
								return
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

				go conn.idleloop(l.timeout)
				pcwg.Add(1)
				l.conns <- conn
			}
		})

		d.OnClose(func() {
			// TODO: possibly tear down the PeerConnection if it is the last DataChannel?
			conn.Close()
			pcwg.Done()
		})
	})

	var bChan chan bool = make(chan bool)

	offerUnmarshal := webrtc.SessionDescription{}
	err = json.Unmarshal(offer, &offerUnmarshal)
	if err != nil {
		return err
	}

	err = peerConnection.SetRemoteDescription(offerUnmarshal)
	if err != nil {
		return err
	}

	// wait for local answer
	go func(blockingChan chan bool) {
		localDescription, err := peerConnection.CreateAnswer(nil)
		if err != nil {
			blockingChan <- false
		}
		// Create channel that is blocked until ICE Gathering is complete
		gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

		// Sets the LocalDescription, and starts our UDP listeners
		err = peerConnection.SetLocalDescription(localDescription)
		if err != nil {
			blockingChan <- false
		}
		<-gatherComplete
		blockingChan <- true
	}(bChan)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case status := <-bChan:
		if !status {
			return errors.New("failed to create local answer")
		}
		answer := peerConnection.LocalDescription()
		// answer to JSON bytes
		answerBytes, err := json.Marshal(answer)
		if err != nil {
			return err
		}
		err = l.signal.Answer(offerID, answerBytes)
		if err != nil {
			return err
		}
	}

	return nil
}

// randomize a uint64 for ID. Must not conflict with existing IDs.
func (l *Listener) nextPCID() uint64 {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	var id uint64
	for {
		id = rand.Uint64()                       // skipcq: GSC-G404
		if _, ok := l.peerConnections[id]; !ok { // not found
			break // okay to use this ID
		}
	}
	return id
}

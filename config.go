package transportc

import (
	"context"
	"encoding/json"
	"errors"
	"net"

	"github.com/pion/ice/v2"
	"github.com/pion/webrtc/v3"
)

type Config struct {
	/**** MANDATORY FIELDS ****/
	// Label of the DataChannel instance to be created.
	Label string

	// SDPRole of the DataChannel instance to be created.
	// MUST be either OFFERER or ANSWERER, as defined in RFC3264
	SDPRole SDPRole

	// DTLSRole
	// MUST be either DTLSRoleClient or DTLSRoleServer, as defined in RFC4347
	// DTLSRoleClient will send the ClientHello and start the handshake.
	// DTLSRoleServer will wait for the ClientHello.
	AnsweringDTLSRole DTLSRole

	/**** OPTIONAL FIELDS ****/
	// SignalMethod offers the automatic signaling when establishing the DataChannel.
	SignalMethod SignalMethod

	// IPs includes a slice of IP addresses and one single ICE Candidate Type.
	// If set, will add these IPs as ICE Candidates
	IPs *NAT1To1IPs

	// PortRange is the range of ports to use for the DataChannel.
	PortRange *PortRange

	// UDPMux allows serving multiple DataChannels over the one or more pre-established UDP socket.
	UDPMux ice.UDPMux

	// CandidateNetworkTypes restricts ICE agent to gather
	// only selected types of ICE candidates.
	CandidateNetworkTypes []webrtc.NetworkType

	// InterfaceFilter restricts ICE agent to gather ICE candidates
	// on only selected interfaces.
	InterfaceFilter func(name string) (gatherPermitted bool)

	// pion structs
	// - SHOULD not be exported, to prevent unintentional use.
	// - When CONFLICT with other DataChannelConfig fields, will be overridden

	pionSettingEngine webrtc.SettingEngine
	pionConfiguration webrtc.Configuration
}

// ConfigWithPionFields is the only way to set the pion structs.
// It is used to guarantee that other fields prevail over the pion structs.
func ConfigWithPionFields(settingEngine webrtc.SettingEngine, configuration webrtc.Configuration) *Config {
	return &Config{
		pionSettingEngine: settingEngine,
		pionConfiguration: configuration,
	}
}

func (c *Config) Dial() (net.Conn, error) {
	return c.DialContext(context.Background())
}

func (c *Config) DialContext(ctx context.Context) (net.Conn, error) {
	var conn *Conn = connWithConfig(c)
	var err error
	var dataChannelStatus chan bool = make(chan bool) // use to block when waiting for new datachannel vs. ctx.Done()

	if c.SDPRole == ANSWERER {
		err = c.pionSettingEngine.SetAnsweringDTLSRole(c.AnsweringDTLSRole)
		if err != nil {
			return nil, err
		}
	}

	if c.IPs != nil {
		c.pionSettingEngine.SetNAT1To1IPs(c.IPs.IPs, c.IPs.Type)
	}

	if c.PortRange != nil {
		err = c.pionSettingEngine.SetEphemeralUDPPortRange(c.PortRange.Min, c.PortRange.Max)
		if err != nil {
			return nil, err
		}
	}

	if c.UDPMux != nil {
		c.pionSettingEngine.SetICEUDPMux(c.UDPMux)
	}

	if c.CandidateNetworkTypes != nil {
		c.pionSettingEngine.SetNetworkTypes(c.CandidateNetworkTypes)
	}

	if c.InterfaceFilter != nil {
		c.pionSettingEngine.SetInterfaceFilter(c.InterfaceFilter)
	}

	api := webrtc.NewAPI(webrtc.WithSettingEngine(c.pionSettingEngine))
	conn.pion.peerConnection, err = api.NewPeerConnection(c.pionConfiguration)
	if err != nil {
		return nil, err
	}

	// OFFERER creates a DataChannel during initialization.
	if c.SDPRole == OFFERER {
		conn.pion.dataChannel, err = conn.pion.peerConnection.CreateDataChannel(c.Label, nil)
	}

	// For offerer, set the rest event handlers.
	if c.SDPRole == OFFERER {
		conn.setEventHandler(dataChannelStatus)

		// conduct automatic signalling if SignalMethod is set
		if c.SignalMethod != nil {
			var bChan chan bool = make(chan bool)

			// wait for local offer
			go func(blockingChan chan bool) {
				err := conn.CreateLocalDescription()
				blockingChan <- (err == nil)
			}(bChan)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case status := <-bChan:
				if !status {
					return nil, errors.New("failed to create local offer")
				}
				offer := conn.GetLocalDescription()
				// offer to JSON bytes
				offerBytes, err := json.Marshal(offer)
				if err != nil {
					return nil, err
				}
				err = c.SignalMethod.MakeOffer(offerBytes)
				if err != nil {
					return nil, err
				}
			}

			// wait for answer
			go func(blockingChan chan bool) {
				answerBytes, err := c.SignalMethod.GetAnswer()
				if err != nil {
					blockingChan <- false
					return
				}
				// answer from JSON bytes
				answer := &webrtc.SessionDescription{}
				err = json.Unmarshal(answerBytes, answer)
				if err != nil {
					blockingChan <- false
					return
				}
				err = conn.SetRemoteDescription(answer)
				blockingChan <- (err == nil)
			}(bChan)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case status := <-bChan:
				if !status {
					return nil, errors.New("failed to receive answer")
				}
			}

			// wait for datachannel
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case status := <-dataChannelStatus:
				if !status {
					return nil, errors.New("failed to receive datachannel")
				}
			}

		}
	} else if c.SDPRole == ANSWERER {
		// ANSWERER will wait until the dataChannel is received then set the event handlers.
		conn.pion.peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
			conn.setEventHandler(dataChannelStatus)
		})

		// conduct automatic signalling if SignalMethod is set
		if c.SignalMethod != nil {
			var bChan chan bool = make(chan bool)

			// wait for offer
			go func(blockingChan chan bool) {
				offerBytes, err := c.SignalMethod.GetOffer()
				if err != nil {
					blockingChan <- false
					return
				}
				// offer from JSON bytes
				offer := &webrtc.SessionDescription{}
				err = json.Unmarshal(offerBytes, offer)
				if err != nil {
					blockingChan <- false
					return
				}
				err = conn.SetRemoteDescription(offer)
				blockingChan <- (err == nil)
			}(bChan)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case status := <-bChan:
				if !status {
					return nil, errors.New("failed to receive offer")
				}
			}

			// wait for local answer
			go func(blockingChan chan bool) {
				err := conn.CreateLocalDescription()
				blockingChan <- (err == nil)
			}(bChan)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case status := <-bChan:
				if !status {
					return nil, errors.New("failed to create local answer")
				}
				answer := conn.GetLocalDescription()
				// answer to JSON bytes
				answerBytes, err := json.Marshal(answer)
				if err != nil {
					return nil, err
				}
				err = c.SignalMethod.Answer(answerBytes)
				if err != nil {
					return nil, err
				}
			}

			// wait for datachannel
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case status := <-dataChannelStatus:
				if !status {
					return nil, errors.New("failed to receive datachannel")
				}
			}
		}
	}

	return conn, nil
}

func (c *Config) Listen() (net.Listener, error) {
	return nil, nil
}

func (c *Config) DialPacket() (net.PacketConn, error) {
	return nil, nil
}

func (c *Config) DialPacketContext(ctx context.Context) (net.PacketConn, error) {
	return nil, nil
}

// Returns a single DataChannel as a PacketConn even if UDPMux is set.
// Call ListenPacket multiple times (with different SDP messages) to get multiple separate PacketConns.
func (c *Config) ListenPacket() (net.PacketConn, error) {
	return nil, nil
}

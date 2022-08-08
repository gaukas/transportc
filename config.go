package transportc

import (
	"math/rand"
	"net"
	"time"

	"github.com/pion/ice/v2"
	"github.com/pion/webrtc/v3"
)

type Config struct {
	// ListenerDTLSRole defines the DTLS role when Listening.
	// MUST be either DTLSRoleClient or DTLSRoleServer, as defined in RFC4347
	// DTLSRoleClient will send the ClientHello and start the handshake.
	// DTLSRoleServer will wait for the ClientHello.
	ListenerDTLSRole DTLSRole

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
	InterfaceFilter func(interfaceName string) (allowed bool)
}

func (c *Config) NewDialer(pConf webrtc.Configuration) (*Dialer, error) {
	settingEngine, err := c.BuildSettingEngine()
	if err != nil {
		return nil, err
	}

	return &Dialer{
		settingEngine: settingEngine,
		configuration: pConf,
	}, nil
}

func (c *Config) NewListener(pConf webrtc.Configuration) (*Listener, error) {
	settingEngine, err := c.BuildSettingEngine()
	if err != nil {
		return nil, err
	}

	settingEngine.SetAnsweringDTLSRole(c.ListenerDTLSRole)

	return &Listener{
		runningStatus:   LISTENER_NEW,
		rand:            rand.New(rand.NewSource(time.Now().UnixNano())),
		settingEngine:   settingEngine,
		configuration:   pConf,
		peerConnections: make(map[uint64]*webrtc.PeerConnection),
		conns:           make(chan net.Conn),
		abortAccept:     make(chan bool),
	}, nil
}

func (c *Config) BuildSettingEngine() (webrtc.SettingEngine, error) {
	var settingEngine webrtc.SettingEngine = webrtc.SettingEngine{}

	if c.IPs != nil {
		settingEngine.SetNAT1To1IPs(c.IPs.IPs, c.IPs.Type)
	}

	if c.PortRange != nil {
		err := settingEngine.SetEphemeralUDPPortRange(c.PortRange.Min, c.PortRange.Max)
		if err != nil {
			return webrtc.SettingEngine{}, err
		}
	}

	if c.UDPMux != nil {
		settingEngine.SetICEUDPMux(c.UDPMux)
	}

	if c.CandidateNetworkTypes != nil {
		settingEngine.SetNetworkTypes(c.CandidateNetworkTypes)
	}

	if c.InterfaceFilter != nil {
		settingEngine.SetInterfaceFilter(c.InterfaceFilter)
	}

	// GW: Making sure we will get a detached DataChannel as
	// a datachannel.ReadWriteCloser upon onOpen event.
	settingEngine.DetachDataChannels()

	return settingEngine, nil
}

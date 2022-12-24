package transportc

import (
	"net"
	"time"

	"github.com/gaukas/logging"
	"github.com/pion/ice/v2"
	"github.com/pion/webrtc/v3"
)

const (
	MTU_DEFAULT              = 1024
	MAX_RECV_TIMEOUT_DEFAULT = time.Second * 10
)

// Config is the configuration for the Dialer and Listener.
type Config struct {
	// CandidateNetworkTypes restricts ICE agent to gather
	// on only selected types of networks.
	CandidateNetworkTypes []webrtc.NetworkType

	// InterfaceFilter restricts ICE agent to gather ICE candidates
	// on only selected interfaces.
	InterfaceFilter func(interfaceName string) (allowed bool)

	// IPs includes a slice of IP addresses and one single ICE Candidate Type.
	// If set, will add these IPs as ICE Candidates
	IPs *NAT1To1IPs

	// ListenerDTLSRole defines the DTLS role when Listening.
	// MUST be either DTLSRoleClient or DTLSRoleServer, as defined in RFC4347
	// DTLSRoleClient will send the ClientHello and start the handshake.
	// DTLSRoleServer will wait for the ClientHello.
	ListenerDTLSRole DTLSRole

	Logger logging.Logger

	// PortRange is the range of ports to use for the DataChannel.
	PortRange *PortRange

	// ReusePeerConnection indicates whether to reuse the same PeerConnection
	// if possible, when Dialer dials multiple times.
	//
	// If set to true, Dialer.Dial() creates a new DataChannel on the same PeerConnection.
	// Otherwise, Dialer.Dial() negotiates for a new PeerConnection and creates a new DataChannel on it.
	ReusePeerConnection bool

	// Signal offers the automatic signaling when establishing the DataChannel.
	Signal Signal

	Timeout time.Duration

	// UDPMux allows serving multiple DataChannels over the one or more pre-established UDP socket.
	UDPMux ice.UDPMux

	// WebRTCConfiguration is the configuration for the underlying WebRTC PeerConnection.
	WebRTCConfiguration webrtc.Configuration
}

// NewDialer creates a new Dialer from the given configuration.
func (c *Config) NewDialer() (*Dialer, error) {
	settingEngine, err := c.BuildSettingEngine()
	if err != nil {
		return nil, err
	}

	if c.Logger == nil {
		c.Logger = logging.DefaultStderrLogger(logging.LOG_WARN)
	}

	return &Dialer{
		logger:              c.Logger,
		signal:              c.Signal,
		timeout:             c.Timeout,
		settingEngine:       settingEngine,
		configuration:       c.WebRTCConfiguration,
		reusePeerConnection: c.ReusePeerConnection,
	}, nil
}

// NewListener creates a new Listener from the given configuration.
func (c *Config) NewListener() (*Listener, error) {
	settingEngine, err := c.BuildSettingEngine()
	if err != nil {
		return nil, err
	}

	if c.Logger == nil {
		c.Logger = logging.DefaultStderrLogger(logging.LOG_ERROR)
	}

	settingEngine.SetAnsweringDTLSRole(c.ListenerDTLSRole) // ignore if any error

	l := &Listener{
		logger:          c.Logger,
		signal:          c.Signal,
		timeout:         c.Timeout,
		runningStatus:   LISTENER_NEW,
		settingEngine:   settingEngine,
		configuration:   c.WebRTCConfiguration,
		peerConnections: make(map[uint64]*webrtc.PeerConnection),
		conns:           make(chan net.Conn),
		closed:          make(chan bool),
	}

	return l, nil
}

// BuildSettingEngine builds a SettingEngine from the configuration.
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
	// a datachannel.ReadWriteCloser upon datachannel.onOpen event.
	settingEngine.DetachDataChannels()

	return settingEngine, nil
}

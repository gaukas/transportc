package transportc

import (
	"errors"

	"github.com/pion/webrtc/v3"
)

// Design adapted from github.com/Gaukas/seed2sdp

const (
	DataChannelBufferSizeDefault uint64 = 1024 * 1024 // Default Buffer Size: 1MB
	DataChannelBufferSizeMin     uint64 = 1024        // 1KB buffer could be too small...
)

type DataChannel struct {
	config               *DataChannelConfig // Config including
	WebRTCSettingEngine  webrtc.SettingEngine
	WebRTCConfiguration  webrtc.Configuration
	WebRTCPeerConnection *webrtc.PeerConnection
	WebRTCDataChannel    *webrtc.DataChannel
}

// DeclareDatachannel sets all the predetermined information needed to establish Peer Connection.
func DeclareDatachannel(dcconfig *DataChannelConfig, pionSettingEngine webrtc.SettingEngine, pionConfiguration webrtc.Configuration) *DataChannel {
	if dcconfig.SendBufferSize < DataChannelBufferSizeMin {
		dcconfig.SendBufferSize = DataChannelBufferSizeDefault
	}

	// Initialize the struct
	dataChannel := DataChannel{
		config:              dcconfig,
		WebRTCSettingEngine: pionSettingEngine,
		WebRTCConfiguration: pionConfiguration,
		// WebRTCPeerConnection: nil,
		// WebRTCDataChannel: nil,
	}
	return &dataChannel
}

// Initialize the DataChannel instance. From now on the SettingEngine functions will be no longer effective.
func (d *DataChannel) Initialize() error {
	var err error
	api := webrtc.NewAPI(webrtc.WithSettingEngine(d.WebRTCSettingEngine))
	d.WebRTCPeerConnection, err = api.NewPeerConnection(d.WebRTCConfiguration)
	if err != nil {
		return err
	}

	if d.config.SelfSDPType == "offer" {
		d.WebRTCDataChannel, err = d.WebRTCPeerConnection.CreateDataChannel(d.config.Label, nil)
	}
	return nil
}

func (d *DataChannel) CreateLocalDescription() error {
	var localDescription webrtc.SessionDescription
	var err error
	if d.config.SelfSDPType == "offer" {
		localDescription, err = d.WebRTCPeerConnection.CreateOffer(nil)
	} else if d.config.SelfSDPType == "answer" {
		localDescription, err = d.WebRTCPeerConnection.CreateAnswer(nil)
	}

	if err != nil {
		panic(err)
	}

	// Create channel that is blocked until ICE Gathering is complete
	gatherComplete := webrtc.GatheringCompletePromise(d.WebRTCPeerConnection)

	// Sets the LocalDescription, and starts our UDP listeners
	err = d.WebRTCPeerConnection.SetLocalDescription(localDescription)
	if err != nil {
		panic(err)
	}

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	<-gatherComplete
	return nil
}

func (d *DataChannel) CreateOffer() error {
	if d.config.SelfSDPType == "offer" {
		return d.CreateLocalDescription()
	}
	return errors.New("Mismatched SelfSDPType in config: " + d.config.SelfSDPType)
}

func (d *DataChannel) CreateAnswer() error {
	if d.config.SelfSDPType == "answer" {
		return d.CreateLocalDescription()
	}
	return errors.New("Mismatched SelfSDPType in config: " + d.config.SelfSDPType)
}

func (d *DataChannel) GetLocalDescription() *webrtc.SessionDescription {
	return d.WebRTCPeerConnection.LocalDescription()
}

func (d *DataChannel) SetRemoteDescription(remoteCandidates []ICECandidate) error {
	peerFp, _ := PredictDTLSFingerprint(d.config.PeerHkdfParams)
	peerICE, _ := PredictIceParameters(d.config.PeerHkdfParams)

	RemoteSDP := SDP{
		SDPType:       d.config.PeerSDPType(),
		Malleables:    SDPMalleablesFromSeed(d.config.PeerHkdfParams),
		Medias:        d.config.PeerMedias,
		Attributes:    d.config.PeerAttributes,
		Fingerprint:   peerFp,
		IceParams:     peerICE,
		IceCandidates: remoteCandidates,
	}

	rdesc := webrtc.SessionDescription{}
	FromJSON(RemoteSDP.String(), &rdesc)

	err := d.WebRTCPeerConnection.SetRemoteDescription(rdesc)
	return err
}

func (d *DataChannel) SetOffer(remoteCandidates []ICECandidate) error {
	if d.config.SelfSDPType == "answer" {
		return d.SetRemoteDescription(remoteCandidates)
	}
	return errors.New("SelfSDPType in config: " + d.config.SelfSDPType + " can't set offer.")
}

func (d *DataChannel) SetAnswer(remoteCandidates []ICECandidate) error {
	if d.config.SelfSDPType == "offer" {
		return d.SetRemoteDescription(remoteCandidates)
	}
	return errors.New("SelfSDPType in config: " + d.config.SelfSDPType + " can't set answer.")
}

// SettingEngine utilization

// SetPort sets the port for candidates. (For both Host's port and Srflx's local port)
func (d *DataChannel) SetPort(port uint16) *DataChannel {
	if d.WebRTCSettingEngine.SetEphemeralUDPPortRange(port, port) != nil {
		return nil
	}
	return d
}

// SetIP sets IP for ICE agents to treat as 1 to 1 IPs
// ips: list of IP in string
// iptype: Host (if full 1-to-1 DNAT), Srflx (if behind a NAT)
func (d *DataChannel) SetIP(ips []string, iptype ICECandidateType) *DataChannel {
	switch iptype {
	case Host:
		d.WebRTCSettingEngine.SetNAT1To1IPs(ips, webrtc.ICECandidateTypeHost)
		break
	case Srflx:
		d.WebRTCSettingEngine.SetNAT1To1IPs(ips, webrtc.ICECandidateTypeSrflx)
		break
	default:
		return nil
	}
	return d
}

// SetNetworkTypes specify the candidates' network type for ICE agent to gather.
func (d *DataChannel) SetNetworkTypes(candidateTypes []webrtc.NetworkType) *DataChannel {
	d.WebRTCSettingEngine.SetNetworkTypes(candidateTypes)
	return d
}

// SetInterfaceFilter uses the filter function and only gather candidates when filter(interface_name.String())==true
func (d *DataChannel) SetInterfaceFilter(filter func(string) bool) *DataChannel {
	d.WebRTCSettingEngine.SetInterfaceFilter(filter)
	return d
}

// SetDTLSActive() makes this instance creates the DTLS Connection (Send ClientHello).
// Only works for SDP answerer (usually the server)
func (d *DataChannel) SetDTLSActive() *DataChannel {
	d.WebRTCSettingEngine.SetAnsweringDTLSRole(webrtc.DTLSRoleClient)
	return d
}

// SetDTLSPassive() makes this instance waits the DTLS Connection (Wait for ClientHello).
// Only works for SDP answerer (usually the server)
func (d *DataChannel) SetDTLSPassive() *DataChannel {
	d.WebRTCSettingEngine.SetAnsweringDTLSRole(webrtc.DTLSRoleServer)
	return d
}

// ReadyToSend() when Data Channal is opened and is not exceeding the bytes limit.
func (d *DataChannel) ReadyToSend() bool {
	return (d.WebRTCDataChannel.ReadyState() == webrtc.DataChannelStateOpen) && (d.WebRTCDataChannel.BufferedAmount() < d.config.TxBufferSize)
}

// Send []byte object via Data Channel
func (d *DataChannel) Send(data []byte) error {
	if d.WebRTCDataChannel.ReadyState() == webrtc.DataChannelStateOpen {
		if d.WebRTCDataChannel.BufferedAmount() >= DataChannelBufferBytesLim {
			return ErrDataChannelAtCapacity
		}
		return d.WebRTCDataChannel.Send(data)
	} else if d.WebRTCDataChannel.ReadyState() == webrtc.DataChannelStateConnecting {
		return ErrDatachannelNotReady
	} else {
		return ErrDataChannelClosed
	}
}

// Close() ends Data Channel and Peer Connection
func (d *DataChannel) Close() {
	d.WebRTCDataChannel.Close()
	d.WebRTCPeerConnection.Close()
}

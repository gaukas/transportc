package transportc

import (
	"github.com/pion/webrtc/v3"
)

// Design adapted from github.com/Gaukas/seed2sdp

type DataChannel struct {
	config               *DataChannelConfig // Config including
	WebRTCSettingEngine  webrtc.SettingEngine
	WebRTCConfiguration  webrtc.Configuration
	WebRTCPeerConnection *webrtc.PeerConnection
	WebRTCDataChannel    *webrtc.DataChannel
}

// DeclareDatachannel sets all the predetermined information needed to establish Peer Connection.
func DeclareDatachannel(dcconfig *DataChannelConfig, pionSettingEngine webrtc.SettingEngine, pionConfiguration webrtc.Configuration) *DataChannel {
	if dcconfig.SendBufferSize > 0 && dcconfig.SendBufferSize < DataChannelBufferSizeMin {
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
	if len(d.config.IPAddr) > 0 {
		d.SetIP(d.config.IPAddr, d.config.CandidateType)
	}
	if d.config.Port > 0 {
		d.SetPort(d.config.Port)
	}

	api := webrtc.NewAPI(webrtc.WithSettingEngine(d.WebRTCSettingEngine))
	d.WebRTCPeerConnection, err = api.NewPeerConnection(d.WebRTCConfiguration)
	if err != nil {
		return err
	}

	if d.config.SelfSDPType == "offer" {
		d.WebRTCDataChannel, err = d.WebRTCPeerConnection.CreateDataChannel(d.config.Label, nil)
	}
	return err
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
		// panic(err) // not safe for release
		return err
	}

	// Create channel that is blocked until ICE Gathering is complete
	gatherComplete := webrtc.GatheringCompletePromise(d.WebRTCPeerConnection)

	// Sets the LocalDescription, and starts our UDP listeners
	err = d.WebRTCPeerConnection.SetLocalDescription(localDescription)
	if err != nil {
		// panic(err) // not safe for release
		return err
	}

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	<-gatherComplete
	return nil
}

func (d *DataChannel) GetLocalDescription() *webrtc.SessionDescription {
	return d.WebRTCPeerConnection.LocalDescription()
}

func (d *DataChannel) SetRemoteDescription(remoteSDP *webrtc.SessionDescription) error {
	// rdesc := webrtc.SessionDescription{}
	// err := json.Unmarshal([]byte(remoteSDP), &rdesc)
	// if err != nil {
	// 	return err
	// }
	return d.WebRTCPeerConnection.SetRemoteDescription(*remoteSDP)
}

// SettingEngine helper

// SetIP() specifies a list of IPs to use for local ICE candidates.
// The first input parameter should be a slice of strings while each string is an IP address.
// The second input parameter should be a webrtc.ICECandidateType
func (d *DataChannel) SetIP(ips []string, iptype webrtc.ICECandidateType) *DataChannel {
	d.WebRTCSettingEngine.SetNAT1To1IPs(ips, iptype)
	return d
}

// SetPort sets the port for candidates. (For both Host's port and Srflx's LOCAL port)
func (d *DataChannel) SetPort(port uint16) *DataChannel {
	if d.WebRTCSettingEngine.SetEphemeralUDPPortRange(port, port) != nil {
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
	return (d.WebRTCDataChannel.ReadyState() == webrtc.DataChannelStateOpen) && (d.config.SendBufferSize == 0 || d.WebRTCDataChannel.BufferedAmount() < d.config.SendBufferSize)
}

// Send []byte object via Data Channel
func (d *DataChannel) Send(data []byte) error {
	if d.WebRTCDataChannel.ReadyState() == webrtc.DataChannelStateOpen {
		if d.config.SendBufferSize > 0 && d.WebRTCDataChannel.BufferedAmount() >= d.config.SendBufferSize {
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
func (d *DataChannel) Close() error {
	err := d.WebRTCDataChannel.Close()
	if err != nil {
		return err
	}
	return d.WebRTCPeerConnection.Close()
}

package transportc

type WebRTCRole uint8

const (
	OFFERER WebRTCRole = iota
	ANSWERER
)

type WebRTConnStatus uint8

const (
	WebRTConnNew               WebRTConnStatus = 0      // No status, newly created.
	WebRTConnInit              WebRTConnStatus = 1 << 0 // Initialized.
	WebRTConnReady             WebRTConnStatus = 1 << 1 // Ready for send/recv
	WebRTConnClosed            WebRTConnStatus = 1 << 2
	WebRTConnErrored           WebRTConnStatus = 1 << 3 // Volatile state: Read error w
	WebRTConnLocalSDPReady     WebRTConnStatus = 1 << 4
	WebRTConnRemoteSDPReceived WebRTConnStatus = 1 << 5
)

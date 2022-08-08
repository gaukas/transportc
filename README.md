# TranspoRTC

A ready-to-go pluggable transport utilizing WebRTC datachannel written in Go. 

## Design

### Config 

A `Config` defines the behavior of the transport. A `Config` could be used to configure: 

- Automatic signalling when establishing the PeerConnection
- IP addresses to be used for ICE candidates
- Port range for ICE candidates
- UDP Mux for serving multiple connections over one UDP socket

### Dialer 

A `Dialer` is created from a `Config` and is used to dial one or more `Conn` backed by WebRTC DataChannel.

On its first call to `Dial`, the `Dialer` will create a new PeerConnection and DataChannel. On subsequent calls, the `Dialer` will reuse the existing PeerConnection and DataChannel.

### Listener 

A `Listener` is created from a `Config` and is used to listen for incoming `Conn` backed by WebRTC DataChannel. It looks for incoming SDP offers to establish new PeerConnections and also looks for incoming DataChannels on existing PeerConnections.

One `Listener` can maintain multiple `PeerConnection`s and on each `PeerConnection` multiple `DataChannel`s may co-exist.

A `Listener` requires a valid `SignalMethod` to function. 

### Conn

A `Conn` is created from a `Dialer` and is used to send and receive messages. Each `Conn` is backed by a single WebRTC DataChannel.
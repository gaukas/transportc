# TranspoRTC

A ready-to-go pluggable transport utilizing WebRTC datachannel written in Go. 

## Dev Roadmap

- `Dial(network, address string) (WebRTConn, error)`
    - [x] return valid `WebRTConn{}`
    - [ ] use `network`
    - [ ] use `address`

- `WebRTConn{}` as `net.Conn{}`
    - [x] `Read(b []byte) (n int, err error)`
    - [x] `Write(b []byte) (n int, err error)`
    - [x] `Close() error`
    - [x] `LocalAddr() net.Addr`
    - [x] `RemoteAddr() net.Addr`
    - [ ] `SetDeadline(t time.Time) error`
    - [ ] `SetReadDeadline(t time.Time) error`
    - [ ] `SetWriteDeadline(t time.Time) error`


## Guide

`transportc.WebRTConn{}` is a `net.Conn{}` compatible object with additional setting-up functions necessary for WebRTC peer connections and data channels.

### Minimal Viable Setup Steps

1. Creates `WebRTConn{}` with `transportc.Dial(network, remoteIP)`
2. Initialize `WebRTConn{}` with `.Init(&conf, pionSettingEngine, pionConfig)`
3. Role-dependent step:
    - For an offerer: Obtain the SDP offer with `.LocalSDP()` or `.LocalSDPJsonString()`
    - For an answerer: Feed in the SDP offer with `.SetRemoteSDP(remoteSDP string)` or `.SetRemoteSDPJsonString(remoteSdp *webrtc.SessionDescription)`
4. Role-dependent step:
    - For an offerer: Feed in the SDP answer with `.SetRemoteSDP(remoteSDP string)` or `.SetRemoteSDPJsonString(remoteSdp *webrtc.SessionDescription)`
    - For an answerer: Obtain the SDP answer with `.LocalSDP()` or `.LocalSDPJsonString()`
5. Wait until bit `WebRTConnReady` is set in `.Status()` evaluates to a non-zero value.
    - At this point, the datachannel is established. Enjoy!
6. Send or receive with `.Write(b)` or `.Read(b)` correspondingly. 
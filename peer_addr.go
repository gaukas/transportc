package transportc

// A net.Addr compliance struct
type DummyAddr struct {
	isLocal bool
}

func LocalDummyAddr() *DummyAddr {
	return &DummyAddr{isLocal: true}
}

func RemoteDummyAddr() *DummyAddr {
	return &DummyAddr{isLocal: false}
}

func (*DummyAddr) Network() string {
	return "udp"
}

func (da *DummyAddr) String() string {
	if da.isLocal {
		return "local"
	} else {
		return "remote"
	}
}

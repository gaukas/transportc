package transportc

import (
	"net"
	"testing"
)

func CheckIfConn(c net.Conn) error {
	return nil
}

func TestCheckIfConn(t *testing.T) {
	conn := WebRTConn{}
	CheckIfConn(conn)
}

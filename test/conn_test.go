package transportc_test

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/gaukas/transportc"
)

// Negative Test: Write to closed Conn
func TestWriteToClosedConn(t *testing.T) {
	config := &transportc.Config{
		Signal: transportc.NewDebugSignal(8),
	}

	// Setup a listener to accept the connection first
	listener, err := config.NewListener()
	if err != nil {
		t.Fatal(err)
	}

	defer listener.Stop()
	listener.Start()

	dialer, err := config.NewDialer()
	if err != nil {
		t.Fatal(err)
	}
	defer dialer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel() // cancel the context to make sure it is done

	// Create 3 Client Connections
	cConn, err := dialer.DialContext(ctx, "RANDOM_LABEL")
	if err != nil {
		t.Fatalf("First DialContext error: %v", err)
	}
	if cConn == nil {
		t.Fatal("First DialContext returned nil")
	}
	defer cConn.Close()

	sConn, err := listener.Accept()
	if err != nil {
		t.Fatalf("First Accept error: %v", err)
	}
	if sConn == nil {
		t.Fatal("First Accept returned nil")
	}
	defer sConn.Close()

	cConn2, err := dialer.DialContext(ctx, "RANDOM_LABEL_2")
	if err != nil {
		t.Fatalf("Second DialContext error: %v", err)
	}
	if cConn2 == nil {
		t.Fatal("Second DialContext returned nil")
	}
	defer cConn2.Close()

	sConn2, err := listener.Accept()
	if err != nil {
		t.Fatalf("Second Accept error: %v", err)
	}
	if sConn2 == nil {
		t.Fatal("Second Accept returned nil")
	}
	defer sConn2.Close()

	// Close the first Conn
	cConn.Close()

	// Write to first Conn - should fail
	_, err = cConn.Write([]byte("Hello"))
	if err == nil {
		t.Fatal("Write to closed Conn should fail")
	}

	// Write to second Conn - should succeed
	_, err = cConn2.Write([]byte("Hello"))
	if err != nil {
		t.Fatalf("Write to second Conn error: %v", err)
	}

	// Receive on second Conn - should succeed
	buf := make([]byte, 1024)
	n, err := sConn2.Read(buf)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if n != 5 {
		t.Fatalf("Read returned %d bytes", n)
	}
	if string(buf[:n]) != "Hello" {
		t.Fatalf("Read returned %s", string(buf[:n]))
	}

	// Generate SUPER LONG message
	longMsg := make([]byte, 65535)
	// fill the message with random data
	rand.Read(longMsg) // skipcq: GSC-G404

	// Write to second Conn - should succeed
	written, err := cConn2.Write(longMsg)
	if err != nil {
		t.Fatalf("Write to second Conn error: %v", err)
	}
	if written != 65535 {
		t.Fatalf("Write to second Conn returned %d bytes", written)
	}

	// Write something else to second Conn following the long message
	_, _ = cConn2.Write([]byte("Hello"))

	longRecvBuf := make([]byte, 65540)
	// Receive on second Conn - should succeed
	n, err = sConn2.Read(longRecvBuf)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}

	if n != 65535 {
		t.Fatalf("Read returned %d bytes", n)
	}

	if string(longRecvBuf[:n]) != string(longMsg) {
		t.Fatalf("Read returned wrong message on super long")
	}

	// Write over-length message to second Conn - should fail
	overLengthMsg := make([]byte, 65550)
	rand.Read(overLengthMsg) // skipcq: GSC-G404
	_, err = cConn2.Write(overLengthMsg)
	if err == nil {
		t.Fatal("Write over-length message to second Conn should fail")
	}

	t.Logf("Closing all connections")
	cConn.Close()
	cConn2.Close()
	sConn.Close()
	sConn2.Close()
	t.Logf("Prepare to sleep...")
	time.Sleep(2 * time.Second)
	t.Logf("After sleep...")

	cConn3, err := dialer.DialContext(ctx, "RANDOM_LABEL_3")
	if err != nil {
		t.Fatalf("Third DialContext error: %v", err)
	}
	if cConn3 == nil {
		t.Fatal("Third DialContext returned nil")
	}
	defer cConn3.Close()

	sConn3, err := listener.Accept()
	if err != nil {
		t.Fatalf("Third Accept error: %v", err)
	}
	if sConn3 == nil {
		t.Fatal("Third Accept returned nil")
	}
	defer sConn3.Close()

	// Write to third Conn - should succeed
	_, err = cConn3.Write([]byte("Hello"))
	if err != nil {
		t.Fatalf("Write to third Conn error: %v", err)
	}

	// Receive on third Conn - should succeed
	n, err = sConn3.Read(buf)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}

	if n != 5 {
		t.Fatalf("Read returned %d bytes", n)
	}

	if string(buf[:n]) != "Hello" {
		t.Fatalf("Read returned %s", string(buf[:n]))
	}

	// Write to third Conn - should succeed
	_, err = cConn3.Write(longMsg)
	if err != nil {
		t.Fatalf("Write to third Conn error: %v", err)
	}

	// Receive on third Conn - should succeed
	n, err = sConn3.Read(longRecvBuf)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}

	if n != 65535 {
		t.Fatalf("Read returned %d bytes", n)
	}

	if string(longRecvBuf[:n]) != string(longMsg) {
		t.Fatalf("Read returned wrong message on super long")
	}
}

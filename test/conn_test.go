package transportc_test

import (
	"context"
	"testing"
	"time"

	"github.com/Gaukas/transportc"
)

// Negative Test: Write to closed Conn
func TestWriteToClosedConn(t *testing.T) {
	signalMethod := transportc.NewDebugSignal(3)

	// Setup a listener to accept the connection first
	listener, err := getDefaultListener()
	if err != nil {
		t.Fatal(err)
	}

	listener.SignalMethod = signalMethod
	defer listener.Stop()
	listener.Start()

	dialer, err := getDefaultDialer()
	if err != nil {
		t.Fatal(err)
	}
	defer dialer.Close()

	dialer.SignalMethod = signalMethod

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
}

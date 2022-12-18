package transportc_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/gaukas/transportc"
)

func TestAccept(t *testing.T) {
	config := &transportc.Config{
		Signal: transportc.NewDebugSignal(8),
	}

	// Setup a listener to accept the connection first
	listener, err := config.NewListener()
	if err != nil {
		t.Fatal(err)
	}

	defer listener.Close()
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

	cConn2, err := dialer.DialContext(ctx, "RANDOM_LABEL_2")
	if err != nil {
		t.Fatalf("Second DialContext error: %v", err)
	}
	if cConn2 == nil {
		t.Fatal("Second DialContext returned nil")
	}
	defer cConn2.Close()

	cConn3, err := dialer.DialContext(ctx, "RANDOM_LABEL_3")
	if err != nil {
		t.Fatalf("Second DialContext error: %v", err)
	}
	if cConn3 == nil {
		t.Fatal("Second DialContext returned nil")
	}
	defer cConn3.Close()

	cConns := []net.Conn{
		cConn,
		cConn2,
		cConn3,
	}
	go func() {
		for idx, clientConn := range cConns {
			_, err = clientConn.Write([]byte(fmt.Sprintf("HELLO%d", idx+1)))
			if err != nil {
				fmt.Printf("#%d Hello error: %v\n", idx+1, err)
				return
			}
		}

		for idx, clientConn := range cConns {
			buf := make([]byte, 16)
			n, err := clientConn.Read(buf)
			if err != nil {
				fmt.Printf("Read error: %v\n", err)
				return
			}
			if n != 2 {
				fmt.Printf("Read error: expected 2 bytes (Hi), got %d bytes (%s)", n, string(buf[:n]))
				return
			}

			_, err = clientConn.Write([]byte(fmt.Sprintf("BYE%d", idx+1)))
			if err != nil {
				fmt.Printf("#%d Bye error: %v", idx, err)
				return
			}
		}
	}()

	sConns := make([]net.Conn, 3)

	// accept 3 connections
	for idx := 0; idx < 3; idx++ {
		conn, err := listener.Accept()
		if err != nil {
			t.Fatalf("Accept error: %v", err)
		}
		if conn == nil {
			t.Fatal("Accept returned nil")
		}
		defer conn.Close()

		buf := make([]byte, 16)
		n, err := conn.Read(buf)
		if err != nil {
			t.Fatalf("Read error: %v", err)
		}
		if n != 6 { // "HELLO1"/"HELLO2"/"HELLO3"
			t.Fatalf("Read error: expected 6 bytes (HELLO#), got %d bytes (%s)", n, string(buf[:n]))
		}

		switch string(buf[:n]) {
		case "HELLO1":
			sConns[0] = conn
		case "HELLO2":
			sConns[1] = conn
		case "HELLO3":
			sConns[2] = conn
		}

		_, err = conn.Write([]byte("Hi"))
		if err != nil {
			t.Fatalf("Write error: %v", err)
		}
	}

	// Receive BYE1, BYE2, BYE3
	for idx, sConn := range sConns {
		buf := make([]byte, 16)
		n, err := sConn.Read(buf)
		if err != nil {
			t.Fatalf("Read error: %v", err)
		}
		if n != 4 { // "BYE1"/"BYE2"/"BYE3"
			t.Fatalf("Read error: expected 4 bytes (BYE#), got %d bytes (%s)", n, string(buf[:n]))
		}

		switch idx {
		case 0:
			if string(buf[:n]) != "BYE1" {
				t.Fatalf("Read error: expected BYE1, got %s", string(buf[:n]))
			}
		case 1:
			if string(buf[:n]) != "BYE2" {
				t.Fatalf("Read error: expected BYE2, got %s", string(buf[:n]))
			}
		case 2:
			if string(buf[:n]) != "BYE3" {
				t.Fatalf("Read error: expected BYE3, got %s", string(buf[:n]))
			}
		}
	}
}

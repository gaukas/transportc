package transportc_test

import (
	"context"
	"testing"
	"time"

	"github.com/gaukas/transportc"
)

// Negative Test for Dialer.DialContext with an expired context
func TestDialContextWithDoneContext(t *testing.T) {
	config := &transportc.Config{
		Signal: transportc.NewDebugSignal(8),
	}

	dialer, err := config.NewDialer()
	if err != nil {
		t.Fatal(err)
	}
	defer dialer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Nanosecond)
	defer cancel()                    // cancel the context to make sure it is done
	time.Sleep(50 * time.Millisecond) // sleep 1ms to make sure the dialer is done

	err = ctx.Err()
	if err == nil {
		t.Fatal("ctx is not expired as expected")
	}

	_, err = dialer.DialContext(ctx, "RANDOM_LABEL")
	if err != context.Canceled && err != context.DeadlineExceeded {
		t.Fatal("DialContext should return context.Canceled or context.DeadlineExceeded")
	}
}

// Negative Test for Dialer.DialContext with no answering peer to connect to
func TestDialContextWithoutPeer(t *testing.T) {
	config := &transportc.Config{
		Signal: transportc.NewDebugSignal(8),
	}

	dialer, err := config.NewDialer()
	if err != nil {
		t.Fatal(err)
	}
	defer dialer.Close()

	timeStart := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel() // cancel the context to make sure it is done

	conn, err := dialer.DialContext(ctx, "RANDOM_LABEL")
	if conn != nil {
		conn.Close() // close the connection if it is not nil. absolutely something is wrong.
	}
	if err == nil {
		t.Fatal("DialContext should timeout as no peer is available")
	}

	timeEnd := time.Now()

	if timeEnd.Sub(timeStart) < 5*time.Second {
		t.Fatal("DialContext returned earlier than set timeout")
	}
}

// Positive Test for Dialer.DialContext with a default answering peer to connect to
func TestDialContext(t *testing.T) {
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

	timeStart := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel() // cancel the context to make sure it is done
	conn, err := dialer.DialContext(ctx, "RANDOM_LABEL")
	if err != nil {
		t.Fatalf("DialContext error: %v", err)
	}
	timeEnd := time.Now()

	if timeEnd.Sub(timeStart) > 5*time.Second {
		t.Fatal("DialContext returned later than set timeout")
	}
	if conn == nil {
		t.Fatal("DialContext returned nil")
	}

	conn.Close()
}

func TestDialContextMultipleCall(t *testing.T) {
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

	conn, err := dialer.DialContext(ctx, "RANDOM_LABEL")
	if err != nil {
		t.Fatalf("First DialContext error: %v", err)
	}
	if conn == nil {
		t.Fatal("First DialContext returned nil")
	}
	conn.Close()

	conn2, err := dialer.DialContext(ctx, "RANDOM_LABEL_2")
	if err != nil {
		t.Fatalf("Second DialContext error: %v", err)
	}
	if conn2 == nil {
		t.Fatal("Second DialContext returned nil")
	}
	defer conn2.Close()

	conn3, err := dialer.DialContext(ctx, "RANDOM_LABEL_3")
	if err != nil {
		t.Fatalf("Second DialContext error: %v", err)
	}
	if conn3 == nil {
		t.Fatal("Second DialContext returned nil")
	}
	defer conn3.Close()
}

func BenchmarkSingleDialerDialing(b *testing.B) {
	config := &transportc.Config{
		Signal: transportc.NewDebugSignal(8),
	}

	// Setup a listener to accept the connection first
	listener, err := config.NewListener()
	if err != nil {
		b.Fatal(err)
	}

	defer listener.Stop()
	listener.Start()

	dialer, err := config.NewDialer()
	if err != nil {
		b.Fatal(err)
	}
	defer dialer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel() // cancel the context to make sure it is done

	for i := 0; i < b.N; i++ {
		conn, err := dialer.DialContext(ctx, "RANDOM_LABEL")
		if err != nil {
			b.Fatalf("DialContext error: %v", err)
		}
		if conn == nil {
			b.Fatal("DialContext returned nil")
		}
		defer conn.Close()
		conn.Write([]byte("Hello"))
	}
}

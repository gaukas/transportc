package transportc_test

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gaukas/transportc"
)

func TestConnComm(t *testing.T) {
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

func BenchmarkConn(b *testing.B) {
	benchmarkSingleConn(b, 1024)
	benchmarkSingleConn(b, 2048)
	benchmarkSingleConn(b, 4096)
	benchmarkSingleConn(b, 8192)
	// benchmarkMultiConn(b, 1024, 10)
	// benchmarkMultiConn(b, 2048, 10)
	// benchmarkMultiConn(b, 4096, 10)
	// benchmarkMultiConn(b, 8192, 10)
	// benchmarkMultiConn(b, 1024, 20)
	// benchmarkMultiConn(b, 2048, 20)
	// benchmarkMultiConn(b, 4096, 20)
	// benchmarkMultiConn(b, 8192, 20)
}

func benchmarkSingleConn(b *testing.B, pktSize int) {
	config := &transportc.Config{
		Signal: transportc.NewDebugSignal(8),
	}

	// Setup a listener to accept the connection first
	listener, err := config.NewListener()
	if err != nil {
		b.Fatal(err)
	}

	defer listener.Close()
	listener.Start()

	dialer, err := config.NewDialer()
	if err != nil {
		b.Fatal(err)
	}
	defer dialer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel() // cancel the context to make sure it is done

	cConn, err := dialer.DialContext(ctx, "RANDOM_LABEL")
	if err != nil {
		b.Fatalf("DialContext error: %v", err)
	}
	if cConn == nil {
		b.Fatal("DialContext returned nil")
	}

	sConn, err := listener.Accept()
	if err != nil {
		b.Fatalf("Accept error: %v", err)
	}
	if sConn == nil {
		b.Fatal("Accept returned nil")
	}

	// goroutine to echo the message received by server
	go func() {
		buf := make([]byte, 65536)
		var byteRecv int = 0
		var msgCntr int = 0
		sTime := time.Now()
		for {
			n, err := sConn.Read(buf)
			if err != nil {
				sConn.Close()
			}
			if n == 4 && string(buf[:n]) == "GOOD" {
				break
			}
			byteRecv += n
			msgCntr++
		}
		elapse := time.Since(sTime)
		lat := float64(elapse.Microseconds()) / float64(msgCntr)
		bandwidth := byteRecv / int(elapse.Microseconds())
		sConn.Write([]byte(fmt.Sprintf("Bw: %dMB/s, Lat: %.2fus", bandwidth, lat)))
	}()

	cBuf := make([]byte, pktSize)
	var i int
	for i = 0; i < 10000; i++ {
		rand.Read(cBuf) // skipcq: GSC-G404
		_, err = cConn.Write(cBuf)
		if err != nil {
			b.Errorf("Write error: %v", err)
		}
	}
	cConn.Write([]byte("GOOD"))
	n, _ := cConn.Read(cBuf)
	b.Logf("%dKB Test, 10000 round(s), %s", pktSize/1024, string(cBuf[:n]))
}

func benchmarkMultiConn(b *testing.B, pktSize int, multi int) {
	config := &transportc.Config{
		Signal: transportc.NewDebugSignal(8),
	}

	// Setup a listener to accept the connection first
	listener, err := config.NewListener()
	if err != nil {
		b.Fatal(err)
	}

	defer listener.Close()
	listener.Start()

	wg := &sync.WaitGroup{}
	bw := &atomic.Uint64{}  // KB/s
	lat := &atomic.Uint64{} // us
	cond := &sync.WaitGroup{}
	cond.Add(multi*2 + 1)

	go func(l net.Listener) {
		for {
			sConn, err := listener.Accept()
			if err != nil {
				return
			}
			if sConn != nil {
				wg.Add(1)
				go func() {
					defer wg.Done()
					buf := make([]byte, 65536)
					var byteRecv int = 0
					var msgCntr int = 0
					cond.Done()
					cond.Wait()
					sTime := time.Now()
					for {
						n, err := sConn.Read(buf)
						if err != nil {
							sConn.Close()
						}
						if n == 4 && string(buf[:n]) == "GOOD" {
							break
						}
						byteRecv += n
						msgCntr++
					}
					elapse := time.Since(sTime)
					if msgCntr > 0 {
						lat.Add(uint64(elapse.Microseconds()) / uint64(msgCntr))
					}
					localBw := byteRecv * 1000 / int(elapse.Microseconds())
					bw.Add(uint64(localBw))
				}()
			}
		}
	}(listener)

	dialer, err := config.NewDialer()
	if err != nil {
		b.Fatal(err)
	}
	defer dialer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel() // cancel the context to make sure it is done

	dummyConn, err := dialer.DialContext(ctx, "RANDOM_LABEL")
	if err != nil {
		b.Fatal("Can't create dummy conn. Fail.")
	}

	for i := 0; i < multi; i++ {
		go func() {
			cConn, err := dialer.DialContext(ctx, "RANDOM_LABEL")
			if err != nil {
				return
			}

			cBuf := make([]byte, pktSize)
			var i int
			cond.Done()
			cond.Wait()
			for i = 0; i < 10000/multi; i++ {
				rand.Read(cBuf) // skipcq: GSC-G404
				_, err = cConn.Write(cBuf)
				if err != nil {
					b.Errorf("Write error: %v", err)
				}
			}
			cConn.Write([]byte("GOOD"))
		}()
	}
	dummyConn.Write([]byte("GOOD"))
	time.Sleep(2 * time.Second)

	wg.Wait()
	listener.Close()
	b.Logf("%d Connections, %dKB Test, %d round(s) each, Bw: %dMB/s, Lat: %dus", multi, pktSize/1024, 10000/multi, bw.Load()/1024, lat.Load()/uint64(multi))
}

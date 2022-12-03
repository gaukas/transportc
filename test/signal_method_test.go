package transportc_test

import (
	"bytes"
	"crypto/rand"
	"testing"
	"time"

	"github.com/gaukas/transportc"
)

const estimatedSDPLength = 512 // an educated guess of max length

func TestDebugSignal(t *testing.T) {
	ds := transportc.NewDebugSignal(1)

	// Generate random dummy offer input as []byte
	dummyOfferInput := make([]byte, estimatedSDPLength)
	_, err := rand.Read(dummyOfferInput)
	if err != nil {
		t.Fatalf("Error generating random dummy offer input: %v", err)
	}

	// Generate random dummy answer input as []byte
	dummyAnswerInput := make([]byte, estimatedSDPLength)
	_, err = rand.Read(dummyAnswerInput)
	if err != nil {
		t.Fatalf("Error generating random dummy answer input: %v", err)
	}

	err = ds.MakeOffer(dummyOfferInput)
	if err != nil {
		t.Fatalf("Error making offer: %v", err)
	}

	chanOffer := make(chan []byte)

	go func() {
		offerOutput, _ := ds.GetOffer()
		chanOffer <- offerOutput
	}()

	select {
	case offerOutput := <-chanOffer:
		// verify offer output matches dummy offer input
		if !bytes.Equal(dummyOfferInput, offerOutput) {
			t.Fatalf("Offer output does not match dummy offer input")
		}
	case <-time.After(time.Second): // 1s timeout
		t.Fatalf("Timeout waiting for offer")
	}
	close(chanOffer)

	err = ds.Answer(dummyAnswerInput)
	if err != nil {
		t.Fatalf("Error answering: %v", err)
	}

	chanAnswer := make(chan []byte)

	go func() {
		answerOutput, _ := ds.GetAnswer()
		chanAnswer <- answerOutput
	}()

	select {
	case answerOutput := <-chanAnswer:
		// verify answer output matches dummy answer input
		if !bytes.Equal(dummyAnswerInput, answerOutput) {
			t.Fatalf("Answer output does not match dummy answer input")
		}
	case <-time.After(time.Second): // 1s timeout
		t.Fatalf("Timeout waiting for answer")
	}
	close(chanAnswer)
}

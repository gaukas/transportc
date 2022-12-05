package utils

import "time"

func DelayedExecution(delay time.Duration, f func()) {
	time.Sleep(delay)
	f()
}

package node

import (
	"math/rand"
	"sync"
	"time"
)

// TODO: randomize timeout
const MIN_DURATION int = 150
const DURATION_RANGE int = 150

type ElectionTimer struct {
	Duration           time.Duration
	startTimer         chan bool
	Finished           chan bool
	EndCurrentElection chan bool
	NoMoreElections    chan bool
	stopTimer          chan bool
	Lapsed             chan bool
	stopped            bool
	mtx                sync.Mutex
}

func newElectionTimer() ElectionTimer {
	return ElectionTimer{
		Duration:           time.Millisecond * time.Duration(rand.Intn(DURATION_RANGE)+MIN_DURATION),
		startTimer:         make(chan bool),
		Finished:           make(chan bool),
		Lapsed:             make(chan bool),
		EndCurrentElection: make(chan bool),
		NoMoreElections:    make(chan bool),
		stopTimer:          make(chan bool),
		stopped:            false,
	}
}

func (timer *ElectionTimer) Start() {
	timer.Reset()
	for {
		select {
		case <-time.After(timer.Duration):
			if !timer.stopped {
				timer.Lapsed <- true
				timer.EndCurrentElection <- true
			}
			return
		case <-timer.stopTimer:
			return
		}
	}
}

func (timer *ElectionTimer) Reset() {
	timer.mtx.Lock()
	timer.Duration = time.Millisecond * time.Duration(rand.Intn(DURATION_RANGE)+MIN_DURATION)
	timer.stopped = false
	timer.mtx.Unlock()
}

// Stop the timer early
func (timer *ElectionTimer) Stop() {
	// WARNING: locks - only call after node.mtx.Unlock()
	timer.mtx.Lock()
	defer timer.mtx.Unlock()
	if !timer.stopped {
		timer.stopped = true
		timer.EndCurrentElection <- true
		timer.NoMoreElections <- true
		timer.stopTimer <- true
	}
}

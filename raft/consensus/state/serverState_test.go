package state

import (
	"fmt"
	"testing"
)

type stateTransitionTest struct {
	input    ServerState
	expected ServerState
}

func TestMakeServerState(t *testing.T) {
	var expected = Follower
	var actual = newServerState()
	if actual != expected {
		t.Errorf("MakeServerState() = %s; expected %s", actual, expected)
	}
}

func TestTimeoutAndStartElection(t *testing.T) {
	var tests []stateTransitionTest = []stateTransitionTest{
		{Follower, Candidate},
		{Candidate, INVALID},
		{Leader, INVALID},
	}
	for _, tt := range tests {
		testname := fmt.Sprintf("%s.timeoutAndStartElection()=%s", tt.input, tt.expected)
		t.Run(testname, func(t *testing.T) {
			answer, _ := tt.input.TimeoutAndStartElection()
			if answer != tt.expected {
				t.Errorf("%s.timeoutAndStartElection() = %s; expected %s", tt.input, answer, tt.expected)
			}
		})
	}
}

func TestTimeoutAndNewElection(t *testing.T) {
	var tests []stateTransitionTest = []stateTransitionTest{
		{Candidate, Candidate},
		{Follower, INVALID},
		{Leader, INVALID},
	}
	for _, tt := range tests {
		testname := fmt.Sprintf("%s.timeoutAndNewElection()=%s", tt.input, tt.expected)
		t.Run(testname, func(t *testing.T) {
			answer, _ := tt.input.TimeoutAndNewElection()
			if answer != tt.expected {
				t.Errorf("%s.timeoutAndNewElection() = %s; expected %s", tt.input, answer, tt.expected)
			}
		})
	}
}
func TestReceivedMajorityVote(t *testing.T) {
	var tests []stateTransitionTest = []stateTransitionTest{
		{Candidate, Leader},
		{Leader, INVALID},
		{Follower, INVALID},
	}
	for _, tt := range tests {
		testname := fmt.Sprintf("%s.receivedMajorityVote()=%s", tt.input, tt.expected)
		t.Run(testname, func(t *testing.T) {
			answer, _ := tt.input.ReceivedMajorityVote()
			if answer != tt.expected {
				t.Errorf("%s.receivedMajorityVote() = %s; expected %s", tt.input, answer, tt.expected)
			}
		})
	}
}

func TestDiscoverServerWithHigherTerm(t *testing.T) {
	var tests []stateTransitionTest = []stateTransitionTest{
		{Leader, Follower},
		{Follower, INVALID},
		{Candidate, INVALID},
	}
	for _, tt := range tests {
		testname := fmt.Sprintf("%s.discoverServerWithHigherTerm()=%s", tt.input, tt.expected)
		t.Run(testname, func(t *testing.T) {
			answer, _ := tt.input.DiscoverServerWithHigherTerm()
			if answer != tt.expected {
				t.Errorf("%s.discoverServerWithHigherTerm() = %s; expected %s", tt.input, answer, tt.expected)
			}
		})
	}
}

func TestDiscoverCurrentLeaderOrNewTerm(t *testing.T) {
	var tests []stateTransitionTest = []stateTransitionTest{
		{Candidate, Follower},
		{Follower, INVALID},
		{Leader, INVALID},
	}
	for _, tt := range tests {
		testname := fmt.Sprintf("%s.discoverCurrentLeaderOrNewTerm()=%s", tt.input, tt.expected)
		t.Run(testname, func(t *testing.T) {
			answer, _ := tt.input.DiscoverCurrentLeaderOrNewTerm()
			if answer != tt.expected {
				t.Errorf("%s.discoverCurrentLeaderOrNewTerm() = %s; expected %s", tt.input, answer, tt.expected)
			}
		})
	}
}

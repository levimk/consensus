package state

import (
	"errors"
	"fmt"
	"log"
)

type ServerState int

type StateTransition struct {
	From ServerState
	To   ServerState
}

const INVALID ServerState = -1

const (
	Follower ServerState = iota
	Candidate
	Leader
)

func (s ServerState) String() string {
	switch s {
	case Follower:
		return "FOLLOWER"
	case Candidate:
		return "CANDIDATE"
	case Leader:
		return "LEADER"
	default:
		return "INVALID"
	}
}

func newServerState() ServerState {
	return Follower
}

func (ns *NodeState) GetServerState() ServerState {
	return ns.ServerState
}
func (ns *NodeState) SetServerState(ss ServerState) {
	if ss != Follower && ss != Candidate && ss != Leader {
		ns.ServerState = ss
	} else {
		log.Fatalln("Tried to set node to invalid serverState", ss)
	}
}

func (actual ServerState) badStateTransitionError(expected ServerState) error {
	return errors.New(fmt.Sprintf("must be a %s, got %s", expected, actual))
}

// ##------ FOLLOWER ------##
func (s ServerState) TimeoutAndStartElection() (ServerState, error) {
	if s == Follower {
		return Candidate, nil
	}
	return INVALID, s.badStateTransitionError(Follower)
}

// ##------ LEADER ------##
func (s ServerState) DiscoverServerWithHigherTerm() (ServerState, error) {
	if s == Leader {
		return Follower, nil
	}
	return INVALID, s.badStateTransitionError(Leader)
}

// ##------ CANDIDATE ------##
func (s ServerState) TimeoutAndNewElection() (ServerState, error) {
	// @DONE (implicitly)
	if s == Candidate {
		return Candidate, nil
	}
	return INVALID, s.badStateTransitionError(Candidate)
}

func (s ServerState) ReceivedMajorityVote() (ServerState, error) {
	// @DONE
	if s == Candidate {
		return Leader, nil
	}
	return INVALID, s.badStateTransitionError(Candidate)
}

func (s ServerState) DiscoverCurrentLeaderOrNewTerm() (ServerState, error) {
	// TODO 1: requestVote DONE
	// TODO 2: discover current leader NOT DONE
	// TODO 2: request vote reply DONE
	if s == Candidate {
		return Follower, nil
	}
	return INVALID, s.badStateTransitionError(Candidate)
}

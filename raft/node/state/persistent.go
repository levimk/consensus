package state

import (
	"errors"
	"fmt"
	"raft/types"
)

const NO_VOTE types.NodeId = -1

// TODO: put this behind persistence layer

type persistentState struct {
	CurrentTerm types.Term
	VotedFor    types.NodeId
	Log         types.Entries
}

func newPersistentState() persistentState {
	return persistentState{
		CurrentTerm: 0,
		VotedFor:    NO_VOTE,
		Log:         make(types.Entries, 0), // TODO: adjust size and capacity?
	}
}

func (ps *persistentState) GetEntryAt(index int) (types.Entry, error) {
	if index < 0 || index < len(ps.Log) {
		return ps.Log[index], nil
	}
	errMsg := fmt.Sprintf("InvalidIndex: must be ≥0 and less than %d but got %d", len(ps.Log), index)
	return types.Entry{}, errors.New(errMsg)
}

func (ps *persistentState) AppendEntry(index types.LogIndex, term types.Term, command types.Command) error {
	if int(index) != len(ps.Log) {
		return errors.New(fmt.Sprintf("PersistentStateError: invalid log index - got %d, expected %d", index, len(ps.Log)))
	}
	entry := types.Entry{
		Index:   index,
		Term:    term,
		Command: command,
	}
	ps.Log = append(ps.Log, entry)
	return nil
}

// TODO: make persistent Log type with getters and setters

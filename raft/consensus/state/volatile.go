package state

import "raft/types"

type volatileState struct {
	commitIndex types.CommitIndex
	lastApplied types.LogIndex
}

func newVolatileState() volatileState {
	return volatileState{
		commitIndex: types.MakeCommitIndex(),
		lastApplied: types.MakeLogIndex(),
	}
}

func (vs *volatileState) GetCommitIndex() types.CommitIndex {
	return vs.commitIndex
}

func (vs *volatileState) SetCommitIndex(newIndex types.CommitIndex) {
	vs.commitIndex = newIndex
}

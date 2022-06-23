package types

type LogIndex int
type CommitIndex int

func MakeLogIndex() LogIndex {
	return 0
}

func MakeCommitIndex() CommitIndex {
	return CommitIndex(0)
}

func (ci CommitIndex) next() CommitIndex {
	return CommitIndex(ci + 1)
}

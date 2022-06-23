package types

type Entry struct {
	Term    Term
	Index   LogIndex
	Command Command
}

type Entries []Entry

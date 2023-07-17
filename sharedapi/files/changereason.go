package files

type ChangeReason struct {
	Reason string
	// Note: I'll probably add lots of other metadata about why a change is requested
}

type DiffWithChangeReason struct {
	Diff         *Diff
	ChangeReason *ChangeReason
}

func (d *DiffWithChangeReason) Validate() error {
	return d.Diff.Validate()
}

type StateWithChangeReason struct {
	State        State
	ChangeReason *ChangeReason
}

func (s *StateWithChangeReason) Validate() error {
	return s.State.Validate()
}

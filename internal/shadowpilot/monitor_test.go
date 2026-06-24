package shadowpilot

import (
	"testing"
)

func TestGetCurrentState(t *testing.T) {
	state := GetCurrentState()
	if state.UntrackedFiles == nil {
		t.Error("UntrackedFiles should not be nil")
	}
	if state.ModifiedFiles == nil {
		t.Error("ModifiedFiles should not be nil")
	}
}

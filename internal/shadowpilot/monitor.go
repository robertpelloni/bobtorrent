package shadowpilot

import (
	"bytes"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// GitState represents the tracked repository anomaly state.
type GitState struct {
	UntrackedFiles []string `json:"untracked_files"`
	ModifiedFiles  []string `json:"modified_files"`
	LastChecked    time.Time `json:"last_checked"`
	HasAnomalies   bool     `json:"has_anomalies"`
}

var (
	currentState GitState
	stateMutex   sync.RWMutex
)

func init() {
	// Initialize default state
	currentState = GitState{
		UntrackedFiles: make([]string, 0),
		ModifiedFiles:  make([]string, 0),
		LastChecked:    time.Now(),
		HasAnomalies:   false,
	}

	// Start the background monitor loop
	go monitorLoop()
}

// GetCurrentState safely returns a copy of the current state.
func GetCurrentState() GitState {
	stateMutex.RLock()
	defer stateMutex.RUnlock()
	return currentState
}

func monitorLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		checkGitStatus()
	}
}

func checkGitStatus() {
	cmd := exec.Command("git", "status", "--porcelain")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		// Log error in a real app, but for now we just skip updating if git fails
		return
	}

	output := out.String()
	lines := strings.Split(output, "\n")

	untracked := make([]string, 0)
	modified := make([]string, 0)
	hasAnomalies := false

	for _, line := range lines {
		if len(line) < 3 {
			continue
		}

		status := line[:2]
		file := strings.TrimSpace(line[3:])

		if status == "??" {
			untracked = append(untracked, file)
			hasAnomalies = true
		} else if status == " M" || status == "M " || status == "MM" {
			modified = append(modified, file)
			hasAnomalies = true
		} else if status == " A" || status == "A " {
			modified = append(modified, file)
			hasAnomalies = true
		}
	}

	stateMutex.Lock()
	currentState = GitState{
		UntrackedFiles: untracked,
		ModifiedFiles:  modified,
		LastChecked:    time.Now(),
		HasAnomalies:   hasAnomalies,
	}
	stateMutex.Unlock()
}

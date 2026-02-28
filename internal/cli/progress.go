package cli

import (
	"fmt"
	"os"
	"sync"
)

type progressState struct {
	done  int
	total int
}

type progressManager struct {
	mu      sync.Mutex
	order   []string
	states  map[string]progressState
	enabled bool
}

var outputMu sync.Mutex

var defaultProgressManager = &progressManager{
	order:   make([]string, 0, 16),
	states:  make(map[string]progressState, 16),
	enabled: isTerminal(os.Stdout),
}

// DownloadProgress renders a terminal progress indicator for one logical task.
type DownloadProgress struct {
	label string
}

// NewDownloadProgress creates a progress renderer for one logical download task.
func NewDownloadProgress(label string) *DownloadProgress {
	return &DownloadProgress{label: label}
}

// Update refreshes the in-place progress line.
//
// If total is not known, it leaves output unchanged.
func (p *DownloadProgress) Update(done, total int) {
	if total <= 0 {
		return
	}
	defaultProgressManager.update(p.label, done, total)
}

// Stop finalizes progress rendering by printing a trailing newline.
func (p *DownloadProgress) Stop() {
	defaultProgressManager.stop(p.label)
}

func (m *progressManager) update(label string, done, total int) {
	if !m.enabled {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.states[label]; !ok {
		m.order = append(m.order, label)
		m.states[label] = progressState{}
	}
	m.states[label] = progressState{done: done, total: total}
}

func (m *progressManager) stop(label string) {
	if !m.enabled {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.states[label]; !ok {
		return
	}
	state := m.states[label]
	delete(m.states, label)
	m.removeLabelLocked(label)

	if state.total <= 0 {
		return
	}
	if state.done < state.total {
		state.done = state.total
	}
	outputMu.Lock()
	defer outputMu.Unlock()
	fmt.Fprintf(os.Stdout, "%s: %d/%d\n", label, state.done, state.total)
}

func (m *progressManager) removeLabelLocked(label string) {
	for i, v := range m.order {
		if v != label {
			continue
		}
		m.order = append(m.order[:i], m.order[i+1:]...)
		return
	}
}

func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

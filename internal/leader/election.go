// Package leader provides a simple leader election mechanism for distributed
// systems. This implementation is a mock that simulates leader election
// for demonstration purposes.

package leader

import (
	"sync"
	"time"
)

// Election represents a leader election mechanism
type Election struct {
	mu       sync.RWMutex
	isLeader bool
}

// NewElection creates a new Election instance
func NewElection() *Election {
	e := &Election{}
	// Simulate leader election process
	go e.simulateElection()
	return e
}

// IsLeader returns the current leader status
func (e *Election) IsLeader() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.isLeader
}

// simulateElection simulates the leader election process
func (e *Election) simulateElection() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		e.mu.Lock()
		// Simulate 70% chance of being leader
		e.isLeader = time.Now().UnixNano()%100 < 70
		e.mu.Unlock()
	}
}

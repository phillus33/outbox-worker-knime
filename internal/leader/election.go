package leader

import (
	"sync"
	"time"
)

type Election struct {
	mu       sync.RWMutex
	isLeader bool
}

func NewElection() *Election {
	e := &Election{}
	// Simulate leader election process
	go e.simulateElection()
	return e
}

func (e *Election) IsLeader() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.isLeader
}

func (e *Election) simulateElection() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		e.mu.Lock()
		// Simulate 70% chance of being leader
		e.isLeader = time.Now().UnixNano()%100 < 70
		e.mu.Unlock()
	}
}

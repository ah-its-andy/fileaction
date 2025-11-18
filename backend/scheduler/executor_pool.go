package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/andi/fileaction/backend/database"
)

// ExecutorPool manages a pool of executors
type ExecutorPool struct {
	executors   []*Executor
	available   chan *Executor
	db          *database.DB
	logDir      string
	taskTimeout time.Duration
	stepTimeout time.Duration
	mu          sync.Mutex
	closed      bool
}

// NewExecutorPool creates a new executor pool
func NewExecutorPool(maxExecutors int, db *database.DB, logDir string, taskTimeout, stepTimeout time.Duration) *ExecutorPool {
	if maxExecutors <= 0 {
		maxExecutors = 2 // Default pool size
	}

	pool := &ExecutorPool{
		executors:   make([]*Executor, maxExecutors),
		available:   make(chan *Executor, maxExecutors),
		db:          db,
		logDir:      logDir,
		taskTimeout: taskTimeout,
		stepTimeout: stepTimeout,
		closed:      false,
	}

	// Create executors
	for i := 0; i < maxExecutors; i++ {
		executor := newExecutor(i+1, db, logDir, taskTimeout, stepTimeout)
		pool.executors[i] = executor
		pool.available <- executor
	}

	log.Printf("Executor pool created with %d executors", maxExecutors)
	return pool
}

// Acquire gets an available executor from the pool, blocking if none are available
func (p *ExecutorPool) Acquire(ctx context.Context) (*Executor, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, fmt.Errorf("executor pool is closed")
	}
	p.mu.Unlock()

	select {
	case executor := <-p.available:
		log.Printf("Executor-%d acquired from pool", executor.GetID())
		return executor, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Release returns an executor to the pool
func (p *ExecutorPool) Release(executor *Executor) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()

	log.Printf("Executor-%d released back to pool", executor.GetID())
	p.available <- executor
}

// GetPoolSize returns the total number of executors in the pool
func (p *ExecutorPool) GetPoolSize() int {
	return len(p.executors)
}

// GetAvailableCount returns the number of available executors
func (p *ExecutorPool) GetAvailableCount() int {
	return len(p.available)
}

// GetBusyCount returns the number of busy executors
func (p *ExecutorPool) GetBusyCount() int {
	return p.GetPoolSize() - p.GetAvailableCount()
}

// GetExecutorStatus returns the status of all executors
func (p *ExecutorPool) GetExecutorStatus() []ExecutorStatus {
	statuses := make([]ExecutorStatus, len(p.executors))
	for i, executor := range p.executors {
		statuses[i] = ExecutorStatus{
			ID:          executor.GetID(),
			Busy:        executor.IsBusy(),
			CurrentTask: executor.GetCurrentTask(),
		}
	}
	return statuses
}

// Close closes the executor pool
func (p *ExecutorPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	p.closed = true
	close(p.available)
	log.Println("Executor pool closed")
}

// ExecutorStatus represents the status of an executor
type ExecutorStatus struct {
	ID          int    `json:"id"`
	Busy        bool   `json:"busy"`
	CurrentTask string `json:"current_task,omitempty"`
}

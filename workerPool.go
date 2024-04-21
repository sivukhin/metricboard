package main

import (
	"context"
	"sync"
)

type work struct {
	f    func()
	done chan<- struct{}
}

type WorkerPool struct {
	sync.RWMutex
	size     int
	capacity int
	queries  chan work
	ctx      context.Context
	cancel   func()
}

func NewWorkerPool(ctx context.Context, size int, capacity int) *WorkerPool {
	ctx, cancel := context.WithCancel(ctx)
	return &WorkerPool{
		size:     size,
		capacity: capacity,
		queries:  make(chan work, capacity),
		ctx:      ctx,
		cancel:   cancel,
	}
}

func worker(queries <-chan work) {
	Logger.Infof("started worker")
	for query := range queries {
		query.f()
		close(query.done)
	}
	Logger.Infof("finished worker")
}

func (p *WorkerPool) Start() {
	for i := 0; i < p.size; i++ {
		go worker(p.queries)
	}
}

func (p *WorkerPool) Stop() {
	p.cancel()
	close(p.queries)
}

func (p *WorkerPool) Resize(size int) {
	if p.size == size {
		return
	}
	p.Lock()
	close(p.queries)
	previous := p.queries
	p.size = size
	p.queries = make(chan work, p.capacity)
	p.Unlock()

	p.Start()
	for query := range previous {
		p.queries <- query
	}
}

func (p *WorkerPool) Exec(f func(ctx context.Context)) {
	done := make(chan struct{})

	p.RLock()
	p.queries <- work{f: func() { f(p.ctx) }, done: done}
	p.RUnlock()

	<-done
}

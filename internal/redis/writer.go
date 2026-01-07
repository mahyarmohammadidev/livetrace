package redis

import (
	"log"
	"sync/atomic"
	"time"
)

type RedisWriter struct {
	store *RedisStore

	queue   chan Location
	workers int
	stopCh  chan struct{}

	enqueued atomic.Uint64
	dropped  atomic.Uint64
	written  atomic.Uint64
	errors   atomic.Uint64
}

type RedisWriterConfig struct {
	QueueSize int
	Workers   int
}

func NewRedisWriter(store *RedisStore, cfg RedisWriterConfig) *RedisWriter {
	w := &RedisWriter{
		store:   store,
		queue:   make(chan Location, cfg.QueueSize),
		workers: cfg.Workers,
		stopCh:  make(chan struct{}),
	}

	for i := 0; i < w.workers; i++ {
		go w.workerLoop(i)
	}

	return w
}

func (w *RedisWriter) Enqueue(loc Location) {
	select {
	case w.queue <- loc:
		w.enqueued.Add(1)
	default:
		w.dropped.Add(1)
	}
}

func (w *RedisWriter) workerLoop(workerID int) {
	for {
		select {
		case loc := <-w.queue:
			if err := w.store.SaveLocation(loc); err != nil {
				w.errors.Add(1)
				log.Printf("[redis-worker-%d] save error: %v", workerID, err)
				time.Sleep(20 * time.Millisecond)
				continue
			}
			w.written.Add(1)

		case <-w.stopCh:
			return
		}
	}
}

func (w *RedisWriter) Shutdown() {
	close(w.stopCh)
}

func (w *RedisWriter) Stats() (enqueued, dropped, written, errors uint64) {
	return w.enqueued.Load(), w.dropped.Load(), w.written.Load(), w.errors.Load()
}

package entities

import (
	"log"
	"sync/atomic"
)

type worker struct {
	id      int
	queue   Queue
	running atomic.Bool
	suspend chan struct{}
	stop    chan struct{}
}

type Worker interface {
	Busy() bool
	Toggle()
	Stop()
}

func NewWorker(id int, queue Queue) Worker {
	w := &worker{
		id:      id,
		queue:   queue,
		suspend: make(chan struct{}),
		stop:    make(chan struct{}),
	}
	go func() {
		var currentTask Task
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Worker %d panicked: %v", w.id, r)
			}
			w.running.Store(false)
		}()
		for {
			select {
			case <-w.suspend:
				<-w.suspend
			case <-w.stop:
				return
			case currentTask = <-w.queue.Pull():
				select {
				case <-currentTask.Timeout():
					continue
				default:
					w.running.Store(true)
					currentTask.Exec()
					w.running.Store(false)
				}
			}

		}
	}()
	return w
}

func (w *worker) Busy() bool {
	return w.running.Load()
}

func (w *worker) Toggle() {
	w.suspend <- struct{}{}
	return
}

func (w *worker) Stop() {
	close(w.stop)
}

package entities

import (
	"sync"
	"time"
)

type Queue interface {
	Push(Task)
	Pause()
	Resume()
	Pull() chan Task
	Size() int
	WaitUntilEmpty()
}

type queue struct {
	tasks chan Task
	mutex *sync.Mutex
}

func NewQueue(size int) Queue {
	return &queue{
		tasks: make(chan Task, size),
		mutex: new(sync.Mutex),
	}
}

func (q queue) WaitUntilEmpty() {
	for q.Size() > 0 {
		time.Sleep(100 * time.Millisecond)
		continue
	}
}

func (q queue) Pause() {
	q.mutex.Lock()
}

func (q queue) Resume() {
	q.mutex.Unlock()
}

func (q queue) Push(task Task) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	select {
	case <-task.Timeout():
		return
	default:
		q.tasks <- task
	}
	return
}

func (q queue) Pull() chan Task {
	return q.tasks
}

func (q queue) Size() int {
	return len(q.tasks)
}

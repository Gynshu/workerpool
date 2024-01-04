package entities

import (
	"context"
)

type Task interface {
	// WaitUntilDone blocks until the task's execution is finished.
	WaitUntilDone() <-chan struct{}
	Timeout() <-chan struct{}
	Exec()
}

type task struct {
	timeout context.Context
	exec    func()
	done    chan struct{}
}

func NewTask(ctx context.Context, job func()) Task {
	if ctx == nil {
		ctx = context.Background()
	}
	t := &task{
		timeout: ctx,
		exec:    job,
		done:    make(chan struct{}),
	}
	return t
}

func (t *task) Timeout() <-chan struct{} {
	return t.timeout.Done()
}
func (t *task) Exec() {
	t.exec()
	close(t.done)
}

func (t *task) WaitUntilDone() <-chan struct{} {
	return t.done
}

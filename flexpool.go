// Package workerpool is a worker pool that can execute tasks concurrently.
// Features:
//   - Can pause the queue and resume without breaking the whole pool.
//   - Can allocate additional workers for priority tasks.
//     Those tasks will pass the queue and will be executed immediately if any "priority" worker is available.
//   - Can push tasks with context.
//     If the context is timed out or canceled before a worker is available, the task will be discarded.
//   - Can push tasks with context and wait until it's done.
//   - Can Limit the size of the queue.
//
// Also, it has a DangerWait() function that will wait until the queue is empty and all workers are idle.
// It is useful when you have multiple loops in goroutine pushing tasks to the queue, and you don't know when the last task will be pushed.
package workerpool

import (
	"context"
	"github.com/gynshu/workerpool/entities"
	"github.com/gynshu/workerpool/workers"
	"log"
	"time"
)

// Pool is a worker pool that can execute tasks concurrently. The main reason for creating this pool is to be able to pause the queue and resume without breaking the whole pool. And maybe the most used feature for me is the ability to allocate workers for priorityPool tasks.
// Those tasks will pass the queue and will be executed immediately if any "priority" worker is available.
// You can set the maximum number of workers as well as the maximum size of the queue. It has a queue that can hold a maximum number of tasks.
//
// If the queue is full, new tasks will be blocked until a queue slot is available.
// If a task is pushed to the queue with a context, the task will be discarded if the context is timed out or canceled before a worker is available.
type Pool interface {
	// Push pushes a task to the queue
	// ctx is the context for the task, a timeout or deadline is optional
	// This context used to restrict waiting time in queue for the task to be executed.
	// After the deadline or timeout, the task will be discarded. The function will return.
	//
	// If no ctx is provided, the task will be pushed to the queue and will wait until a worker is available
	// This function blocks goroutine until a worker is available, but not until the task is done.
	// Priority
	//
	// if urgent is true, the task will be pushed to the priority workers.
	// This function will send a task to regular pool if priorityPool is not set and priorityTask is true.
	Push(ctx context.Context, urgent bool, task func())

	// PushWait adds a job to the queue and waits until the job is completed, then returns
	//
	// if urgent is true, the task will be pushed to the priority workers.
	// This function will send a task to regular pool if priorityPool is not set and priorityTask is true.
	PushWait(ctx context.Context, urgent bool, task func())

	// WaitForAll Block until all tasks in the queue are executed, including urgent tasks.
	// You will not be able to push any new task, since PauseQueue is called.
	// Any new Push, PushWait etc. will block until all tasks in queue are executed.
	WaitForAll()

	// WaitOnlyForRunningTasks Block until all tasks on running workers are finished, including urgent tasks.
	// You are able to push new tasks to the queue.
	WaitOnlyForRunningTasks()

	// DangerWait Wait until the queue is empty. Once it is empty, it will wait for workers to finish tasks.
	// But it does not Pause Queue or Workers, all incoming tasks will be executed.
	// Warning: This function may block forever if new tasks never end.
	DangerWait()

	// PauseQueue pauses the queue. You will not be able to push any new task.
	// Any new Push, PushWait etc. will block until the queue is resumed, including urgent tasks.
	// If you use context.Context, you can pass its' Deadline() as time parameter.
	PauseQueue(until time.Time)

	// PauseWorkers pauses all workers.
	//  Workers will finish their current task and will not start a new one, including urgent workers.
	// You are able to push new tasks to the queue until the queue is full.
	// If you use context.Context, you can pass its' Deadline() as time parameter.
	PauseWorkers(until time.Time)

	// The Report returns a report about the current state of the pool
	// Such as queue size, running workers, etc.
	Report() entities.Report
}

type pool struct {
	maxQueueSize    int
	workers         workers.Manager
	queue           entities.Queue
	priorityWorkers workers.Manager
	priorityQueue   entities.Queue
	report          entities.Report
}

// New creates a new worker pool
// maxQueueSize is the maximum size of the queue.
// maxWorkers is the maximum number of "regular" workers.
// priorityWorkers is the number of workers that will be allocated additionally for urgent tasks.
// These workers will not be used for regular tasks.
// E.g., the total number of workers will be maxWorkers + priorityPool.
// They will not be used for regular tasks.
// For more information, see Pool interface
func New(maxQueueSize, regularWorkers, priorityWorkers int) Pool {
	if regularWorkers <= 0 {
		log.Println("maxWorkers is set to 0, resetting to 1")
		regularWorkers = 1
	}
	if maxQueueSize < 0 {
		log.Println("maxQueueSize is negative, resetting to 0")
		maxQueueSize = 0
	}

	regular := make(map[int]entities.Worker, regularWorkers)
	queue := entities.NewQueue(maxQueueSize)
	// initialize workers
	for i := 0; i < regularWorkers; i++ {
		regular[i] = entities.NewWorker(i, queue)
	}

	var priority map[int]entities.Worker
	p := &pool{
		maxQueueSize: maxQueueSize,
		workers:      workers.NewManager(regular),
		queue:        queue,
	}
	if priorityWorkers > 0 {
		priority = make(map[int]entities.Worker, priorityWorkers)
		priorityQueue := entities.NewQueue(0)
		// initialize priorityPool workers
		for i := 0; i < priorityWorkers; i++ {
			priority[i] = entities.NewWorker(i, priorityQueue)
		}
		p.priorityWorkers = workers.NewManager(priority)
		p.priorityQueue = priorityQueue
	}
	return p
}

func (p *pool) Push(ctx context.Context, urgent bool, job func()) {
	task := entities.NewTask(ctx, job)
	p.distribute(urgent, task)
}
func (p *pool) PushWait(ctx context.Context, urgent bool, job func()) {
	task := entities.NewTask(ctx, job)
	p.distribute(urgent, task)
	select {
	case <-task.Timeout():
		return
	case <-task.WaitUntilDone():
		return
	}
}

func (p *pool) distribute(urgent bool, task entities.Task) {
	if urgent {
		if p.priorityWorkers != nil {
			p.priorityQueue.Push(task)
		} else {
			p.queue.Push(task)
		}
	} else {
		p.queue.Push(task)
	}
}

func (p *pool) WaitOnlyForRunningTasks() {
	// Pause workers
	// Since We are not able to stop already running tasks, wi just wait until all running tasks are finished
	p.workers.PauseAll()
	// wait for running tasks to finish
	defer p.workers.ResumeAll()
	if p.priorityWorkers != nil {
		p.priorityWorkers.PauseAll()
		defer p.priorityWorkers.ResumeAll()
	}
}

func (p *pool) WaitForAll() {
	// Regular
	p.queue.Pause()
	defer p.queue.Resume()
	p.queue.WaitUntilEmpty()

	// Priority
	if p.priorityWorkers != nil {
		p.priorityQueue.Pause()
		defer p.priorityQueue.Resume()
		p.priorityQueue.WaitUntilEmpty()
	}

	// Wait for running tasks to finish
	p.WaitOnlyForRunningTasks()
}

func (p *pool) DangerWait() {
	p.queue.WaitUntilEmpty()
	if p.priorityWorkers != nil {
		p.priorityQueue.WaitUntilEmpty()
	}
	p.WaitOnlyForRunningTasks()
}

func (p *pool) PauseQueue(until time.Time) {
	p.queue.Pause()
	if p.priorityWorkers != nil {
		p.priorityQueue.Pause()
	}
	time.AfterFunc(time.Until(until), func() {
		p.queue.Resume()
		if p.priorityWorkers != nil {
			p.priorityQueue.Resume()
		}
	})
}

func (p *pool) PauseWorkers(until time.Time) {
	p.workers.PauseAll()
	if p.priorityWorkers != nil {
		p.priorityWorkers.PauseAll()
	}
	time.AfterFunc(time.Until(until), func() {
		p.workers.ResumeAll()
		if p.priorityWorkers != nil {
			p.priorityWorkers.ResumeAll()
		}
	})
}

func (p *pool) Report() entities.Report {
	running, idle := p.workers.Report()
	p.report.TasksInQueue = p.queue.Size()
	p.report.ExecutingRegularWorkers = running
	p.report.IdleRegularWorkers = idle
	if p.priorityWorkers != nil {
		p.report.TasksInQueue += p.priorityQueue.Size()
		pRunning, pIdle := p.priorityWorkers.Report()
		p.report.ExecutingPriorityWorkers = pRunning
		p.report.IdlePriorityWorkers = pIdle
	}
	return p.report
}

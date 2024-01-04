## workerpool

Package workerpool is a worker pool that can execute tasks concurrently.
Features:
  - Can pause the queue and resume without breaking the whole pool.
  - Can allocate additional workers for priority tasks.
    Those tasks will pass the queue and will be executed immediately if any "priority" worker is available.
  - Can push tasks with context.
    If the context is timed out or canceled before a worker is available, the task will be discarded.
  - Can push tasks with context and wait until it's done.
  - Can Limit the size of the queue.

Also, it has a DangerWait() function that will wait until the queue is empty and all workers are idle.
It is useful when you have multiple loops in goroutine pushing tasks to the queue, and you don't know when the last task will be pushed to the limited queue.
## Install

```bash
go get -u https://github.com/gynshu/workerpool@latest
```

## Usage

```
New(maxQueueSize, regularWorkers, priorityWorkers int) Pool
```
New creates a new worker pool **maxQueueSize** is the maximum size of the queue.
**maxWorkers** is the maximum number of "regular" workers. **priorityWorkers** is the
number of workers that will be allocated **additionally** for urgent tasks. They will not be used
for regular tasks. For more
information, see Pool interface


```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gynshu/workerpool"
)

func main() {
	// Queue 100, 5 regular workers, 1 urgent worker
	// if we set queue size to 0, all tasks will be
	// passed to workers directly
	wp := workerpool.New(100, 5, 1)
	someSlice := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	for item := range someSlice {
		k := item
		// Push regular tasks with delay
		wp.Push(nil, false, func() {
			time.Sleep(100 * time.Millisecond)
			fmt.Println("regular", k)
		})
	}
	time.Sleep(100 * time.Millisecond)
	// This one will exec immediately, bec out urgent worker is idle all the time
	wp.Push(nil, true, func() {
		fmt.Println("urgent")
	})

	// Push another and wait
	wp.PushWait(nil, false, func() {
		fmt.Println("another regular")
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	wp.PushWait(ctx, true, func() {
		fmt.Println("will print")
		// bec queue is empty and workers are idle
		// task will not be executed if the context is canceled by the time
		// it is delivered to the worker
	})
	time.Sleep(2 * time.Second)
	wp.PushWait(ctx, false, func() {
		fmt.Println("this one will not print")
		// bec the context is canceled by the time it is delivered to the worker
	})
	wp.PushWait(ctx, true, func() {
		fmt.Println("this too")
		// bec the context is canceled by the time it is delivered to the worker
	})
	wp.WaitForAll()
}

```

## Interface


```go
// Pool is a worker pool that can execute tasks concurrently. The main reason for creating this pool is to be able to pause the queue and resume without breaking the whole pool. And maybe the most used feature for me is the ability to allocate workers for priorityPool tasks.
// Those tasks will pass the queue and will be executed immediately if an "priorityPool" worker is available.
// You can set the maximum number of workers as well as the maximum size of the queue. It has a queue that can hold a maximum number of tasks.
//
// If the queue is full, new tasks will be blocked until a worker is available.
// If a task is pushed to the queue with a context, the task will be discarded if the context is timed out or canceled. Before a worker is available.
// If a task is pushed to the queue without a context, the task will be queued until a worker is available.
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
```

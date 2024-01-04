package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gynshu/workerpool"
)

func main() {
	// 100 queue, 5 regular workers, 1 urgent worker
	wp := workerpool.New(100, 5, 0)
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

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	wp.PushWait(ctx, true, func() {
		fmt.Println("will print")
		// bec queue is empty and workers are idle
		// task will not be executed if the context is canceled by the time
		// it is delivered to the worker
	})
	time.Sleep(110 * time.Millisecond)
	wp.PushWait(ctx, false, func() {
		fmt.Println("this one will not print")
		// bec the context is canceled by the time it is delivered to the worker
	})
	wp.PushWait(ctx, true, func() {
		fmt.Println("this too")
		// bec the context is canceled by the time it is delivered to the worker
	})
	wp.WaitForAll()
	wp.Report()
}

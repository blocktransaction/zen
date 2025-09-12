package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/blocktransaction/zen/internal/retryx"
)

func main() {
	pool := retryx.NewPool[string](5) // 5 workers
	defer pool.Close()

	var futures []*retryx.Future[string]
	// taskCount := 20

	// Define a reusable retrier configuration for our API calls
	apiRetrier := retryx.NewRetrier[string](
		retryx.WithMaxRetries[string](4),
		retryx.WithInitialDelay[string](100*time.Millisecond),
	)

	// Submit 20 tasks to the pool
	// for i := 0; i < taskCount; i++ {
	// taskID := i // Capture loop variable
	i := 0
	task := retryx.Task[string]{
		Fn: func() (string, error) {
			fmt.Println("----", i)
			i += 1
			return "test", errors.New("fail")
		},
		Retrier: apiRetrier, // Use our custom retrier
	}
	future := pool.Submit(task)
	futures = append(futures, future)
	// }

	log.Println("All tasks submitted. Waiting for results...")

	// Wait for and process the results
	for i, future := range futures {
		value, err := future.Get()
		if err != nil {
			log.Printf("Result for task %d: FAILED -> %v", i, err)
		} else {
			log.Printf("Result for task %d: SUCCESS -> %s", i, value)
		}
	}

	log.Println("All tasks completed. Pool is closing.")
}

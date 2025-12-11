package retryx

import (
	"errors"
	"fmt"
	"log"
	"testing"
	"time"
)

func TestRetryx(t *testing.T) {
	pool := NewPool[string](5) // 5 workers
	defer pool.Close()

	var futures []*Future[string]
	// taskCount := 20

	// Define a reusable retrier configuration for our API calls
	apiRetrier := NewRetrier(
		WithMaxRetries[string](3),
		WithInitialDelay[string](10*time.Millisecond),
	)

	i := 0
	task := Task[string]{
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

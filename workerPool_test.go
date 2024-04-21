package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestName(t *testing.T) {
	var kek chan int
	select {
	case <-kek:
		t.Log("kek")
	}
}

func TestWorkerPool(t *testing.T) {
	t.Run("1 worker", func(t *testing.T) {
		results := make(chan int, 3)
		p := NewWorkerPool(context.Background(), 1, 1024)
		p.Start()
		startTime := time.Now()
		go p.Exec(func(ctx context.Context) {
			time.Sleep(1 * time.Second)
			results <- 1
		})
		go p.Exec(func(ctx context.Context) {
			time.Sleep(1 * time.Second)
			results <- 2
		})
		go p.Exec(func(ctx context.Context) {
			time.Sleep(1 * time.Second)
			results <- 3
		})
		sum := 0
		for i := 0; i < 3; i++ {
			sum += <-results
		}
		require.Greater(t, time.Since(startTime), 2*time.Second)
		require.Less(t, time.Since(startTime), 4*time.Second)
		require.Equal(t, 1+2+3, sum)
	})
	t.Run("3 worker", func(t *testing.T) {
		results := make(chan int, 3)
		p := NewWorkerPool(context.Background(), 3, 1024)
		p.Start()
		startTime := time.Now()
		go p.Exec(func(ctx context.Context) {
			time.Sleep(1 * time.Second)
			results <- 1
		})
		go p.Exec(func(ctx context.Context) {
			time.Sleep(1 * time.Second)
			results <- 2
		})
		go p.Exec(func(ctx context.Context) {
			time.Sleep(1 * time.Second)
			results <- 3
		})
		sum := 0
		for i := 0; i < 3; i++ {
			sum += <-results
		}
		require.Greater(t, time.Since(startTime), 1*time.Second)
		require.Less(t, time.Since(startTime), 2*time.Second)
		require.Equal(t, 1+2+3, sum)
	})
	t.Run("resize", func(t *testing.T) {
		results := make(chan int, 16)
		p := NewWorkerPool(context.Background(), 1, 1024)
		p.Start()
		startTime := time.Now()
		for i := 0; i < 16; i++ {
			value := i
			go p.Exec(func(ctx context.Context) {
				time.Sleep(1 * time.Second)
				results <- value
			})
		}
		time.Sleep(1 * time.Second)
		p.Resize(16)

		sum := 0
		for i := 0; i < 16; i++ {
			sum += <-results
		}
		require.Greater(t, time.Since(startTime), 2*time.Second)
		require.Less(t, time.Since(startTime), 4*time.Second)
		require.Equal(t, 16*15/2, sum)
	})
}

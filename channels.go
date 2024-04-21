package main

import (
	"context"
)

func NewStreamingWriter[T any](ctx context.Context, capacity int, write func(T)) chan<- T {
	result := make(chan T, capacity)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case item, ok := <-result:
				if !ok {
					return
				}
				write(item)
			}
		}
	}()
	return result
}

func NewStreamingReader[T any](ctx context.Context, capacity int, read func() (T, error)) <-chan T {
	result := make(chan T, capacity)
	go func() {
		defer close(result)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				item, err := read()
				if err != nil {
					return
				}
				result <- item
			}
		}
	}()
	return result
}

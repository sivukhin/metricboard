package main

import (
	"context"
)

func CombineContexts(a, b context.Context) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-a.Done():
			cancel()
		case <-b.Done():
			cancel()
		}
	}()
	return ctx
}

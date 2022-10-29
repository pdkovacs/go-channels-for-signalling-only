package strsli_to_strsli

import (
	"context"
	"sync"
	"sync/atomic"
)

type doSomethingFunc func(string) string

func process(ctx context.Context, doSomethig doSomethingFunc, workCount int, intputs []string) []string {
	canceled := atomic.Bool{}
	currentIndex := atomic.Int64{}

	canceled.Store(false)
	currentIndex.Store(-1)

	completed := make(chan struct{}, 1)
	defer func() {
		completed <- struct{}{}
	}()
	go func() {
		select {
		case <-ctx.Done():
			canceled.Store(true)
		case <-completed:
		}
	}()

	outputs := make([]string, len(intputs))
	wgWorker := sync.WaitGroup{}
	for i := 0; i < workCount; i++ {
		wgWorker.Add(1)
		go func() {
			defer wgWorker.Done()
			nextIndex := currentIndex.Add(1)

			for nextIndex < int64(len(intputs)) && !canceled.Load() {
				outputs[nextIndex] = doSomethig(intputs[nextIndex])
				nextIndex = currentIndex.Add(1)
			}
		}()
	}
	wgWorker.Wait()

	if canceled.Load() {
		return nil
	}

	return outputs
}

package main

import (
	"context"
	"errors"
	"sync"
)

type Result[R any] struct {
	Value R
	Err   error
}

var ErrInvalidWorkers = errors.New("workers must be positive")

func ParallelMap[T any, R any](ctx context.Context, workers int, in []T, fn func(context.Context, T) (R, error)) ([]Result[R], error) {

	if workers <= 0 {
		return nil, ErrInvalidWorkers
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if len(in) == 0 {
		return []Result[R]{}, nil
	}

	var wg sync.WaitGroup

	results := make([]Result[R], len(in))
	indexesChan := make(chan int, len(in))

	for i := range in {
		indexesChan <- i
	}
	close(indexesChan)

	goroNum := min(workers, len(in))

	for iW := 0; iW < goroNum; iW++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range indexesChan {
				res := Result[R]{}
				if ctx.Err() != nil {
					return
				}

				val, errFn := fn(ctx, in[i])
				if errFn == nil {
					res.Value = val
				}
				res.Err = errFn
				results[i] = res
			}
		}()

	}

	wg.Wait()

	if errCtxFinal := ctx.Err(); errCtxFinal != nil {
		return nil, errCtxFinal
	}

	return results, nil
}

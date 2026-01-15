package parallel

import (
	"context"
	"runtime"
	"sync"
)

// FileResult holds the result of processing a single file
type FileResult struct {
	Filename string
	Result   interface{}
	Error    error
}

// ProcessFunc is the function type for processing a single file
type ProcessFunc func(ctx context.Context, filename string) (interface{}, error)

// Processor handles parallel file processing
type Processor struct {
	workers      int
	preserveOrder bool
}

// Option configures a Processor
type Option func(*Processor)

// New creates a new Processor with the given options
func New(opts ...Option) *Processor {
	p := &Processor{
		workers:      runtime.GOMAXPROCS(0),
		preserveOrder: true,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// WithWorkers sets the number of workers
func WithWorkers(n int) Option {
	return func(p *Processor) {
		if n > 0 {
			p.workers = n
		}
	}
}

// WithPreserveOrder sets whether to preserve input order in results
func WithPreserveOrder(preserve bool) Option {
	return func(p *Processor) {
		p.preserveOrder = preserve
	}
}

// Process processes multiple files in parallel
func (p *Processor) Process(ctx context.Context, files []string, fn ProcessFunc) []FileResult {
	if len(files) == 0 {
		return nil
	}

	numWorkers := p.workers
	if numWorkers > len(files) {
		numWorkers = len(files)
	}

	// Create job channel
	type job struct {
		index    int
		filename string
	}
	jobs := make(chan job, len(files))
	for i, f := range files {
		jobs <- job{index: i, filename: f}
	}
	close(jobs)

	// Create result channel
	resultsChan := make(chan struct {
		index  int
		result FileResult
	}, len(files))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				select {
				case <-ctx.Done():
					resultsChan <- struct {
						index  int
						result FileResult
					}{
						index: j.index,
						result: FileResult{
							Filename: j.filename,
							Error:    ctx.Err(),
						},
					}
				default:
					result, err := fn(ctx, j.filename)
					resultsChan <- struct {
						index  int
						result FileResult
					}{
						index: j.index,
						result: FileResult{
							Filename: j.filename,
							Result:   result,
							Error:    err,
						},
					}
				}
			}
		}()
	}

	// Wait for workers and close results channel
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	results := make([]FileResult, len(files))
	for r := range resultsChan {
		results[r.index] = r.result
	}

	return results
}

// AggregateError collects multiple errors from parallel processing
type AggregateError struct {
	Errors []error
}

// Error implements the error interface
func (e *AggregateError) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return e.Errors[0].Error() + " (and more errors)"
}

// HasErrors returns true if there are any errors
func (e *AggregateError) HasErrors() bool {
	return len(e.Errors) > 0
}

// CollectErrors extracts errors from file results
func CollectErrors(results []FileResult) *AggregateError {
	var errors []error
	for _, r := range results {
		if r.Error != nil {
			errors = append(errors, r.Error)
		}
	}
	return &AggregateError{Errors: errors}
}

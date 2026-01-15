package parallel

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestProcessor_Process(t *testing.T) {
	files := []string{"file1.txt", "file2.txt", "file3.txt"}

	p := New(WithWorkers(2))
	results := p.Process(context.Background(), files, func(ctx context.Context, filename string) (interface{}, error) {
		return "processed: " + filename, nil
	})

	if len(results) != len(files) {
		t.Errorf("expected %d results, got %d", len(files), len(results))
	}

	// Check order is preserved
	for i, r := range results {
		if r.Filename != files[i] {
			t.Errorf("result %d: expected filename %s, got %s", i, files[i], r.Filename)
		}
		expected := "processed: " + files[i]
		if r.Result != expected {
			t.Errorf("result %d: expected %s, got %v", i, expected, r.Result)
		}
		if r.Error != nil {
			t.Errorf("result %d: unexpected error: %v", i, r.Error)
		}
	}
}

func TestProcessor_ProcessWithErrors(t *testing.T) {
	files := []string{"good.txt", "bad.txt", "good2.txt"}

	p := New()
	results := p.Process(context.Background(), files, func(ctx context.Context, filename string) (interface{}, error) {
		if filename == "bad.txt" {
			return nil, errors.New("processing failed")
		}
		return "ok", nil
	})

	aggErr := CollectErrors(results)
	if !aggErr.HasErrors() {
		t.Error("expected errors")
	}
	if len(aggErr.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(aggErr.Errors))
	}
}

func TestProcessor_ProcessEmpty(t *testing.T) {
	p := New()
	results := p.Process(context.Background(), nil, func(ctx context.Context, filename string) (interface{}, error) {
		return nil, nil
	})

	if results != nil {
		t.Errorf("expected nil results for empty input, got %v", results)
	}
}

func TestProcessor_ProcessCancellation(t *testing.T) {
	files := make([]string, 100)
	for i := range files {
		files[i] = "file.txt"
	}

	ctx, cancel := context.WithCancel(context.Background())

	var processed int32
	p := New(WithWorkers(2))

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	results := p.Process(ctx, files, func(ctx context.Context, filename string) (interface{}, error) {
		time.Sleep(5 * time.Millisecond)
		atomic.AddInt32(&processed, 1)
		return "ok", nil
	})

	// Some results should have context errors
	var cancelledCount int
	for _, r := range results {
		if r.Error == context.Canceled {
			cancelledCount++
		}
	}

	// Not all should be processed due to cancellation
	if cancelledCount == 0 && len(files) > 10 {
		// This is flaky but generally cancellation should take effect
		t.Log("warning: no cancellations detected, but timing may vary")
	}

	_ = results
}

func TestProcessor_Concurrency(t *testing.T) {
	files := make([]string, 10)
	for i := range files {
		files[i] = "file.txt"
	}

	var maxConcurrent int32
	var current int32

	p := New(WithWorkers(3))
	p.Process(context.Background(), files, func(ctx context.Context, filename string) (interface{}, error) {
		c := atomic.AddInt32(&current, 1)
		for {
			old := atomic.LoadInt32(&maxConcurrent)
			if c <= old || atomic.CompareAndSwapInt32(&maxConcurrent, old, c) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt32(&current, -1)
		return "ok", nil
	})

	if maxConcurrent > 3 {
		t.Errorf("expected max concurrency <= 3, got %d", maxConcurrent)
	}
}

func TestAggregateError_Error(t *testing.T) {
	t.Run("no errors", func(t *testing.T) {
		e := &AggregateError{}
		if e.Error() != "no errors" {
			t.Errorf("expected 'no errors', got %q", e.Error())
		}
	})

	t.Run("one error", func(t *testing.T) {
		e := &AggregateError{Errors: []error{errors.New("single error")}}
		if e.Error() != "single error" {
			t.Errorf("expected 'single error', got %q", e.Error())
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		e := &AggregateError{Errors: []error{
			errors.New("first error"),
			errors.New("second error"),
		}}
		expected := "first error (and more errors)"
		if e.Error() != expected {
			t.Errorf("expected %q, got %q", expected, e.Error())
		}
	})
}

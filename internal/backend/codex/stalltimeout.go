package codex

import (
	"context"
	"io"
	"sync"
	"time"
)

// stallTimeoutReader wraps an io.Reader with per-read stall timeout and
// total timeout detection. This is the Go-idiomatic equivalent of the
// Python socket timeout hack.
type stallTimeoutReader struct {
	reader       io.Reader
	ctx          context.Context
	cancel       context.CancelFunc
	stallTimeout time.Duration
	totalTimeout time.Duration
	startTime    time.Time
	lastPhase    string
	err          error
	totalExpired bool
	mu           sync.Mutex
}

func newStallTimeoutReader(
	parentCtx context.Context,
	reader io.Reader,
	stallTimeout, totalTimeout time.Duration,
) *stallTimeoutReader {
	ctx, cancel := context.WithCancel(parentCtx)
	return &stallTimeoutReader{
		reader:       reader,
		ctx:          ctx,
		cancel:       cancel,
		stallTimeout: stallTimeout,
		totalTimeout: totalTimeout,
		startTime:    time.Now(),
	}
}

func (s *stallTimeoutReader) Read(p []byte) (int, error) {
	s.mu.Lock()
	if s.err != nil {
		s.mu.Unlock()
		return 0, s.err
	}
	s.mu.Unlock()

	// Check total timeout
	elapsed := time.Since(s.startTime)
	if elapsed >= s.totalTimeout {
		s.mu.Lock()
		s.err = context.DeadlineExceeded
		s.totalExpired = true
		s.mu.Unlock()
		s.cancel()
		return 0, io.EOF
	}

	// Calculate effective stall timeout (clamped to remaining budget)
	remaining := s.totalTimeout - elapsed
	stallT := s.stallTimeout
	if stallT > remaining {
		stallT = remaining
	}

	// Use a goroutine + timer for per-read timeout
	type readResult struct {
		n   int
		err error
	}
	ch := make(chan readResult, 1)

	go func() {
		n, err := s.reader.Read(p)
		ch <- readResult{n, err}
	}()

	timer := time.NewTimer(stallT)
	defer timer.Stop()

	select {
	case res := <-ch:
		return res.n, res.err
	case <-timer.C:
		// Check if total also expired
		s.mu.Lock()
		if time.Since(s.startTime) >= s.totalTimeout {
			s.totalExpired = true
		}
		s.err = context.DeadlineExceeded
		s.mu.Unlock()
		s.cancel()
		return 0, io.EOF
	case <-s.ctx.Done():
		s.mu.Lock()
		s.err = s.ctx.Err()
		s.mu.Unlock()
		return 0, io.EOF
	}
}

func (s *stallTimeoutReader) setPhase(phase string) {
	s.mu.Lock()
	s.lastPhase = phase
	s.mu.Unlock()
}

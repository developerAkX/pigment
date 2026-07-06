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
	closer       io.Closer // closed on timeout to unblock the reading goroutine
	ctx          context.Context
	cancel       context.CancelFunc
	stallTimeout time.Duration
	totalTimeout time.Duration
	startTime    time.Time
	lastPhase    string
	err          error
	totalExpired bool
	closeOnce    sync.Once
	mu           sync.Mutex
}

// newStallTimeoutReader wraps reader with stall/total timeout detection.
// closer (may be nil) is closed when a timeout fires so that the blocked
// underlying Read returns and its goroutine exits instead of leaking.
func newStallTimeoutReader(
	parentCtx context.Context,
	reader io.Reader,
	closer io.Closer,
	stallTimeout, totalTimeout time.Duration,
) *stallTimeoutReader {
	ctx, cancel := context.WithCancel(parentCtx)
	return &stallTimeoutReader{
		reader:       reader,
		closer:       closer,
		ctx:          ctx,
		cancel:       cancel,
		stallTimeout: stallTimeout,
		totalTimeout: totalTimeout,
		startTime:    time.Now(),
	}
}

// abort closes the underlying body (once) to unblock any in-flight Read.
func (s *stallTimeoutReader) abort() {
	s.cancel()
	if s.closer != nil {
		s.closeOnce.Do(func() { _ = s.closer.Close() })
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
		s.abort()
		return 0, io.EOF
	}

	// Calculate effective stall timeout (clamped to remaining budget)
	remaining := s.totalTimeout - elapsed
	stallT := s.stallTimeout
	if stallT > remaining {
		stallT = remaining
	}

	// Use a goroutine + timer for per-read timeout. The goroutine reads
	// into its own buffer (not p): if we time out and return, the caller
	// may reuse p, and writing into it from the goroutine would race.
	type readResult struct {
		buf []byte
		err error
	}
	ch := make(chan readResult, 1)

	go func() {
		buf := make([]byte, len(p))
		n, err := s.reader.Read(buf)
		ch <- readResult{buf[:n], err}
	}()

	timer := time.NewTimer(stallT)
	defer timer.Stop()

	select {
	case res := <-ch:
		return copy(p, res.buf), res.err
	case <-timer.C:
		// Check if total also expired
		s.mu.Lock()
		if time.Since(s.startTime) >= s.totalTimeout {
			s.totalExpired = true
		}
		s.err = context.DeadlineExceeded
		s.mu.Unlock()
		s.abort()
		return 0, io.EOF
	case <-s.ctx.Done():
		s.mu.Lock()
		s.err = s.ctx.Err()
		s.mu.Unlock()
		s.abort()
		return 0, io.EOF
	}
}

// state returns a consistent snapshot of the reader's error status.
func (s *stallTimeoutReader) state() (err error, lastPhase string, totalExpired bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err, s.lastPhase, s.totalExpired
}

func (s *stallTimeoutReader) setPhase(phase string) {
	s.mu.Lock()
	s.lastPhase = phase
	s.mu.Unlock()
}

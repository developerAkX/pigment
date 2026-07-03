package config

import (
	"context"
	"encoding/json"
	"os"
	"time"
)

// lockInfo is written into each lock file for stale detection.
type lockInfo struct {
	PID       int   `json:"pid"`
	Timestamp int64 `json:"ts"`
}

// SlotLock represents an acquired concurrency slot.
type SlotLock struct {
	path string
}

// Release releases the concurrency slot.
func (s *SlotLock) Release() {
	if s == nil {
		return
	}
	os.Remove(s.path)
}

// AcquireSlot acquires a concurrency slot, blocking until one is available.
// If limit <= 0, returns nil (unlimited). The onWait callback is called once
// when the first poll wait is needed.
func AcquireSlot(ctx context.Context, backend string, limit int, onWait func()) (*SlotLock, error) {
	if limit <= 0 {
		return nil, nil
	}

	waited := false
	for {
		for i := 0; i < limit; i++ {
			p := LockSlotPath(backend, i)
			if tryAcquire(p) {
				return &SlotLock{path: p}, nil
			}
		}
		if !waited && onWait != nil {
			onWait()
			waited = true
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

const staleDuration = 10 * time.Minute

func tryAcquire(path string) bool {
	// Try O_EXCL create
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err == nil {
		writeLockInfo(f)
		f.Close()
		return true
	}
	// File exists — check for staleness
	if isStale(path) {
		os.Remove(path)
		f, err = os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			writeLockInfo(f)
			f.Close()
			return true
		}
	}
	return false
}

func writeLockInfo(f *os.File) {
	info := lockInfo{PID: os.Getpid(), Timestamp: time.Now().Unix()}
	data, _ := json.Marshal(info)
	f.Write(data)
}

func isStale(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var info lockInfo
	if err := json.Unmarshal(data, &info); err != nil {
		// Can't parse — check file age
		st, err := os.Stat(path)
		if err != nil {
			return false
		}
		return time.Since(st.ModTime()) > staleDuration
	}
	// Check if PID is still alive
	if info.PID > 0 && !processAlive(info.PID) {
		return true
	}
	// Check timestamp staleness
	if time.Since(time.Unix(info.Timestamp, 0)) > staleDuration {
		return true
	}
	return false
}

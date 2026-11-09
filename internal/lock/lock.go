// Package lock provides a simple flock-based single-instance lock.
package lock

import (
	"fmt"
	"os"
	"syscall"
)

// Lock is an acquired advisory file lock.
type Lock struct {
	fd int
}

// Acquire takes an exclusive non-blocking lock on path, creating the file if
// needed. It writes the current PID into the file for diagnostics.
func Acquire(path string) (*Lock, error) {
	fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open lock %s: %w", path, err)
	}
	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("another alt-shift-ime instance holds %s: %w", path, err)
	}
	_ = syscall.Ftruncate(fd, 0)
	_, _ = syscall.Write(fd, []byte(fmt.Sprintf("%d\n", os.Getpid())))
	return &Lock{fd: fd}, nil
}

// Close releases the lock.
func (l *Lock) Close() error {
	if l == nil || l.fd < 0 {
		return nil
	}
	err := syscall.Flock(l.fd, syscall.LOCK_UN)
	cerr := syscall.Close(l.fd)
	l.fd = -1
	if err != nil {
		return err
	}
	return cerr
}

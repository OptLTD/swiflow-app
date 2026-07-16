package httputil

import (
	"io"
	"sync"
	"time"
)

// idleReadCloser closes the underlying body if no bytes arrive within idle. A
// plain http.Client timeout does not fire once a server has sent headers and
// then stalls the body, so streaming/large responses can hang until the caller
// context deadline. This guard bounds that stall.
type idleReadCloser struct {
	rc    io.ReadCloser
	timer *time.Timer
	idle  time.Duration
	once  sync.Once
}

// NewIdleReadCloser wraps rc so that a gap of more than idle between successful
// reads closes rc (making the in-flight Read return an error). idle <= 0 returns
// rc unchanged.
func NewIdleReadCloser(rc io.ReadCloser, idle time.Duration) io.ReadCloser {
	if idle <= 0 || rc == nil {
		return rc
	}
	ir := &idleReadCloser{rc: rc, idle: idle}
	ir.timer = time.AfterFunc(idle, func() { _ = ir.rc.Close() })
	return ir
}

func (ir *idleReadCloser) Read(p []byte) (int, error) {
	n, err := ir.rc.Read(p)
	if n > 0 {
		ir.timer.Reset(ir.idle)
	}
	return n, err
}

func (ir *idleReadCloser) Close() error {
	ir.once.Do(func() { ir.timer.Stop() })
	return ir.rc.Close()
}

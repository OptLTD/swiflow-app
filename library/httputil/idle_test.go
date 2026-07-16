package httputil

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// blockingRC blocks in Read until Close is called.
type blockingRC struct {
	ch   chan struct{}
	once sync.Once
}

func (b *blockingRC) Read(p []byte) (int, error) {
	<-b.ch
	return 0, nil
}
func (b *blockingRC) Close() error {
	b.once.Do(func() { close(b.ch) })
	return nil
}

func TestIdleReadCloserAbortsStall(t *testing.T) {
	b := &blockingRC{ch: make(chan struct{})}
	rc := NewIdleReadCloser(b, 30*time.Millisecond)
	done := make(chan struct{})
	go func() {
		buf := make([]byte, 8)
		_, _ = rc.Read(buf)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("idle guard did not close a stalled body")
	}
}

func TestIdleReadCloserPassesData(t *testing.T) {
	src := &nopCloser{Reader: strings.NewReader("hello world")}
	rc := NewIdleReadCloser(src, 5*time.Second)
	buf := make([]byte, 5)
	n, err := rc.Read(buf)
	if err != nil || n != 5 || string(buf) != "hello" {
		t.Fatalf("n=%d err=%v buf=%q", n, err, string(buf))
	}
	_ = rc.Close()
}

type nopCloser struct{ Reader interface{ Read([]byte) (int, error) } }

func (n *nopCloser) Read(p []byte) (int, error) { return n.Reader.Read(p) }
func (n *nopCloser) Close() error               { return nil }

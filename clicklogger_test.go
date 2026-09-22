package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type mockExecer struct {
	mu    sync.Mutex
	calls int
}

func (m *mockExecer) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	return pgconn.CommandTag{}, nil
}

func (m *mockExecer) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func TestClickLogger_Log_DropsWhenBufferFull(t *testing.T) {
	cl := NewClickLogger(nil, 1)

	cl.Log(ClickEvent{ShortCode: "a"})

	done := make(chan struct{})
	go func() {
		cl.Log(ClickEvent{ShortCode: "b"}) // buffer full must return
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Log blocked instead of dropping when the buffer was full")
	}

	if got := len(cl.events); got != 1 {
		t.Fatalf("expected 1 buffered event (the drop should not have been queued), got %d", got)
	}
}

func TestClickLogger_Log_AcceptsWhenBufferHasRoom(t *testing.T) {
	cl := NewClickLogger(nil, 2)

	cl.Log(ClickEvent{ShortCode: "a"})
	cl.Log(ClickEvent{ShortCode: "b"})

	if got := len(cl.events); got != 2 {
		t.Fatalf("expected 2 buffered events, got %d", got)
	}
}

func TestClickLogger_DrainOnShutdown(t *testing.T) {
	mock := &mockExecer{}
	cl := NewClickLogger(mock, 100)

	const total = 20
	for i := 0; i < total; i++ {
		cl.Log(ClickEvent{
			ShortCode: fmt.Sprintf("code-%d", i),
			Timestamp: time.Now(),
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cl.Run(ctx)

	if got := mock.CallCount(); got != total {
		t.Fatalf("expected %d events drained, got %d", total, got)
	}
}

func TestClickLogger_NothingLostDuringGracefulShutdown(t *testing.T) {
	mock := &mockExecer{}
	cl := NewClickLogger(mock, 200)

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		cl.Run(ctx)
	}()

	const total = 50
	for i := 0; i < total; i++ {
		cl.Log(ClickEvent{
			ShortCode: fmt.Sprintf("ev-%d", i),
			Timestamp: time.Now(),
		})
	}

	time.Sleep(50 * time.Millisecond)

	cancel()
	wg.Wait()

	if got := mock.CallCount(); got != total {
		t.Fatalf("expected all %d events written, got %d", total, got)
	}
}

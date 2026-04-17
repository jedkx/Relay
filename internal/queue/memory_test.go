package queue

import (
	"testing"
	"time"
)

func TestMemoryQueue_EnqueueDequeue_RoundTrip(t *testing.T) {
	t.Parallel()

	q := NewMemoryQueue(2)

	const want = "event-1"
	q.Enqueue(want)

	got := q.Dequeue()
	if got != want {
		t.Fatalf("Dequeue() = %v, want %v", got, want)
	}
}

func TestMemoryQueue_EnqueueBlocksWhenFull(t *testing.T) {
	t.Parallel()

	q := NewMemoryQueue(1)

	q.Enqueue("a")

	done := make(chan struct{})
	go func() {
		q.Enqueue("b")
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("expected Enqueue to block while buffer is full")
	case <-time.After(50 * time.Millisecond):
	}

	if got := q.Dequeue(); got != "a" {
		t.Fatalf("Dequeue() = %v, want %q", got, "a")
	}

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("expected blocked Enqueue to complete after dequeue")
	}
}

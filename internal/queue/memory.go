package queue

// MemoryQueue is a buffered FIFO queue backed by a channel.
type MemoryQueue struct {
	ch chan any
}

// NewMemoryQueue creates a queue with a buffer of size elements.
func NewMemoryQueue(size int) *MemoryQueue {
	if size <= 0 {
		panic("size must be greater than 0")
	}
	return &MemoryQueue{ch: make(chan any, size)}
}

// Enqueue sends v to the queue. It blocks when the buffer is full.
func (q *MemoryQueue) Enqueue(v any) {
	q.ch <- v
}

// Dequeue receives the next value. It blocks when the queue is empty.
func (q *MemoryQueue) Dequeue() any {
	return <-q.ch
}

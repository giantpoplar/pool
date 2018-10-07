package pool

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

var errDequeFull = errors.New("deque elements reach max capacity limit")

type blockingDeque struct {
	*deque
	gate chan struct{}
}

func newBlockingDeque(maxCap, nilCap int) *blockingDeque {
	q := &blockingDeque{
		newDeque(maxCap),
		make(chan struct{}, maxCap),
	}
	for i := 0; i < nilCap; i++ {
		q.gate <- struct{}{}
	}
	return q
}

// Pop pops an item from the tail of double-ended queue in timeout duraton
func (q *blockingDeque) pop(timeout time.Duration) (interface{}, error) {
	select {
	case <-time.After(timeout):
		return nil, ErrTimeout
	case <-q.gate:
		elem := q.deque.pop()
		return elem, nil
	}

}

// Append appends item to the tail of double-ended queue
func (q *blockingDeque) append(item interface{}) error {
	select {
	case q.gate <- struct{}{}:
		q.deque.append(item)
		return nil
	default:
		return errDequeFull
	}
}

// Prepend appends item to the head of double-end queue
func (q *blockingDeque) prepend(item interface{}) error {
	select {
	case q.gate <- struct{}{}:
		q.deque.prepend(item)
		return nil
	default:
		return errDequeFull
	}
}

type WalkFunc func(item interface{})

// Walk
func (q *blockingDeque) Walk(walkFn WalkFunc) {
	q.walk(walkFn)
}

// Deque is a head-tail linked list data structure implementation.
// It is based on a doubly linked list container, so that every
// operations time complexity is O(1).
//
// every operations over an instiated Deque are synchronized and
// safe for concurrent usage.
type deque struct {
	sync.RWMutex
	container *list.List
	capacity  int
}

// NewCappedDeque creates a Deque with the specified capacity limit.
func newDeque(capacity int) *deque {
	return &deque{
		container: list.New(),
		capacity:  capacity,
	}
}

// Append inserts element at the back of the Deque in a O(1) time complexity,
// returning true if successful or false if the deque is at capacity.
func (s *deque) append(item interface{}) bool {
	s.Lock()
	defer s.Unlock()

	if s.capacity < 0 || s.container.Len() < s.capacity {
		s.container.PushBack(item)
		return true
	}

	return false
}

// prepend inserts element at the Deques front in a O(1) time complexity,
// returning true if successful or false if the deque is at capacity.
func (s *deque) prepend(item interface{}) bool {
	s.Lock()
	defer s.Unlock()

	if s.capacity < 0 || s.container.Len() < s.capacity {
		s.container.PushFront(item)
		return true
	}

	return false
}

// pop removes the last element of the deque in a O(1) time complexity
func (s *deque) pop() interface{} {
	s.Lock()
	defer s.Unlock()

	var item interface{} = nil
	var lastContainerItem *list.Element = nil
	lastContainerItem = s.container.Back()
	if lastContainerItem != nil {
		item = s.container.Remove(lastContainerItem)
	}

	return item
}

// walk traverses the deque from front, calling walkFn for each element,
// and return polymerized result
func (s *deque) walk(walkFn WalkFunc) {
	s.Lock()
	defer s.Unlock()

	item := s.container.Front()

	for {
		if item != nil {
			walkFn(item.Value)
			item = item.Next()
		} else {
			break
		}
	}
}

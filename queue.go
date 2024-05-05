package main

import (
	"context"
	"errors"
	"sync"
)

var (
	_ FifoQueue[int] = (*list[int])(nil)

	ErrNotFound = errors.New("not found")
)

type FifoQueue[T any] interface {
	PushBack(val T)
	PopFront() (T, bool)
	Size() int
}

type node[T any] struct {
	val  T
	next *node[T]
}

type list[T any] struct {
	head *node[T]
	tail *node[T]

	len int
}

func NewList[T any]() *list[T] {
	return &list[T]{}
}

func (q *list[T]) PushBack(val T) {
	n := &node[T]{
		val:  val,
		next: nil,
	}

	if q.tail == nil {
		q.head = n
		q.tail = n
	} else {
		q.tail.next = n
		q.tail = n
	}

	q.len++
}

func (q *list[T]) PopFront() (T, bool) {
	if q.head == nil {
		var val T
		return val, false
	}

	val := q.head.val
	q.head = q.head.next
	q.len--

	if q.head == nil {
		q.tail = nil
	}

	return val, true
}

func (q *list[T]) Size() int {
	return q.len
}

type fchan[T any] struct {
	mu sync.Mutex

	elems   FifoQueue[T]
	waiters FifoQueue[waiter[T]]
}

func newfchan[T any]() *fchan[T] {
	return &fchan[T]{
		elems:   NewList[T](),
		waiters: NewList[waiter[T]](),
	}
}

type waiter[T any] struct {
	ctx context.Context
	res chan T
}

func (w *waiter[T]) isValid() bool {
	return w.ctx.Err() == nil
}

func (fq *fchan[T]) PushBack(val T) {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	for fq.waiters.Size() > 0 {
		w, _ := fq.waiters.PopFront()
		if !w.isValid() {
			close(w.res)
			continue
		}
		w.res <- val
		return
	}

	fq.elems.PushBack(val)
}

func (fq *fchan[T]) PopFront(ctx context.Context, shouldWait bool) (chan T, error) {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	res := make(chan T)

	if fq.elems.Size() == 0 {
		if !shouldWait {
			return nil, ErrNotFound
		}
		fq.waiters.PushBack(waiter[T]{
			ctx: ctx,
			res: res,
		})
		return res, nil
	}

	val, _ := fq.elems.PopFront()
	go func() {
		res <- val
		close(res)
	}()

	return res, nil
}

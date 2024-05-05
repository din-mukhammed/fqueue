package main

import (
	"context"
	"testing"
	"time"
)

func TestFifoQueue(t *testing.T) {
	const N = 1_000_000
	var (
		q = list[int]{}

		seen = map[int]struct{}{}
	)

	for i := 0; i < N; i++ {
		q.PushBack(i)
	}

	for q.Size() > 0 {
		val, _ := q.PopFront()
		seen[val] = struct{}{}
	}

	if len(seen) != N {
		t.Fatalf("not all elements are seen: %v != %v", seen, N)
	}
}

func TestFchanWaiting(t *testing.T) {
	var (
		q = newfchan[int]()
	)

	ctx := context.Background()
	q.PopFront(ctx, false)
	q.PopFront(ctx, false)
	q.PopFront(ctx, false)

	time.Sleep(time.Second)

	const expectedVal = 10
	go q.PushBack(expectedVal)

	testTimer := time.NewTimer(5 * time.Second)
	defer testTimer.Stop()

	ch, err := q.PopFront(ctx, true)
	if err != nil {
		t.Fatalf("pop: %v", err)
	}

	select {
	case val := <-ch:
		if val != expectedVal {
			t.Fatalf("expected: %v, but received: %v", expectedVal, val)
		}
		return
	case <-testTimer.C:
		t.Fatalf("test timeout")
	}
}

func TestFchanOrder(t *testing.T) {
	var (
		q = newfchan[int]()
	)

	ctx1, _ := context.WithTimeout(context.Background(), time.Hour)
	res1, _ := q.PopFront(ctx1, true)

	ctx2, _ := context.WithTimeout(context.Background(), time.Hour)
	res2, _ := q.PopFront(ctx2, true)

	ctx3, _ := context.WithTimeout(context.Background(), time.Hour)
	res3, _ := q.PopFront(ctx3, true)

	go func() {
		q.PushBack(1)
		q.PushBack(2)
		q.PushBack(3)
	}()

	selectTimer := time.NewTimer(1 * time.Second)
	defer selectTimer.Stop()

	vv := []int{}
Loop:
	for {
		select {
		case val := <-res1:
			vv = append(vv, val)
		case val := <-res2:
			vv = append(vv, val)
		case val := <-res3:
			vv = append(vv, val)
		case <-selectTimer.C:
			break Loop
		}
	}

	if vv[0] != 1 {
		t.Fatalf("vv[0] should be: 1")
	}
	if vv[1] != 2 {
		t.Fatalf("vv[1] should be: 2")
	}
	if vv[2] != 3 {
		t.Fatalf("vv[2] should be: 3")
	}
}

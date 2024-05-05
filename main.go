package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

func main() {
	port := os.Args[1]
	if port == "" {
		panic("port is not specified")
	}

	srv := newService()

	mux := http.NewServeMux()
	mux.HandleFunc("PUT /{queue}", srv.addHandler)
	mux.HandleFunc("GET /{queue}", srv.getHandler)

	slog.Info("web server starting...")
	if err := http.ListenAndServe("127.0.0.1:"+port, mux); err != nil {
		panic(fmt.Sprintf("http serve listener: %v", err))
	}
}

type service struct {
	mu     sync.Mutex
	queues map[string]*fchan[string]
}

func newService() *service {
	return &service{
		queues: map[string]*fchan[string]{},
	}
}

func (s *service) queueByName(name string) *fchan[string] {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.queues[name]; !ok {
		s.queues[name] = newfchan[string]()
	}
	return s.queues[name]
}

func (s *service) addHandler(w http.ResponseWriter, r *http.Request) {
	v := r.FormValue("v")
	if v == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	queueName := r.PathValue("queue")
	s.queueByName(queueName).PushBack(v)
}

func (s *service) getHandler(w http.ResponseWriter, r *http.Request) {
	var (
		ctx    = r.Context()
		cancel context.CancelFunc
	)
	ts := r.FormValue("timeout")
	if ts != "" {
		tsSec, err := strconv.Atoi(ts)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid timeout param"))
			return
		}

		ctx, cancel = context.WithTimeout(r.Context(), time.Duration(tsSec)*time.Second)
		defer cancel()
	}

	queueName := r.PathValue("queue")
	shouldWait := ts != ""
	ch, err := s.queueByName(queueName).PopFront(ctx, shouldWait)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	select {
	case v, ok := <-ch:
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(v))
	case <-ctx.Done():
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

/*
 * Copyright 2026 Holger de Carne
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package consorsbank

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"sync"
	"time"

	"github.com/tdrn-org/go-finance"
	"google.golang.org/grpc"
)

const initialReplyTimeout time.Duration = 250 * time.Millisecond

type subscriptionReply[T any] struct {
	reply    *T
	replyErr error
}

func (r *subscriptionReply[T]) Empty() bool {
	return r.reply == nil && r.replyErr == nil
}

type streamSubscription[R, T any] struct {
	Timeout        time.Duration
	client         grpc.ServerStreamingClient[R]
	cancel         context.CancelFunc
	processReply   func(*streamSubscription[R, T], *R)
	initialReply   chan subscriptionReply[T]
	lastReply      subscriptionReply[T]
	subscribeUntil time.Time
	closed         bool
	logger         *slog.Logger
	mutex          sync.Mutex
}

func newSubscription[R, T any](client grpc.ServerStreamingClient[R], cancel context.CancelFunc, processReply func(*streamSubscription[R, T], *R), logger *slog.Logger) *streamSubscription[R, T] {
	return &streamSubscription[R, T]{
		client:       client,
		cancel:       cancel,
		processReply: processReply,
		initialReply: make(chan subscriptionReply[T], 1),
		logger:       logger,
	}
}

func (s *streamSubscription[R, T]) IsClosed() bool {
	if s == nil {
		return true
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.closed
}

func (s *streamSubscription[R, T]) Close() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.closeLocked()
}

func (s *streamSubscription[R, T]) closeLocked() {
	s.closed = true
	s.cancel()
}

func (s *streamSubscription[R, T]) Start() {
	s.logger.Info("subscription starting...")
	s.resetTimeout()
	for {
		reply, err := s.client.Recv()
		if err != nil {
			s.Close()
			if errors.Is(err, io.EOF) {
				s.logger.Info("closing subscription; server closed connection")
			} else if s.client.Context().Err() != nil {
				s.logger.Info("closing subscription; client is disconnecting")
			} else {
				s.logger.Info("closing subscription; recv failure", slog.Any("err", err))
			}
			return
		}
		s.processReply(s, reply)
	}
}

func (s *streamSubscription[R, T]) resetTimeout() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.subscribeUntil = time.Now().Add(s.Timeout)
}

func (s *streamSubscription[R, T]) WaitInitialReply(ctx context.Context, timeout time.Duration) (*T, error) {
	reply, err := s.LastReply()
	if err == nil || !errors.Is(err, finance.ErrRequestPending) {
		return reply, err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	select {
	case reply := <-s.initialReply:
		return reply.reply, reply.replyErr
	case <-timeoutCtx.Done():
		return s.LastReply()
	}
}

func (s *streamSubscription[R, T]) MergeReply(rawReply *R, merge func(lastReply *T, rawReply *R) (*T, error)) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	reply, replyErr := merge(s.lastReply.reply, rawReply)
	if reply == nil && replyErr == nil {
		return
	}
	initialReply := s.lastReply.Empty()
	s.lastReply.reply, s.lastReply.replyErr = reply, replyErr
	if reply != nil {
		s.subscribeUntil = time.Now().Add(s.Timeout)
	}

	replyType := reflect.TypeFor[T]()
	if initialReply {
		s.logger.Info("recorded initial reply", slog.Any("reply", replyType))
		s.initialReply <- s.lastReply
	} else {
		s.logger.Debug("recorded reply", slog.Any("reply", replyType))
	}
}

func (s *streamSubscription[R, T]) LastReply() (*T, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	timedOut, closed := s.timeoutReachedOrClosedLocked()
	if timedOut && !closed {
		s.logger.Info("closing subscription; timeout reached")
		s.closeLocked()
	}
	if s.lastReply.Empty() {
		return nil, finance.ErrRequestPending
	}
	return s.lastReply.reply, s.lastReply.replyErr
}

func (s *streamSubscription[R, T]) timeoutReachedOrClosedLocked() (bool, bool) {
	if s.Timeout <= 0 || s.subscribeUntil.IsZero() {
		return false, s.closed
	}
	return time.Now().After(s.subscribeUntil), s.closed
}

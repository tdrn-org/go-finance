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

type streamSubscription[R, T any] struct {
	client              grpc.ServerStreamingClient[R]
	subscriptionTimeout time.Duration
	suscribeUntil       time.Time
	lastReply           *T
	processReply        func(*streamSubscription[R, T], *R)
	ctx                 context.Context
	cancel              context.CancelFunc
	closed              bool
	logger              *slog.Logger
	mutex               sync.Mutex
}

func (s *streamSubscription[R, T]) IsClosed() bool {
	if s == nil {
		return true
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.closed
}

func (s *streamSubscription[R, T]) MarkClosed() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.closed = true
}

func (s *streamSubscription[R, T]) Run() {
	s.logger.Info("subscription running...")
	for {
		reply, err := s.client.Recv()
		if err != nil {
			s.MarkClosed()
			if errors.Is(err, io.EOF) {
				s.logger.Info("closing subscription; server closed connection")
			} else if s.ctx.Err() != nil || s.client.Context().Err() != nil {
				s.logger.Info("closing subscription; client is disconnecting")
			} else {
				s.logger.Info("closing subscription; recv failure", slog.Any("err", err))
			}
			return
		}
		s.processReply(s, reply)
	}
}

func (s *streamSubscription[R, T]) RecordReply(reply *T) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	replyType := reflect.TypeOf(reply).Elem()
	if s.lastReply == nil {
		s.logger.Info("recording initial reply", slog.Any("reply", replyType))
	} else {
		s.logger.Debug("recording reply", slog.Any("reply", replyType))
	}
	s.lastReply = reply
	s.suscribeUntil = time.Now().Add(s.subscriptionTimeout)
}

func (s *streamSubscription[R, T]) LastReply() (*T, error) {
	if s.timeoutReached() {
		s.logger.Info("closing subscription; timeout reached")
		s.MarkClosed()
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.lastReply == nil {
		return nil, finance.ErrRequestPending
	}
	return s.lastReply, nil
}

func (s *streamSubscription[R, T]) timeoutReached() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.subscriptionTimeout <= 0 {
		return false
	}
	now := time.Now()
	if s.suscribeUntil.IsZero() {
		s.suscribeUntil = now.Add(s.subscriptionTimeout)
		return false
	}
	return now.After(s.suscribeUntil)
}

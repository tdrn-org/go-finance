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

package trace

import (
	"context"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Tracer struct {
	tracer trace.Tracer
}

func NewTracer[T any](opts ...trace.TracerOption) *Tracer {
	return &Tracer{
		tracer: otel.Tracer(reflect.TypeFor[T]().PkgPath(), opts...),
	}
}

func (t *Tracer) StartConsuming(ctx context.Context, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, callerName(), trace.WithSpanKind(trace.SpanKindConsumer), trace.WithAttributes(attributes...))
}

func SetHttpResponse(span trace.Span, rsp *http.Response) {
	span.SetAttributes(attribute.Int("http.status_code", rsp.StatusCode))
}

func SetStatus(span trace.Span, err error) error {
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	return err
}

func callerName() string {
	pc, _, _, _ := runtime.Caller(2)
	name := runtime.FuncForPC(pc).Name()
	split := strings.Split(name, "/")
	return split[len(split)-1]
}

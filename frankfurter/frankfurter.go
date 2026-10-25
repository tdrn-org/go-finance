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

// Package frankfurter provides access to the [Frankfurter REST API] service.
//
// [Frankfurter REST API]: https://frankfurter.dev/
package frankfurter

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	tracehelper "github.com/tdrn-org/go-finance/internal/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Name of Frankfurter Provider.
const Name string = "frankfurter"

var defaultLogger *slog.Logger = slog.With(slog.String("provider", Name))

// API provides access to the [Frankfurter REST API] and implements
//   - [finance.FX]
//
// [Frankfurter REST API]: https://frankfurter.dev/
type API struct {
	baseURL    *url.URL
	httpClient *http.Client
	logger     *slog.Logger
	tracer     *tracehelper.Tracer
}

// NewAPI creates a new Frankfurter provider instance using the given [Config].
func NewAPI(config Config) (*API, error) {
	baseURL, err := config.GetBaseURL()
	if err != nil {
		return nil, err
	}
	httpClient, err := config.GetHttpClient()
	if err != nil {
		return nil, err
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	api := &API{
		baseURL:    baseURL,
		httpClient: httpClient,
		logger:     defaultLogger.With(slog.Any("url", baseURL)),
		tracer:     tracehelper.NewTracer[API](trace.WithInstrumentationAttributes(attribute.String("provider", Name))),
	}
	api.logger.Info("provider enabled")
	return api, nil
}

// See [finance.APIProvider.ProviderName]
func (api *API) ProviderName() string {
	return Name
}

// See [finance.APIProvider.Shutdown]
func (api *API) Shutdown(_ context.Context) error {
	return nil
}

// See [finance.APIProvider.Close]
func (api *API) Close() error {
	return nil
}

func (api *API) url(service string, args ...string) *url.URL {
	apiURL := api.baseURL.JoinPath(service)
	query := apiURL.Query()
	for i := 0; i < len(args); i += 2 {
		query.Set(args[i], args[i+1])
	}
	apiURL.RawQuery = query.Encode()
	return apiURL
}

func (api *API) decodeResponse(ctx context.Context, rsp *http.Response, decoded any) error {
	tracehelper.SetHttpResponse(trace.SpanFromContext(ctx), rsp)
	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: service failure (status: %s)", Name, rsp.Status)
	}
	err := json.NewDecoder(rsp.Body).Decode(decoded)
	if err != nil {
		return fmt.Errorf("%s: failed to decode response body (cause: %w)", Name, err)
	}
	return nil
}

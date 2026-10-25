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

// Package openfigi provides access to the [OpenFIGI API].
//
// [OpenFIGI API]: https://www.openfigi.com/api
package openfigi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"reflect"

	"github.com/tdrn-org/go-finance"
	tracehelper "github.com/tdrn-org/go-finance/internal/trace"
	openfigiapi "github.com/tdrn-org/go-finance/openfigi/api"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Name of OpenFIGI Provider.
const Name string = "openfigi"

var defaultLogger *slog.Logger = slog.With(slog.String("provider", Name))

// API provides access to the [OpenFIGI API].
//
// [OpenFIGI API]: https://www.openfigi.com/api
type API struct {
	baseURL   *url.URL
	apiKey    string
	apiClient openfigiapi.ClientWithResponsesInterface
	logger    *slog.Logger
	tracer    *tracehelper.Tracer
}

// NewAPI creates a new OpenFIGI provider instance using the given [Config].
func NewAPI(config Config) (*API, error) {
	baseURL, err := config.GetBaseURL()
	if err != nil {
		return nil, err
	}
	apiKey, err := config.GetAPIKey()
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
	httpClientOption := func(apiClient *openfigiapi.Client) error {
		apiClient.Client = httpClient
		return nil
	}
	apiClient, err := openfigiapi.NewClientWithResponses(baseURL.String(), httpClientOption)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to create API client (cause: %w)", Name, err)
	}
	api := &API{
		baseURL:   baseURL,
		apiKey:    apiKey,
		apiClient: apiClient,
		logger:    defaultLogger.With(slog.Any("url", baseURL)),
		tracer:    tracehelper.NewTracer[API](trace.WithInstrumentationAttributes(attribute.String("provider", Name))),
	}
	api.logger.Info("provider enabled")
	return api, nil
}

// See [finance.APIProvider]
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

func (api *API) authenticateRequest(ctx context.Context, request *http.Request) error {
	if api.apiKey != "" {
		request.Header.Set("X-OPENFIGI-APIKEY", api.apiKey)
	}
	return nil
}

func (api *API) wrapSystemError(operation string, err error) error {
	return fmt.Errorf("%s: %s call failure (cause: %w)", Name, operation, err)
}

func (api *API) checkAPIResponse(ctx context.Context, operation string, rsp *http.Response) error {
	tracehelper.SetHttpResponse(trace.SpanFromContext(ctx), rsp)
	switch rsp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusTooManyRequests:
		return finance.ErrRateLimitReached
	default:
		return fmt.Errorf("%s: %s API failure (status: %s)", Name, operation, rsp.Status)
	}
}

func (api *API) checkAPIResponseWithBody(ctx context.Context, operation string, rsp *http.Response, body any) error {
	err := api.checkAPIResponse(ctx, operation, rsp)
	if err != nil {
		return err
	}
	v := reflect.ValueOf(body)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("%s: %s yields empty or unexpected response", Name, operation)
	}
	return nil
}

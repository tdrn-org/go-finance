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

package config

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/tdrn-org/go-cache"
	"github.com/tdrn-org/go-cache/memory"
	"github.com/tdrn-org/go-cache/redis"
	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/composite"
)

type CacheConfig struct {
	Type   CacheType         `toml:"type"`
	Memory MemoryCacheConfig `toml:"memory"`
	Redis  RedisCacheConfig  `toml:"redis"`
}

const unrecognizedCacheTypeErrMessage string = "unrecognized cache type '%s'"

func (c *CacheConfig) NewExchangeRateCache(ttl time.Duration) (composite.ExchangeRateCache, error) {
	switch c.Type {
	case CacheTypeMemory:
		return memory.NewKeyValue(0, -ttl, cache.NotFound[string, *finance.ExchangeRate]())
	case CacheTypeRedis:
		options := &redis.Options{
			Addr:     c.Redis.Address,
			Password: c.Redis.Password,
			DB:       c.Redis.DB,
		}
		return redis.NewKeyValue(options, -ttl, redis.StringKey, cache.JSONSerializer[*finance.ExchangeRate]())
	default:
		return nil, fmt.Errorf(unrecognizedCacheTypeErrMessage, c.Type)
	}
}

func (c *CacheConfig) NewQuoteCache(ttl time.Duration) (composite.QuoteCache, error) {
	switch c.Type {
	case CacheTypeMemory:
		return memory.NewKeyValue(0, -ttl, cache.NotFound[string, *finance.Quote]())
	case CacheTypeRedis:
		options := &redis.Options{
			Addr:     c.Redis.Address,
			Password: c.Redis.Password,
			DB:       c.Redis.DB,
		}
		return redis.NewKeyValue(options, -ttl, redis.StringKey, cache.JSONSerializer[*finance.Quote]())
	default:
		return nil, fmt.Errorf(unrecognizedCacheTypeErrMessage, c.Type)
	}
}

type MemoryCacheConfig struct{}

type RedisCacheConfig struct {
	Address  string `toml:"address"`
	Password string `toml:"password"`
	DB       int    `toml:"db"`
}

type CacheType string

const (
	CacheTypeMemory CacheType = "memory"
	CacheTypeRedis  CacheType = "redis"
)

var knownCacheTypes map[string]CacheType = map[string]CacheType{
	string(CacheTypeMemory): CacheTypeMemory,
	string(CacheTypeRedis):  CacheTypeRedis,
}

func (t *CacheType) Value() string {
	for value, cacheType := range knownCacheTypes {
		if *t == cacheType {
			return value
		}
	}
	slog.Warn("unexpected cache type", slog.Any("t", *t))
	return ""
}

func (t *CacheType) MarshalTOML() ([]byte, error) {
	return []byte(`"` + t.Value() + `"`), nil
}

func (t *CacheType) UnmarshalTOML(value any) error {
	databaseTypeString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected cache type type %v", value)
	}
	cacheType, ok := knownCacheTypes[databaseTypeString]
	if !ok {
		return fmt.Errorf("unknown cache type: '%s'", databaseTypeString)
	}
	*t = cacheType
	return nil
}

package handler

import (
	"sync"
	"time"

	"github.com/compliance-framework/api/internal/config"
	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

// lineageEngineCache shares built lineage engines across requests and users, keyed by
// scope (SSP x component). Building an engine loads every catalog, filter, latest
// evidence stream, risk and SSP, while serving one node from it takes about a
// millisecond, so without the cache every expand or drawer open in the Compliance Map
// paid for a full rebuild. Engines are reused until the TTL passes or a caller asks
// for a refresh; concurrent misses for one scope share a single build.
//
// Sharing is safe because the engine applies no per-user filtering: the lineage
// routes are guarded at the resource level only.
type lineageEngineCache struct {
	ttl        time.Duration
	maxEntries int
	build      func(sspID, componentID *uuid.UUID) (*lineageEngine, error)
	now        func() time.Time

	mu      sync.Mutex
	entries map[lineageScopeKey]lineageCacheEntry
	flight  singleflight.Group
}

func lineageConfigOrDefault(cfg *config.LineageConfig) *config.LineageConfig {
	if cfg == nil {
		return config.DefaultLineageConfig()
	}
	return cfg
}

type lineageScopeKey struct {
	sspID       uuid.UUID
	componentID uuid.UUID
}

func (k lineageScopeKey) String() string {
	return k.sspID.String() + "|" + k.componentID.String()
}

type lineageCacheEntry struct {
	engine  *lineageEngine
	builtAt time.Time
}

func newLineageEngineCache(ttl time.Duration, maxEntries int, build func(sspID, componentID *uuid.UUID) (*lineageEngine, error)) *lineageEngineCache {
	if maxEntries < 1 {
		maxEntries = 1
	}
	return &lineageEngineCache{
		ttl:        ttl,
		maxEntries: maxEntries,
		build:      build,
		now:        time.Now,
		entries:    map[lineageScopeKey]lineageCacheEntry{},
	}
}

// get returns the engine for the scope and when its data was loaded. refresh skips
// the cached engine and replaces it with a new build.
func (c *lineageEngineCache) get(sspID, componentID *uuid.UUID, refresh bool) (*lineageEngine, time.Time, error) {
	if c.ttl <= 0 {
		builtAt := c.now()
		engine, err := c.build(sspID, componentID)
		return engine, builtAt, err
	}

	key := lineageScopeKey{}
	if sspID != nil {
		key.sspID = *sspID
	}
	if componentID != nil {
		key.componentID = *componentID
	}

	if !refresh {
		c.mu.Lock()
		entry, ok := c.entries[key]
		c.mu.Unlock()
		if ok && c.now().Sub(entry.builtAt) < c.ttl {
			return entry.engine, entry.builtAt, nil
		}
	}

	// A refresh that joins a build already in flight shares it: that build started
	// after the refresh was asked for or moments before, so its data is current.
	v, err, _ := c.flight.Do(key.String(), func() (any, error) {
		builtAt := c.now()
		engine, err := c.build(sspID, componentID)
		if err != nil {
			return nil, err
		}
		entry := lineageCacheEntry{engine: engine, builtAt: builtAt}
		c.store(key, entry)
		return entry, nil
	})
	if err != nil {
		return nil, time.Time{}, err
	}
	entry := v.(lineageCacheEntry)
	return entry.engine, entry.builtAt, nil
}

// store saves the entry, first dropping expired entries and then, while still full,
// the oldest one.
func (c *lineageEngineCache) store(key lineageScopeKey, entry lineageCacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	for k, e := range c.entries {
		if now.Sub(e.builtAt) >= c.ttl {
			delete(c.entries, k)
		}
	}
	delete(c.entries, key)
	for len(c.entries) >= c.maxEntries {
		var oldest lineageScopeKey
		first := true
		for k, e := range c.entries {
			if first || e.builtAt.Before(c.entries[oldest].builtAt) {
				oldest, first = k, false
			}
		}
		delete(c.entries, oldest)
	}
	c.entries[key] = entry
}

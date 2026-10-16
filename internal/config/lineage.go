package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	// DefaultLineageCacheTTL is how long a built Compliance Map (lineage) dataset is
	// reused before a request rebuilds it.
	DefaultLineageCacheTTL = 5 * time.Minute
	// DefaultLineageCacheMaxEntries bounds how many scopes (SSP x component) are cached.
	DefaultLineageCacheMaxEntries = 32
)

// LineageConfig controls the shared cache behind the /api/lineage read routes.
type LineageConfig struct {
	// CacheTTL is how long a built dataset is reused (CCF_LINEAGE_CACHE_TTL, a Go
	// duration). 0 disables the cache, so every request rebuilds.
	CacheTTL time.Duration
	// CacheMaxEntries bounds the cached scopes (CCF_LINEAGE_CACHE_MAX_ENTRIES); the
	// oldest is evicted first.
	CacheMaxEntries int
}

func DefaultLineageConfig() *LineageConfig {
	return &LineageConfig{
		CacheTTL:        DefaultLineageCacheTTL,
		CacheMaxEntries: DefaultLineageCacheMaxEntries,
	}
}

func LoadLineageConfig() (*LineageConfig, error) {
	return LoadLineageConfigFromViper(viper.GetViper())
}

func LoadLineageConfigFromViper(v *viper.Viper) (*LineageConfig, error) {
	cfg := DefaultLineageConfig()
	if v.IsSet("lineage_cache_ttl") {
		raw := strings.TrimSpace(v.GetString("lineage_cache_ttl"))
		ttl, err := time.ParseDuration(raw)
		if err != nil || ttl < 0 {
			return nil, fmt.Errorf("CCF_LINEAGE_CACHE_TTL must be a duration such as 5m, or 0 to disable, got %q", raw)
		}
		cfg.CacheTTL = ttl
	}
	if v.IsSet("lineage_cache_max_entries") {
		value := v.GetInt("lineage_cache_max_entries")
		if value <= 0 {
			return nil, fmt.Errorf("CCF_LINEAGE_CACHE_MAX_ENTRIES must be a positive integer, got %q", v.GetString("lineage_cache_max_entries"))
		}
		cfg.CacheMaxEntries = value
	}
	return cfg, nil
}

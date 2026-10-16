package config

import (
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadLineageConfigDefaults(t *testing.T) {
	cfg, err := LoadLineageConfigFromViper(viper.New())
	require.NoError(t, err)
	assert.Equal(t, DefaultLineageConfig(), cfg)
}

func TestLoadLineageConfigOverrides(t *testing.T) {
	v := viper.New()
	v.Set("lineage_cache_ttl", "30s")
	v.Set("lineage_cache_max_entries", 4)

	cfg, err := LoadLineageConfigFromViper(v)
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, cfg.CacheTTL)
	assert.Equal(t, 4, cfg.CacheMaxEntries)
}

func TestLoadLineageConfigZeroTTLDisablesCache(t *testing.T) {
	v := viper.New()
	v.Set("lineage_cache_ttl", "0")

	cfg, err := LoadLineageConfigFromViper(v)
	require.NoError(t, err)
	assert.Zero(t, cfg.CacheTTL)
}

func TestLoadLineageConfigRejectsInvalidValues(t *testing.T) {
	for name, set := range map[string]func(v *viper.Viper){
		"negative ttl":     func(v *viper.Viper) { v.Set("lineage_cache_ttl", "-1m") },
		"unparsable ttl":   func(v *viper.Viper) { v.Set("lineage_cache_ttl", "soon") },
		"zero max entries": func(v *viper.Viper) { v.Set("lineage_cache_max_entries", 0) },
	} {
		t.Run(name, func(t *testing.T) {
			v := viper.New()
			set(v)
			_, err := LoadLineageConfigFromViper(v)
			assert.Error(t, err)
		})
	}
}

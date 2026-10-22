package config

import (
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestLoadSSPTemplateConfig(t *testing.T) {
	v := viper.New()
	cfg, err := LoadSSPTemplateConfigFromViper(v)
	require.NoError(t, err)
	require.Equal(t, &SSPTemplateConfig{DiscoverySchedule: DefaultSSPTemplateDiscoverySchedule, StaleAfter: DefaultSSPTemplateStaleAfter}, cfg)

	v.Set("ssp_template_discovery_schedule", " 0 0 * * * * ")
	v.Set("ssp_template_stale_after", "24h")
	cfg, err = LoadSSPTemplateConfigFromViper(v)
	require.NoError(t, err)
	require.Equal(t, &SSPTemplateConfig{DiscoverySchedule: "0 0 * * * *", StaleAfter: 24 * time.Hour}, cfg)

	for _, bad := range []string{"soon", "0", "-1h"} {
		v.Set("ssp_template_stale_after", bad)
		_, err = LoadSSPTemplateConfigFromViper(v)
		require.Error(t, err, bad)
	}
}

package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestLoadSSPScopeConfig(t *testing.T) {
	v := viper.New()
	require.Equal(t, DefaultSSPScopeAutoFiltersSchedule, LoadSSPScopeConfigFromViper(v).AutoFiltersSchedule)

	v.Set("ssp_scope_auto_filters_schedule", " 0 0 * * * * ")
	require.Equal(t, "0 0 * * * *", LoadSSPScopeConfigFromViper(v).AutoFiltersSchedule)
}

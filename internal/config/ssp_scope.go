package config

import (
	"strings"

	"github.com/spf13/viper"
)

// DefaultSSPScopeAutoFiltersSchedule runs the automatic per-policy filter reconcile
// hourly (River 6-field cron).
const DefaultSSPScopeAutoFiltersSchedule = "0 23 * * * *"

// SSPScopeConfig configures SSP scopes.
type SSPScopeConfig struct {
	// AutoFiltersSchedule is the River cron of the job that creates the missing automatic
	// per-policy filters of every SSP with automatic filters on
	// (CCF_SSP_SCOPE_AUTO_FILTERS_SCHEDULE).
	AutoFiltersSchedule string
}

func LoadSSPScopeConfig() *SSPScopeConfig {
	return LoadSSPScopeConfigFromViper(viper.GetViper())
}

func LoadSSPScopeConfigFromViper(v *viper.Viper) *SSPScopeConfig {
	cfg := &SSPScopeConfig{AutoFiltersSchedule: DefaultSSPScopeAutoFiltersSchedule}
	if schedule := strings.TrimSpace(v.GetString("ssp_scope_auto_filters_schedule")); schedule != "" {
		cfg.AutoFiltersSchedule = schedule
	}
	return cfg
}

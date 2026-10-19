package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	// DefaultSSPTemplateDiscoverySchedule runs SSP template discovery hourly (River
	// 6-field cron).
	DefaultSSPTemplateDiscoverySchedule = "0 37 * * * *"
	// DefaultSSPTemplateStaleAfter is how long an SSP a template created can go without
	// evidence for its key values before it is reported as stale.
	DefaultSSPTemplateStaleAfter = 7 * 24 * time.Hour
)

// SSPTemplateConfig configures SSP template discovery.
type SSPTemplateConfig struct {
	// DiscoverySchedule is the River cron of the discovery job
	// (CCF_SSP_TEMPLATE_DISCOVERY_SCHEDULE).
	DiscoverySchedule string
	// StaleAfter is how long a created SSP can go without evidence before it is
	// reported as stale (CCF_SSP_TEMPLATE_STALE_AFTER, a Go duration).
	StaleAfter time.Duration
}

func LoadSSPTemplateConfig() (*SSPTemplateConfig, error) {
	return LoadSSPTemplateConfigFromViper(viper.GetViper())
}

func LoadSSPTemplateConfigFromViper(v *viper.Viper) (*SSPTemplateConfig, error) {
	cfg := &SSPTemplateConfig{DiscoverySchedule: DefaultSSPTemplateDiscoverySchedule, StaleAfter: DefaultSSPTemplateStaleAfter}
	if schedule := strings.TrimSpace(v.GetString("ssp_template_discovery_schedule")); schedule != "" {
		cfg.DiscoverySchedule = schedule
	}
	if v.IsSet("ssp_template_stale_after") {
		raw := strings.TrimSpace(v.GetString("ssp_template_stale_after"))
		staleAfter, err := time.ParseDuration(raw)
		if err != nil || staleAfter <= 0 {
			return nil, fmt.Errorf("CCF_SSP_TEMPLATE_STALE_AFTER must be a positive duration such as 168h, got %q", raw)
		}
		cfg.StaleAfter = staleAfter
	}
	return cfg, nil
}

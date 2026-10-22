package worker

import (
	"testing"

	"github.com/compliance-framework/api/internal/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSSPScopeAutoFiltersSchedule(t *testing.T) {
	require.Equal(t, config.DefaultSSPScopeAutoFiltersSchedule, sspScopeAutoFiltersSchedule(nil))
	require.Equal(t, config.DefaultSSPScopeAutoFiltersSchedule, sspScopeAutoFiltersSchedule(&config.Config{}))
	require.Equal(t, "0 0 * * * *", sspScopeAutoFiltersSchedule(&config.Config{SSPScope: &config.SSPScopeConfig{AutoFiltersSchedule: "0 0 * * * *"}}))
}

func TestNewSSPScopeAutoFiltersPeriodicJobAcceptsBadSchedule(t *testing.T) {
	// An invalid schedule falls back to the default instead of failing start-up.
	require.NotNil(t, NewSSPScopeAutoFiltersPeriodicJob("not a cron", zap.NewNop().Sugar()))
	require.NotNil(t, NewSSPScopeAutoFiltersPeriodicJob(config.DefaultSSPScopeAutoFiltersSchedule, zap.NewNop().Sugar()))
}

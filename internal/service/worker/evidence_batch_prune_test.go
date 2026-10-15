package worker

import (
	"testing"
	"time"

	"github.com/compliance-framework/api/internal/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestEvidenceBatchPruneSettings(t *testing.T) {
	retention, schedule := evidenceBatchPruneSettings(nil)
	require.Equal(t, config.DefaultEvidenceBatchRetention, retention)
	require.Equal(t, config.DefaultEvidenceBatchPruneSchedule, schedule)

	retention, schedule = evidenceBatchPruneSettings(&config.Config{EvidenceBatch: &config.EvidenceBatchConfig{Retention: time.Hour, PruneSchedule: "0 0 * * * *"}})
	require.Equal(t, time.Hour, retention)
	require.Equal(t, "0 0 * * * *", schedule)
}

func TestNewEvidenceBatchPruneWorkerDefaultsRetention(t *testing.T) {
	require.Equal(t, config.DefaultEvidenceBatchRetention, NewEvidenceBatchPruneWorker(nil, 0, nil).retention)
	require.Equal(t, 2*time.Hour, NewEvidenceBatchPruneWorker(nil, 2*time.Hour, nil).retention)
}

func TestNewEvidenceBatchPrunePeriodicJobAcceptsBadSchedule(t *testing.T) {
	// An invalid schedule falls back to the default instead of failing start-up.
	require.NotNil(t, NewEvidenceBatchPrunePeriodicJob("not a cron", zap.NewNop().Sugar()))
	require.NotNil(t, NewEvidenceBatchPrunePeriodicJob(config.DefaultEvidenceBatchPruneSchedule, zap.NewNop().Sugar()))
}

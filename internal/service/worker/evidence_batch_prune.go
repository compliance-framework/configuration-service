package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/compliance-framework/api/internal/config"
	evidencesvc "github.com/compliance-framework/api/internal/service/relational/evidence"
	"github.com/riverqueue/river"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// JobTypeEvidenceBatchPrune deletes evidence batches older than their retention.
const JobTypeEvidenceBatchPrune = "evidence_batch_prune"

// EvidenceBatchPruneArgs are the (empty) args of the periodic batch prune job.
type EvidenceBatchPruneArgs struct{}

// Kind implements river.JobArgs.
func (EvidenceBatchPruneArgs) Kind() string { return JobTypeEvidenceBatchPrune }

// EvidenceBatchPruneWorker deletes evidence batches accepted more than retention ago
// (CCF_EVIDENCE_BATCH_RETENTION).
type EvidenceBatchPruneWorker struct {
	db        *gorm.DB
	retention time.Duration
	logger    *zap.SugaredLogger
	now       func() time.Time
}

// NewEvidenceBatchPruneWorker builds the worker; a non-positive retention uses the default.
func NewEvidenceBatchPruneWorker(db *gorm.DB, retention time.Duration, logger *zap.SugaredLogger) *EvidenceBatchPruneWorker {
	if retention <= 0 {
		retention = config.DefaultEvidenceBatchRetention
	}
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &EvidenceBatchPruneWorker{db: db, retention: retention, logger: logger, now: func() time.Time { return time.Now().UTC() }}
}

// Work implements the River worker function.
func (w *EvidenceBatchPruneWorker) Work(ctx context.Context, _ *river.Job[EvidenceBatchPruneArgs]) error {
	deleted, err := evidencesvc.PruneBatches(ctx, w.db, w.now().Add(-w.retention))
	if deleted > 0 {
		w.logger.Infow("Pruned evidence batches", "deleted", deleted, "retention", w.retention)
	}
	if err != nil {
		return fmt.Errorf("evidence batch prune: %w", err)
	}
	return nil
}

// NewEvidenceBatchPrunePeriodicJob schedules the prune job on the "scheduler" queue.
func NewEvidenceBatchPrunePeriodicJob(schedule string, logger *zap.SugaredLogger) *river.PeriodicJob {
	sched := parseCronScheduleWithFallback(schedule, config.DefaultEvidenceBatchPruneSchedule, "evidence batch prune", logger)
	return river.NewPeriodicJob(
		sched,
		func() (river.JobArgs, *river.InsertOpts) {
			return &EvidenceBatchPruneArgs{}, &river.InsertOpts{
				Queue:       "scheduler",
				MaxAttempts: 3,
				// Deduplicated per hour: a CCF_EVIDENCE_BATCH_PRUNE_SCHEDULE more frequent
				// than hourly is not honored.
				UniqueOpts: river.UniqueOpts{
					ByArgs:   true,
					ByPeriod: time.Hour,
				},
			}
		},
		&river.PeriodicJobOpts{RunOnStart: false},
	)
}

func evidenceBatchPruneSettings(cfg *config.Config) (time.Duration, string) {
	if cfg == nil || cfg.EvidenceBatch == nil {
		return config.DefaultEvidenceBatchRetention, config.DefaultEvidenceBatchPruneSchedule
	}
	return cfg.EvidenceBatch.Retention, cfg.EvidenceBatch.PruneSchedule
}

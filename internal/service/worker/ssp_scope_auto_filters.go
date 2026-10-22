package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/compliance-framework/api/internal/config"
	"github.com/compliance-framework/api/internal/service/relational/sspscope"
	"github.com/riverqueue/river"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// JobTypeSSPScopeAutoFilters creates the missing automatic per-policy filters of every
// SSP with automatic filters on.
const JobTypeSSPScopeAutoFilters = "ssp_scope_auto_filters"

// SSPScopeAutoFiltersArgs are the (empty) args of the periodic automatic-filter job.
type SSPScopeAutoFiltersArgs struct{}

// Kind implements river.JobArgs.
func (SSPScopeAutoFiltersArgs) Kind() string { return JobTypeSSPScopeAutoFilters }

// SSPScopeAutoFiltersWorker reconciles automatic per-policy filters, so policies that
// appear in in-scope evidence get a filter without a scope change.
type SSPScopeAutoFiltersWorker struct {
	scope  *sspscope.Service
	logger *zap.SugaredLogger
}

// NewSSPScopeAutoFiltersWorker builds the worker.
func NewSSPScopeAutoFiltersWorker(db *gorm.DB, logger *zap.SugaredLogger) *SSPScopeAutoFiltersWorker {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &SSPScopeAutoFiltersWorker{scope: sspscope.NewService(db), logger: logger}
}

// Work implements the River worker function.
func (w *SSPScopeAutoFiltersWorker) Work(ctx context.Context, _ *river.Job[SSPScopeAutoFiltersArgs]) error {
	created, err := w.scope.ReconcileAllAutoFilters(ctx)
	if created > 0 {
		w.logger.Infow("Created automatic per-policy filters", "created", created)
	}
	if err != nil {
		return fmt.Errorf("ssp scope automatic filters: %w", err)
	}
	return nil
}

// NewSSPScopeAutoFiltersPeriodicJob schedules the job on the "scheduler" queue.
func NewSSPScopeAutoFiltersPeriodicJob(schedule string, logger *zap.SugaredLogger) *river.PeriodicJob {
	sched := parseCronScheduleWithFallback(schedule, config.DefaultSSPScopeAutoFiltersSchedule, "ssp scope automatic filters", logger)
	return river.NewPeriodicJob(
		sched,
		func() (river.JobArgs, *river.InsertOpts) {
			return &SSPScopeAutoFiltersArgs{}, &river.InsertOpts{
				Queue:       "scheduler",
				MaxAttempts: 3,
				// Deduplicated per hour: a CCF_SSP_SCOPE_AUTO_FILTERS_SCHEDULE more frequent
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

func sspScopeAutoFiltersSchedule(cfg *config.Config) string {
	if cfg == nil || cfg.SSPScope == nil {
		return config.DefaultSSPScopeAutoFiltersSchedule
	}
	return cfg.SSPScope.AutoFiltersSchedule
}

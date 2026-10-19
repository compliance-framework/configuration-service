package worker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/compliance-framework/api/internal/config"
	"github.com/compliance-framework/api/internal/service/relational/sspservice"
	"github.com/compliance-framework/api/internal/service/relational/ssptemplates"
	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// JobTypeSSPTemplateDiscovery applies SSP templates to the latest evidence.
const JobTypeSSPTemplateDiscovery = "ssp_template_discovery"

// SSPTemplateDiscoveryArgs runs one template when TemplateID is set (a manual run), or
// every enabled template (the periodic run).
type SSPTemplateDiscoveryArgs struct {
	TemplateID *uuid.UUID `json:"template_id,omitempty"`
}

// Kind implements river.JobArgs.
func (SSPTemplateDiscoveryArgs) Kind() string { return JobTypeSSPTemplateDiscovery }

// SSPTemplateDiscoveryWorker creates the SSPs of SSP templates and learns their scopes.
type SSPTemplateDiscoveryWorker struct {
	db       *gorm.DB
	resolver ProfileControlResolver
	logger   *zap.SugaredLogger
}

// NewSSPTemplateDiscoveryWorker builds the worker; resolver resolves the controls of the
// profiles attached to created SSPs.
func NewSSPTemplateDiscoveryWorker(db *gorm.DB, resolver ProfileControlResolver, logger *zap.SugaredLogger) *SSPTemplateDiscoveryWorker {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &SSPTemplateDiscoveryWorker{db: db, resolver: resolver, logger: logger}
}

// Work implements the River worker function.
func (w *SSPTemplateDiscoveryWorker) Work(ctx context.Context, job *river.Job[SSPTemplateDiscoveryArgs]) error {
	discovery := ssptemplates.NewDiscovery(w.db, controlIDResolver(ctx, w.resolver))
	if job.Args.TemplateID == nil {
		if err := discovery.RunAll(ctx); err != nil {
			return fmt.Errorf("ssp template discovery: %w", err)
		}
		return nil
	}
	result, err := discovery.Run(ctx, *job.Args.TemplateID)
	w.logger.Infow("SSP template discovery run", "templateId", *job.Args.TemplateID,
		"created", result.Created, "updated", result.Updated, "failed", result.Failed)
	if err != nil {
		return fmt.Errorf("ssp template discovery %s: %w", *job.Args.TemplateID, err)
	}
	return nil
}

// controlIDResolver adapts the worker's profile resolver to the SSP service's: control
// ids de-duplicated case-insensitively, keeping the first casing seen.
func controlIDResolver(ctx context.Context, resolver ProfileControlResolver) sspservice.ControlResolver {
	return func(profileID uuid.UUID) ([]string, error) {
		keys, err := resolver.ResolveProfileControlKeys(ctx, profileID)
		if err != nil {
			return nil, err
		}
		seen := make(map[string]struct{}, len(keys))
		ids := make([]string, 0, len(keys))
		for _, key := range keys {
			folded := strings.ToLower(key.ControlID)
			if _, dup := seen[folded]; dup || key.ControlID == "" {
				continue
			}
			seen[folded] = struct{}{}
			ids = append(ids, key.ControlID)
		}
		return ids, nil
	}
}

// NewSSPTemplateDiscoveryPeriodicJob schedules discovery of every enabled template on the
// "scheduler" queue.
func NewSSPTemplateDiscoveryPeriodicJob(schedule string, logger *zap.SugaredLogger) *river.PeriodicJob {
	sched := parseCronScheduleWithFallback(schedule, config.DefaultSSPTemplateDiscoverySchedule, "ssp template discovery", logger)
	return river.NewPeriodicJob(
		sched,
		func() (river.JobArgs, *river.InsertOpts) {
			return &SSPTemplateDiscoveryArgs{}, &river.InsertOpts{
				Queue:       "scheduler",
				MaxAttempts: 3,
				// Deduplicated per hour: a CCF_SSP_TEMPLATE_DISCOVERY_SCHEDULE more frequent
				// than hourly is not honored.
				UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: time.Hour},
			}
		},
		&river.PeriodicJobOpts{RunOnStart: false},
	)
}

func sspTemplateDiscoverySchedule(cfg *config.Config) string {
	if cfg == nil || cfg.SSPTemplate == nil {
		return config.DefaultSSPTemplateDiscoverySchedule
	}
	return cfg.SSPTemplate.DiscoverySchedule
}

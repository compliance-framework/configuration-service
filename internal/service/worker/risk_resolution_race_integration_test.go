//go:build integration

package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/compliance-framework/api/internal/service/relational/risks"
	"github.com/compliance-framework/api/internal/tests"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RiskResolutionRaceIntegrationSuite struct {
	tests.IntegrationTestSuite
}

func TestRiskResolutionRaceIntegrationSuite(t *testing.T) {
	suite.Run(t, new(RiskResolutionRaceIntegrationSuite))
}

func (s *RiskResolutionRaceIntegrationSuite) SetupTest() {
	s.Require().NoError(s.Migrator.Refresh())
}

// Two streams of a risk pass at once: the second job runs between the first's count of the
// risk's remaining links and its commit, and the risk must still end up remediated.
func (s *RiskResolutionRaceIntegrationSuite) TestConcurrentPassingStreamsRemediateTheRisk() {
	ctx := context.Background()
	worker := NewRiskEvidenceWorker(s.DB, zap.NewNop().Sugar())
	first := createStreamEvidence(s.T(), s.DB, uuid.New(), relational.EvidenceStatusSatisfied, time.Now())
	second := createStreamEvidence(s.T(), s.DB, uuid.New(), relational.EvidenceStatusSatisfied, time.Now())
	risk := createRiskWithEvidenceLink(s.T(), s.DB, string(risks.RiskStatusOpen), first.UUID, nil)
	s.Require().NoError(s.DB.Create(&risks.RiskEvidenceLink{RiskID: *risk.ID, EvidenceID: second.UUID}).Error)

	const cb = "test:second_stream_resolves"
	var fired atomic.Bool
	var secondErr error
	secondDone := make(chan struct{})
	s.Require().NoError(s.DB.Callback().Query().After("gorm:query").Register(cb, func(db *gorm.DB) {
		_, isCount := db.Statement.Dest.(*int64)
		if !isCount || db.Statement.Table != (risks.RiskEvidenceLink{}).TableName() || !fired.CompareAndSwap(false, true) {
			return
		}
		go func() {
			defer close(secondDone)
			secondErr = worker.Work(ctx, makeWorkerJob(RiskProcessEvidenceArgs{EvidenceID: *second.ID}))
		}()
		// Let the second job finish, or block on a lock the first job holds.
		s.Eventually(func() bool {
			select {
			case <-secondDone:
				return true
			default:
			}
			var waiting int64
			err := s.DB.Raw(`SELECT count(*) FROM pg_stat_activity
				WHERE datname = current_database() AND wait_event_type = 'Lock'`).Scan(&waiting).Error
			return err == nil && waiting > 0
		}, 10*time.Second, 10*time.Millisecond)
	}))
	defer func() { s.Require().NoError(s.DB.Callback().Query().Remove(cb)) }()

	s.Require().NoError(worker.Work(ctx, makeWorkerJob(RiskProcessEvidenceArgs{EvidenceID: *first.ID})))
	s.Require().True(fired.Load())
	<-secondDone
	s.Require().NoError(secondErr)

	var stored risks.Risk
	s.Require().NoError(s.DB.First(&stored, "id = ?", risk.ID).Error)
	s.Equal(string(risks.RiskStatusRemediated), stored.Status)
	var links int64
	s.Require().NoError(s.DB.Model(&risks.RiskEvidenceLink{}).Where("risk_id = ?", risk.ID).Count(&links).Error)
	s.Zero(links)
}

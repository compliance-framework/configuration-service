//go:build integration

package worker

import (
	"context"
	"testing"
	"time"

	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/compliance-framework/api/internal/tests"
	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/stretchr/testify/suite"
	"gorm.io/datatypes"
)

type EvidenceBatchPruneWorkerIntegrationSuite struct {
	tests.IntegrationTestSuite
}

func TestEvidenceBatchPruneWorkerIntegrationSuite(t *testing.T) {
	suite.Run(t, new(EvidenceBatchPruneWorkerIntegrationSuite))
}

func (s *EvidenceBatchPruneWorkerIntegrationSuite) insertBatch(acceptedAt time.Time) uuid.UUID {
	batch := relational.EvidenceBatch{
		SubmittedBy: "user:prune@example.com",
		Signer:      datatypes.JSON(`{}`),
		Total:       1,
		ChunkSize:   1,
		AcceptedAt:  acceptedAt,
	}
	s.Require().NoError(s.DB.Create(&batch).Error)
	s.Require().NoError(s.DB.Create(&relational.EvidenceBatchItem{
		BatchID: *batch.ID,
		Index:   0,
		Status:  relational.EvidenceBatchItemSucceeded,
	}).Error)
	return *batch.ID
}

func (s *EvidenceBatchPruneWorkerIntegrationSuite) exists(id uuid.UUID) (bool, bool) {
	var batches, items int64
	s.Require().NoError(s.DB.Model(&relational.EvidenceBatch{}).Where("id = ?", id).Count(&batches).Error)
	s.Require().NoError(s.DB.Model(&relational.EvidenceBatchItem{}).Where("batch_id = ?", id).Count(&items).Error)
	return batches == 1, items == 1
}

func (s *EvidenceBatchPruneWorkerIntegrationSuite) TestWorkPrunesBatchesOlderThanRetention() {
	s.Require().NoError(s.Migrator.Refresh())
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	retention := 48 * time.Hour
	older := s.insertBatch(now.Add(-retention - time.Minute))
	newer := s.insertBatch(now.Add(-retention + time.Minute))

	w := NewEvidenceBatchPruneWorker(s.DB, retention, nil)
	w.now = func() time.Time { return now }
	s.Require().NoError(w.Work(context.Background(), &river.Job[EvidenceBatchPruneArgs]{}))

	batch, item := s.exists(older)
	s.False(batch, "a batch accepted before now-retention is pruned")
	s.False(item, "with its items")
	batch, item = s.exists(newer)
	s.True(batch, "a batch accepted after now-retention is kept")
	s.True(item)
}

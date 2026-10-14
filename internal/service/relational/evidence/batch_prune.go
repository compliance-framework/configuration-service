package evidence

import (
	"context"
	"fmt"
	"time"

	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// pruneBatchesPerRound bounds how many batches one prune transaction deletes.
const pruneBatchesPerRound = 500

// PruneBatches deletes the evidence batches accepted before acceptedBefore, with their
// items, and returns how many batches it deleted. The evidence the batches stored is
// kept. Batches still pending are deleted too: a batch whose jobs were never queued
// (see BatchService.Submit) would otherwise stay pending forever, and nobody polls a
// batch that old. A chunk job that runs after its batch is gone does nothing.
func PruneBatches(ctx context.Context, db *gorm.DB, acceptedBefore time.Time) (int64, error) {
	var deleted int64
	for {
		var ids []uuid.UUID
		if err := db.WithContext(ctx).Model(&relational.EvidenceBatch{}).
			Where("accepted_at < ?", acceptedBefore).
			Order("accepted_at").
			Limit(pruneBatchesPerRound).
			Pluck("id", &ids).Error; err != nil {
			return deleted, fmt.Errorf("list evidence batches to prune: %w", err)
		}
		if len(ids) == 0 {
			return deleted, nil
		}
		if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("batch_id IN ?", ids).Delete(&relational.EvidenceBatchItem{}).Error; err != nil {
				return err
			}
			return tx.Where("id IN ?", ids).Delete(&relational.EvidenceBatch{}).Error
		}); err != nil {
			return deleted, fmt.Errorf("prune evidence batches: %w", err)
		}
		deleted += int64(len(ids))
		if len(ids) < pruneBatchesPerRound {
			return deleted, nil
		}
	}
}

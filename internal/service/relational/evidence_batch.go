package relational

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// EvidenceBatchItemStatus is where one evidence of a batch is in its asynchronous
// processing.
type EvidenceBatchItemStatus string

const (
	// EvidenceBatchItemPending items are accepted and waiting to be processed.
	EvidenceBatchItemPending EvidenceBatchItemStatus = "pending"
	// EvidenceBatchItemSucceeded items were stored as evidence (EvidenceID).
	EvidenceBatchItemSucceeded EvidenceBatchItemStatus = "succeeded"
	// EvidenceBatchItemFailed items were rejected or could not be stored (Error).
	EvidenceBatchItemFailed EvidenceBatchItemStatus = "failed"
)

// EvidenceBatch is a list of evidence submitted in one request and stored
// asynchronously. Its progress is derived from its items.
type EvidenceBatch struct {
	UUIDModel
	// SubmittedBy identifies the authenticated caller that submitted the batch
	// ("user:<subject>" or "agent:<agent id>"); only that caller may read it back.
	SubmittedBy string `gorm:"not null;index"`
	// Signer is the submitter's identity at acceptance time, used to sign each evidence.
	Signer datatypes.JSON `gorm:"not null"`
	Total  int            `gorm:"not null"`
	// ChunkSize is how many items, by index, each background job of this batch processes.
	ChunkSize int `gorm:"not null"`
	// AcceptedAt is when the API accepted the batch; evidence signatures are dated then,
	// while the submitter's token was valid.
	AcceptedAt time.Time `gorm:"not null"`
}

// EvidenceBatchItem is one evidence of a batch, keyed by its position in the request.
type EvidenceBatchItem struct {
	BatchID uuid.UUID               `gorm:"type:uuid;primaryKey"`
	Index   int                     `gorm:"column:item_index;primaryKey;autoIncrement:false"`
	Status  EvidenceBatchItemStatus `gorm:"not null;index"`
	// Payload is the submitted evidence, kept only until the item is processed.
	Payload    datatypes.JSON
	EvidenceID *uuid.UUID `gorm:"type:uuid"`
	Error      *string
	UpdatedAt  time.Time
}

package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/compliance-framework/api/internal/authn"
	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	// ErrBatchSubmitterRequired is returned when a batch is submitted without an
	// authenticated user or agent: its evidence is signed and only its submitter may
	// read the batch back.
	ErrBatchSubmitterRequired = errors.New("evidence batches need an authenticated user or agent")
	// ErrBatchNotFound is returned for a batch that doesn't exist or wasn't submitted by
	// the caller.
	ErrBatchNotFound = errors.New("evidence batch not found")

	// errBatchItemAlreadyProcessed rolls back an evidence whose item another job
	// already processed.
	errBatchItemAlreadyProcessed = errors.New("evidence batch item already processed")
)

// batchItemInternalError is what a failed item reports when it failed for a reason
// other than its content; the cause is logged, not returned.
const batchItemInternalError = "internal error while storing the evidence"

// BatchItemRejectedError marks an item the submitter has to fix (invalid evidence, an
// unknown subject, ...). The item fails with this message and is not retried.
type BatchItemRejectedError struct {
	Err error
}

func (e *BatchItemRejectedError) Error() string { return e.Err.Error() }

func (e *BatchItemRejectedError) Unwrap() error { return e.Err }

// RejectBatchItem wraps err as a BatchItemRejectedError.
func RejectBatchItem(err error) error {
	return &BatchItemRejectedError{Err: err}
}

// BatchItemFunc stores one batch item as evidence. It must hand onCreated to
// EvidenceService.Create (CreateEvidenceParams.OnCreated) so the item is marked done in
// the transaction that stores its evidence, and return a BatchItemRejectedError for an
// item the submitter has to fix. Any other error is retried.
type BatchItemFunc func(ctx context.Context, payload json.RawMessage, signer *SignerContext, onCreated func(tx *gorm.DB, evidence *relational.Evidence) error) error

// BatchJobEnqueuer starts the background jobs that process a batch, one per chunk.
type BatchJobEnqueuer interface {
	// EvidenceBatchesAvailable reports why batches can't be processed (no worker
	// service), or nil.
	EvidenceBatchesAvailable() error
	EnqueueEvidenceBatchChunks(ctx context.Context, batchID uuid.UUID, chunks int) error
}

// BatchStatus is a batch's progress, derived from its items.
type BatchStatus string

const (
	// BatchStatusPending batches have no item processed yet.
	BatchStatusPending BatchStatus = "pending"
	// BatchStatusProcessing batches have some items processed and some pending.
	BatchStatusProcessing BatchStatus = "processing"
	// BatchStatusCompleted batches have every item succeeded or failed.
	BatchStatusCompleted BatchStatus = "completed"
)

// BatchItemResult is one item's outcome.
type BatchItemResult struct {
	Index      int
	Status     relational.EvidenceBatchItemStatus
	EvidenceID *uuid.UUID
	Error      string
}

// BatchSummary is a batch's progress and the outcome of each item.
type BatchSummary struct {
	ID         uuid.UUID
	Status     BatchStatus
	Total      int
	Pending    int
	Succeeded  int
	Failed     int
	AcceptedAt time.Time
	Items      []BatchItemResult
}

// BatchService accepts evidence batches and stores their evidence in the background.
type BatchService struct {
	db          *gorm.DB
	logger      *zap.SugaredLogger
	enqueuer    BatchJobEnqueuer
	processItem BatchItemFunc
	chunkSize   int
	now         func() time.Time
}

func NewBatchService(db *gorm.DB, logger *zap.SugaredLogger, enqueuer BatchJobEnqueuer, processItem BatchItemFunc, chunkSize int) *BatchService {
	if chunkSize <= 0 {
		chunkSize = 1
	}
	return &BatchService{
		db:          db,
		logger:      logger,
		enqueuer:    enqueuer,
		processItem: processItem,
		chunkSize:   chunkSize,
		// Truncated to what Postgres stores, so the time returned on submit is the one
		// signatures are later dated with.
		now: func() time.Time { return time.Now().UTC().Truncate(time.Microsecond) },
	}
}

// Available reports why batches can't be accepted right now, or nil.
func (s *BatchService) Available() error {
	if s.enqueuer == nil {
		return errors.New("evidence batches need the worker service")
	}
	return s.enqueuer.EvidenceBatchesAvailable()
}

// Submit stores the payloads as a pending batch and starts its background jobs.
func (s *BatchService) Submit(ctx context.Context, payloads []json.RawMessage, signer *SignerContext) (*BatchSummary, error) {
	submittedBy := BatchSubmitter(signer)
	if submittedBy == "" {
		return nil, ErrBatchSubmitterRequired
	}
	if err := s.Available(); err != nil {
		return nil, err
	}
	signerJSON, err := encodeBatchSigner(signer)
	if err != nil {
		return nil, fmt.Errorf("encode evidence batch signer: %w", err)
	}

	batch := relational.EvidenceBatch{
		SubmittedBy: submittedBy,
		Signer:      signerJSON,
		Total:       len(payloads),
		ChunkSize:   s.chunkSize,
		AcceptedAt:  s.now(),
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&batch).Error; err != nil {
			return err
		}
		if len(payloads) == 0 {
			return nil
		}
		items := make([]relational.EvidenceBatchItem, len(payloads))
		for i, payload := range payloads {
			items[i] = relational.EvidenceBatchItem{
				BatchID: *batch.ID,
				Index:   i,
				Status:  relational.EvidenceBatchItemPending,
				Payload: datatypes.JSON(payload),
			}
		}
		return tx.CreateInBatches(&items, 500).Error
	})
	if err != nil {
		return nil, fmt.Errorf("store evidence batch: %w", err)
	}

	chunks := (len(payloads) + s.chunkSize - 1) / s.chunkSize
	if chunks > 0 {
		if err := s.enqueuer.EnqueueEvidenceBatchChunks(ctx, *batch.ID, chunks); err != nil {
			// Nothing will process the batch: fail its items so it reads as completed.
			message := "the evidence could not be queued for processing"
			if failErr := s.db.WithContext(ctx).Model(&relational.EvidenceBatchItem{}).
				Where("batch_id = ? AND status = ?", *batch.ID, relational.EvidenceBatchItemPending).
				Updates(map[string]any{"status": relational.EvidenceBatchItemFailed, "error": message, "payload": nil}).Error; failErr != nil {
				s.logger.Errorw("Failed to fail the items of an evidence batch that could not be queued", "batch_id", *batch.ID, "error", failErr)
			}
			return nil, fmt.Errorf("enqueue evidence batch: %w", err)
		}
	}

	return &BatchSummary{
		ID:         *batch.ID,
		Status:     batchStatus(batch.Total, batch.Total),
		Total:      batch.Total,
		Pending:    batch.Total,
		AcceptedAt: batch.AcceptedAt,
		Items:      []BatchItemResult{},
	}, nil
}

// ProcessChunk stores the pending items of one chunk of a batch. An item rejected by
// BatchItemFunc fails with its message. Any other error stops the chunk so the job is
// retried; on the job's final attempt the item fails instead. Each item is marked done
// in the transaction that stores its evidence, so a retried or duplicated job never
// stores an item twice.
func (s *BatchService) ProcessChunk(ctx context.Context, batchID uuid.UUID, chunk int, finalAttempt bool) error {
	var batch relational.EvidenceBatch
	if err := s.db.WithContext(ctx).First(&batch, "id = ?", batchID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("load evidence batch: %w", err)
	}
	first := chunk * batch.ChunkSize
	end := first + batch.ChunkSize

	signer, err := decodeBatchSigner(batch.Signer, batch.AcceptedAt)
	if err != nil {
		return s.failChunkOnFinalAttempt(ctx, batchID, first, end, finalAttempt, fmt.Errorf("decode evidence batch signer: %w", err))
	}

	var items []relational.EvidenceBatchItem
	if err := s.db.WithContext(ctx).
		Where("batch_id = ? AND status = ? AND item_index >= ? AND item_index < ?", batchID, relational.EvidenceBatchItemPending, first, end).
		Order("item_index").
		Find(&items).Error; err != nil {
		return s.failChunkOnFinalAttempt(ctx, batchID, first, end, finalAttempt, fmt.Errorf("load evidence batch items: %w", err))
	}

	for _, item := range items {
		onCreated := func(tx *gorm.DB, evidence *relational.Evidence) error {
			result := tx.Model(&relational.EvidenceBatchItem{}).
				Where("batch_id = ? AND item_index = ? AND status = ?", batchID, item.Index, relational.EvidenceBatchItemPending).
				Updates(map[string]any{"status": relational.EvidenceBatchItemSucceeded, "evidence_id": evidence.ID, "payload": nil})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return errBatchItemAlreadyProcessed
			}
			return nil
		}

		err := s.processItem(ctx, json.RawMessage(item.Payload), signer, onCreated)
		if err == nil || errors.Is(err, errBatchItemAlreadyProcessed) {
			continue
		}

		var rejected *BatchItemRejectedError
		switch {
		case errors.As(err, &rejected):
			if err := s.failItem(ctx, batchID, item.Index, rejected.Error()); err != nil {
				return err
			}
		case finalAttempt && ctx.Err() == nil:
			s.logger.Errorw("Failed to store evidence batch item", "batch_id", batchID, "index", item.Index, "error", err)
			if err := s.failItem(ctx, batchID, item.Index, batchItemInternalError); err != nil {
				return err
			}
		default:
			return fmt.Errorf("store evidence batch item %d: %w", item.Index, err)
		}
	}
	return nil
}

// failChunkOnFinalAttempt returns err so the job is retried, unless this is its final
// attempt: then the chunk's pending items fail, since nothing would process them again.
func (s *BatchService) failChunkOnFinalAttempt(ctx context.Context, batchID uuid.UUID, first, end int, finalAttempt bool, err error) error {
	if !finalAttempt || ctx.Err() != nil {
		return err
	}
	s.logger.Errorw("Failed to process evidence batch chunk", "batch_id", batchID, "first_index", first, "error", err)
	if failErr := s.db.WithContext(ctx).Model(&relational.EvidenceBatchItem{}).
		Where("batch_id = ? AND status = ? AND item_index >= ? AND item_index < ?", batchID, relational.EvidenceBatchItemPending, first, end).
		Updates(map[string]any{"status": relational.EvidenceBatchItemFailed, "error": batchItemInternalError, "payload": nil}).Error; failErr != nil {
		return errors.Join(err, failErr)
	}
	return nil
}

func (s *BatchService) failItem(ctx context.Context, batchID uuid.UUID, index int, message string) error {
	if err := s.db.WithContext(ctx).Model(&relational.EvidenceBatchItem{}).
		Where("batch_id = ? AND item_index = ? AND status = ?", batchID, index, relational.EvidenceBatchItemPending).
		Updates(map[string]any{"status": relational.EvidenceBatchItemFailed, "error": message, "payload": nil}).Error; err != nil {
		return fmt.Errorf("fail evidence batch item %d: %w", index, err)
	}
	return nil
}

// Get returns a batch submitted by submittedBy (see BatchSubmitter) with every item's
// outcome, or ErrBatchNotFound.
func (s *BatchService) Get(ctx context.Context, batchID uuid.UUID, submittedBy string) (*BatchSummary, error) {
	if submittedBy == "" {
		return nil, ErrBatchNotFound
	}
	var batch relational.EvidenceBatch
	if err := s.db.WithContext(ctx).
		Where("id = ? AND submitted_by = ?", batchID, submittedBy).
		First(&batch).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBatchNotFound
		}
		return nil, fmt.Errorf("load evidence batch: %w", err)
	}

	var items []relational.EvidenceBatchItem
	if err := s.db.WithContext(ctx).
		Select("item_index", "status", "evidence_id", "error").
		Where("batch_id = ?", batchID).
		Order("item_index").
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("load evidence batch items: %w", err)
	}

	summary := &BatchSummary{
		ID:         *batch.ID,
		Total:      batch.Total,
		AcceptedAt: batch.AcceptedAt.UTC(),
		Items:      make([]BatchItemResult, 0, len(items)),
	}
	for _, item := range items {
		result := BatchItemResult{Index: item.Index, Status: item.Status, EvidenceID: item.EvidenceID}
		if item.Error != nil {
			result.Error = *item.Error
		}
		switch item.Status {
		case relational.EvidenceBatchItemSucceeded:
			summary.Succeeded++
		case relational.EvidenceBatchItemFailed:
			summary.Failed++
		default:
			summary.Pending++
		}
		summary.Items = append(summary.Items, result)
	}
	summary.Status = batchStatus(summary.Total, summary.Pending)
	return summary, nil
}

func batchStatus(total, pending int) BatchStatus {
	switch pending {
	case 0:
		return BatchStatusCompleted
	case total:
		return BatchStatusPending
	default:
		return BatchStatusProcessing
	}
}

// BatchSubmitter is the identity a batch is stored under: "user:<subject>" for a user,
// "agent:<agent id>" for an agent, or "" when the signer identifies no one.
func BatchSubmitter(signer *SignerContext) string {
	if signer == nil {
		return ""
	}
	if signer.User != nil && signer.User.Claims != nil && signer.User.Claims.Subject != "" {
		return "user:" + signer.User.Claims.Subject
	}
	if signer.Agent != nil {
		if signer.Agent.Agent != nil && signer.Agent.Agent.ID != nil {
			return "agent:" + signer.Agent.Agent.ID.String()
		}
		if signer.Agent.Claims != nil && signer.Agent.Claims.AgentID != "" {
			return "agent:" + signer.Agent.Claims.AgentID
		}
	}
	return ""
}

// batchSigner is the part of a SignerContext that signing reads, stored with a batch so
// its evidence is signed in the background exactly as it would have been in the request.
type batchSigner struct {
	User        *authn.UserClaims  `json:"user,omitempty"`
	AgentClaims *authn.AgentClaims `json:"agent-claims,omitempty"`
	AgentID     *uuid.UUID         `json:"agent-id,omitempty"`
	AgentName   string             `json:"agent-name,omitempty"`
	KeyID       *uuid.UUID         `json:"key-id,omitempty"`
}

func encodeBatchSigner(signer *SignerContext) (datatypes.JSON, error) {
	stored := batchSigner{}
	if signer.User != nil {
		stored.User = signer.User.Claims
	}
	if signer.Agent != nil {
		stored.AgentClaims = signer.Agent.Claims
		if signer.Agent.Agent != nil {
			stored.AgentID = signer.Agent.Agent.ID
			stored.AgentName = signer.Agent.Agent.Name
		}
		if signer.Agent.Key != nil {
			stored.KeyID = signer.Agent.Key.ID
		}
	}
	encoded, err := json.Marshal(stored)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(encoded), nil
}

func decodeBatchSigner(raw datatypes.JSON, acceptedAt time.Time) (*SignerContext, error) {
	var stored batchSigner
	if err := json.Unmarshal(raw, &stored); err != nil {
		return nil, err
	}
	signer := &SignerContext{AcceptedAt: acceptedAt}
	if stored.User != nil {
		signer.User = &UserSignerContext{Claims: stored.User}
	}
	if stored.AgentClaims != nil || stored.AgentID != nil || stored.KeyID != nil {
		agent := &AgentSignerContext{Claims: stored.AgentClaims}
		if stored.AgentID != nil {
			agent.Agent = &relational.Agent{UUIDModel: relational.UUIDModel{ID: stored.AgentID}, Name: stored.AgentName}
		}
		if stored.KeyID != nil {
			agent.Key = &relational.AgentServiceAccountKey{UUIDModel: relational.UUIDModel{ID: stored.KeyID}}
		}
		signer.Agent = agent
	}
	return signer, nil
}

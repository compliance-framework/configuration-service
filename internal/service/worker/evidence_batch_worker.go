package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
)

const (
	JobTypeEvidenceBatchChunk = "evidence_batch_chunk"
	EvidenceBatchQueue        = "evidence"
	EvidenceBatchMaxAttempts  = 5

	// evidenceBatchProcessorWait is how long a chunk waits when it runs before the API
	// has registered its processor (the worker starts before the handlers).
	evidenceBatchProcessorWait = 10 * time.Second
)

// ErrEvidenceBatchWorkerDisabled is returned when evidence batches can't be processed
// because the worker service is disabled.
var ErrEvidenceBatchWorkerDisabled = errors.New("evidence batches need the worker service, which is disabled")

// EvidenceBatchChunkArgs identifies one chunk of an evidence batch.
type EvidenceBatchChunkArgs struct {
	BatchID uuid.UUID `json:"batch_id" river:"unique"`
	Chunk   int       `json:"chunk" river:"unique"`
}

func (EvidenceBatchChunkArgs) Kind() string { return JobTypeEvidenceBatchChunk }

func (EvidenceBatchChunkArgs) Timeout() time.Duration { return 15 * time.Minute }

func JobInsertOptionsForEvidenceBatchChunk() *river.InsertOpts {
	return &river.InsertOpts{
		Queue:       EvidenceBatchQueue,
		MaxAttempts: EvidenceBatchMaxAttempts,
		UniqueOpts: river.UniqueOpts{
			ByArgs: true,
		},
	}
}

// EvidenceBatchProcessor stores the evidence of one chunk of a batch
// (evidence.BatchService). finalAttempt is true on the job's last attempt, when items
// that keep failing must be failed rather than left pending.
type EvidenceBatchProcessor interface {
	ProcessChunk(ctx context.Context, batchID uuid.UUID, chunk int, finalAttempt bool) error
}

// EvidenceBatchWorker runs evidence batch chunks through the processor the API
// registers with SetProcessor.
type EvidenceBatchWorker struct {
	river.WorkerDefaults[EvidenceBatchChunkArgs]

	mu        sync.RWMutex
	processor EvidenceBatchProcessor
}

func NewEvidenceBatchWorker() *EvidenceBatchWorker {
	return &EvidenceBatchWorker{}
}

// SetProcessor registers the processor chunks run through.
func (w *EvidenceBatchWorker) SetProcessor(processor EvidenceBatchProcessor) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.processor = processor
}

func (w *EvidenceBatchWorker) Work(ctx context.Context, job *river.Job[EvidenceBatchChunkArgs]) error {
	w.mu.RLock()
	processor := w.processor
	w.mu.RUnlock()
	if processor == nil {
		return river.JobSnooze(evidenceBatchProcessorWait)
	}
	finalAttempt := job.Attempt >= job.MaxAttempts
	return processor.ProcessChunk(ctx, job.Args.BatchID, job.Args.Chunk, finalAttempt)
}

// SetEvidenceBatchProcessor registers the processor evidence batch chunks run through.
// It does nothing when the worker service is disabled.
func (s *Service) SetEvidenceBatchProcessor(processor EvidenceBatchProcessor) {
	if s == nil || s.evidenceBatchWorker == nil {
		return
	}
	s.evidenceBatchWorker.SetProcessor(processor)
}

// EvidenceBatchesAvailable reports whether evidence batches can be processed.
func (s *Service) EvidenceBatchesAvailable() error {
	if s == nil || s.config == nil || !s.config.Enabled || s.client == nil || s.evidenceBatchWorker == nil {
		return ErrEvidenceBatchWorkerDisabled
	}
	return nil
}

// EnqueueEvidenceBatchChunks starts one job per chunk of an evidence batch.
func (s *Service) EnqueueEvidenceBatchChunks(ctx context.Context, batchID uuid.UUID, chunks int) error {
	if err := s.EvidenceBatchesAvailable(); err != nil {
		return err
	}
	if chunks <= 0 {
		return nil
	}
	params := make([]river.InsertManyParams, 0, chunks)
	for chunk := 0; chunk < chunks; chunk++ {
		params = append(params, river.InsertManyParams{
			Args:       EvidenceBatchChunkArgs{BatchID: batchID, Chunk: chunk},
			InsertOpts: JobInsertOptionsForEvidenceBatchChunk(),
		})
	}
	if _, err := s.client.InsertMany(ctx, params); err != nil {
		return fmt.Errorf("failed to enqueue evidence batch chunk jobs for batch %s: %w", batchID, err)
	}
	return nil
}

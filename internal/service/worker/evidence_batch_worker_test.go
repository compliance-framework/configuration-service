package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/compliance-framework/api/internal/config"
	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/require"
)

type recordingEvidenceBatchProcessor struct {
	batchID      uuid.UUID
	chunk        int
	finalAttempt bool
	err          error
}

func (p *recordingEvidenceBatchProcessor) ProcessChunk(_ context.Context, batchID uuid.UUID, chunk int, finalAttempt bool) error {
	p.batchID, p.chunk, p.finalAttempt = batchID, chunk, finalAttempt
	return p.err
}

func evidenceBatchJob(batchID uuid.UUID, chunk, attempt, maxAttempts int) *river.Job[EvidenceBatchChunkArgs] {
	return &river.Job[EvidenceBatchChunkArgs]{
		JobRow: &rivertype.JobRow{Attempt: attempt, MaxAttempts: maxAttempts},
		Args:   EvidenceBatchChunkArgs{BatchID: batchID, Chunk: chunk},
	}
}

func TestEvidenceBatchWorkerSnoozesUntilProcessorIsSet(t *testing.T) {
	w := NewEvidenceBatchWorker()

	err := w.Work(context.Background(), evidenceBatchJob(uuid.New(), 0, 1, EvidenceBatchMaxAttempts))

	var snooze *rivertype.JobSnoozeError
	require.True(t, errors.As(err, &snooze))
	require.Equal(t, evidenceBatchProcessorWait, snooze.Duration)
}

func TestEvidenceBatchWorkerRunsChunkThroughProcessor(t *testing.T) {
	w := NewEvidenceBatchWorker()
	processor := &recordingEvidenceBatchProcessor{err: errors.New("db down")}
	w.SetProcessor(processor)
	batchID := uuid.New()

	err := w.Work(context.Background(), evidenceBatchJob(batchID, 3, 2, EvidenceBatchMaxAttempts))

	require.EqualError(t, err, "db down")
	require.Equal(t, batchID, processor.batchID)
	require.Equal(t, 3, processor.chunk)
	require.False(t, processor.finalAttempt)
}

func TestEvidenceBatchWorkerFlagsFinalAttempt(t *testing.T) {
	w := NewEvidenceBatchWorker()
	processor := &recordingEvidenceBatchProcessor{}
	w.SetProcessor(processor)

	require.NoError(t, w.Work(context.Background(), evidenceBatchJob(uuid.New(), 0, EvidenceBatchMaxAttempts, EvidenceBatchMaxAttempts)))
	require.True(t, processor.finalAttempt)
}

func TestDisabledServiceRejectsEvidenceBatches(t *testing.T) {
	service := &Service{config: &config.WorkerConfig{Enabled: false}}

	require.ErrorIs(t, service.EvidenceBatchesAvailable(), ErrEvidenceBatchWorkerDisabled)
	require.ErrorIs(t, service.EnqueueEvidenceBatchChunks(context.Background(), uuid.New(), 2), ErrEvidenceBatchWorkerDisabled)
	// Registering a processor on a disabled service is a no-op, not a panic.
	service.SetEvidenceBatchProcessor(&recordingEvidenceBatchProcessor{})
}

func TestBuildRiverConfig_EvidenceBatchQueue(t *testing.T) {
	cfg := config.DefaultWorkerConfig()

	defaults := buildRiverConfig(cfg, river.NewWorkers(), nil, nil, nil)
	require.Equal(t, config.DefaultEvidenceBatchWorkers, defaults.Queues[EvidenceBatchQueue].MaxWorkers)

	configured := buildRiverConfig(cfg, river.NewWorkers(), nil, nil, &config.EvidenceBatchConfig{Workers: 9})
	require.Equal(t, 9, configured.Queues[EvidenceBatchQueue].MaxWorkers)
}

func TestJobInsertOptionsForEvidenceBatchChunk(t *testing.T) {
	opts := JobInsertOptionsForEvidenceBatchChunk()

	require.Equal(t, EvidenceBatchQueue, opts.Queue)
	require.Equal(t, EvidenceBatchMaxAttempts, opts.MaxAttempts)
	require.True(t, opts.UniqueOpts.ByArgs)
}

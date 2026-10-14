//go:build integration

package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/compliance-framework/api/internal/authn"
	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/compliance-framework/api/internal/tests"
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type EvidenceBatchIntegrationSuite struct {
	tests.IntegrationTestSuite
}

func TestEvidenceBatchIntegration(t *testing.T) {
	suite.Run(t, new(EvidenceBatchIntegrationSuite))
}

type fakeBatchEnqueuer struct {
	mu          sync.Mutex
	available   error
	enqueueErr  error
	batchIDs    []uuid.UUID
	chunkCounts []int
}

func (e *fakeBatchEnqueuer) EvidenceBatchesAvailable() error { return e.available }

func (e *fakeBatchEnqueuer) EnqueueEvidenceBatchChunks(_ context.Context, batchID uuid.UUID, chunks int) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.batchIDs = append(e.batchIDs, batchID)
	e.chunkCounts = append(e.chunkCounts, chunks)
	return e.enqueueErr
}

// testBatchPayload is the item format of these tests: a title, plus switches that make
// the item func reject the item or fail transiently.
type testBatchPayload struct {
	Title     string `json:"title"`
	Reject    bool   `json:"reject,omitempty"`
	Transient bool   `json:"transient,omitempty"`
}

func batchPayloads(items ...testBatchPayload) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(items))
	for _, item := range items {
		raw, _ := json.Marshal(item)
		out = append(out, raw)
	}
	return out
}

// newBatchService returns a batch service whose items are stored through a real
// EvidenceService, so signing and the OnCreated transaction are exercised.
// transientFailures counts down the transient failures left before such items succeed.
func (suite *EvidenceBatchIntegrationSuite) newBatchService(enqueuer BatchJobEnqueuer, chunkSize int, transientFailures *int) *BatchService {
	evidenceService := NewEvidenceService(suite.DB, zap.NewNop().Sugar(), suite.Config, nil)
	var mu sync.Mutex
	itemFunc := func(ctx context.Context, payload json.RawMessage, signer *SignerContext, onCreated func(tx *gorm.DB, evidence *relational.Evidence) error) error {
		var item testBatchPayload
		if err := json.Unmarshal(payload, &item); err != nil {
			return RejectBatchItem(err)
		}
		if item.Reject {
			return RejectBatchItem(errors.New("title " + item.Title + " is not allowed"))
		}
		if item.Transient {
			mu.Lock()
			fail := transientFailures != nil && *transientFailures > 0
			if fail {
				*transientFailures--
			}
			mu.Unlock()
			if fail {
				return errors.New("connection reset")
			}
		}
		now := time.Now().UTC()
		_, err := evidenceService.Create(ctx, CreateEvidenceParams{
			Evidence: relational.Evidence{
				UUID:   uuid.New(),
				Title:  item.Title,
				Start:  now,
				End:    now,
				Status: datatypes.NewJSONType(oscalTypes_1_1_3.ObjectiveStatus{State: "satisfied"}),
			},
			Signer:    signer,
			OnCreated: onCreated,
		})
		return err
	}
	return NewBatchService(suite.DB, zap.NewNop().Sugar(), enqueuer, itemFunc, chunkSize)
}

// userSigner is a user whose token expires a minute from now.
func userSigner(email string) *SignerContext {
	now := time.Now()
	return NewUserSignerContextFromClaims(&authn.UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   email,
			IssuedAt:  jwt.NewNumericDate(now.Add(-time.Minute)),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
		},
		TokenKind: authn.TokenKindUser,
		GivenName: "Batch",
	})
}

func (suite *EvidenceBatchIntegrationSuite) evidenceCount() int64 {
	var count int64
	suite.Require().NoError(suite.DB.Model(&relational.Evidence{}).Count(&count).Error)
	return count
}

func (suite *EvidenceBatchIntegrationSuite) TestSubmitStoresPendingBatchAndEnqueuesChunks() {
	suite.Require().NoError(suite.Migrator.Refresh())
	enqueuer := &fakeBatchEnqueuer{}
	svc := suite.newBatchService(enqueuer, 2, nil)
	signer := userSigner("submitter@example.com")

	summary, err := svc.Submit(context.Background(), batchPayloads(
		testBatchPayload{Title: "a"}, testBatchPayload{Title: "b"}, testBatchPayload{Title: "c"},
		testBatchPayload{Title: "d"}, testBatchPayload{Title: "e"},
	), signer)
	suite.Require().NoError(err)
	suite.Equal(BatchStatusPending, summary.Status)
	suite.Equal(5, summary.Total)
	suite.Equal(5, summary.Pending)
	suite.Equal([]uuid.UUID{summary.ID}, enqueuer.batchIDs)
	suite.Equal([]int{3}, enqueuer.chunkCounts)

	got, err := svc.Get(context.Background(), summary.ID, BatchSubmitter(signer))
	suite.Require().NoError(err)
	suite.Equal(BatchStatusPending, got.Status)
	suite.Equal(summary.AcceptedAt, got.AcceptedAt)
	suite.Require().Len(got.Items, 5)
	for i, item := range got.Items {
		suite.Equal(i, item.Index)
		suite.Equal(relational.EvidenceBatchItemPending, item.Status)
	}

	_, err = svc.Get(context.Background(), summary.ID, BatchSubmitter(userSigner("someone-else@example.com")))
	suite.ErrorIs(err, ErrBatchNotFound)
}

func (suite *EvidenceBatchIntegrationSuite) TestSubmitRejectsWhatCannotBeProcessed() {
	suite.Require().NoError(suite.Migrator.Refresh())

	_, err := suite.newBatchService(&fakeBatchEnqueuer{}, 2, nil).
		Submit(context.Background(), batchPayloads(testBatchPayload{Title: "a"}), nil)
	suite.ErrorIs(err, ErrBatchSubmitterRequired)

	disabled := errors.New("worker disabled")
	_, err = suite.newBatchService(&fakeBatchEnqueuer{available: disabled}, 2, nil).
		Submit(context.Background(), batchPayloads(testBatchPayload{Title: "a"}), userSigner("u@example.com"))
	suite.ErrorIs(err, disabled)

	var batches int64
	suite.Require().NoError(suite.DB.Model(&relational.EvidenceBatch{}).Count(&batches).Error)
	suite.Zero(batches, "nothing is stored for a batch that can't be processed")
}

func (suite *EvidenceBatchIntegrationSuite) TestSubmitFailsItemsWhenEnqueueFails() {
	suite.Require().NoError(suite.Migrator.Refresh())
	enqueuer := &fakeBatchEnqueuer{enqueueErr: errors.New("queue down")}
	svc := suite.newBatchService(enqueuer, 2, nil)
	signer := userSigner("submitter@example.com")

	_, err := svc.Submit(context.Background(), batchPayloads(testBatchPayload{Title: "a"}, testBatchPayload{Title: "b"}), signer)
	suite.Require().Error(err)
	suite.Require().Len(enqueuer.batchIDs, 1)

	got, err := svc.Get(context.Background(), enqueuer.batchIDs[0], BatchSubmitter(signer))
	suite.Require().NoError(err)
	suite.Equal(BatchStatusCompleted, got.Status)
	suite.Equal(2, got.Failed)
	suite.Equal("the evidence could not be queued for processing", got.Items[0].Error)
}

func (suite *EvidenceBatchIntegrationSuite) TestProcessChunkStoresEachItemOnce() {
	suite.Require().NoError(suite.Migrator.Refresh())
	svc := suite.newBatchService(&fakeBatchEnqueuer{}, 2, nil)
	signer := userSigner("submitter@example.com")
	ctx := context.Background()

	summary, err := svc.Submit(ctx, batchPayloads(
		testBatchPayload{Title: "a"}, testBatchPayload{Title: "b"},
		testBatchPayload{Title: "c", Reject: true}, testBatchPayload{Title: "d"},
		testBatchPayload{Title: "e"},
	), signer)
	suite.Require().NoError(err)

	suite.Require().NoError(svc.ProcessChunk(ctx, summary.ID, 0, false))
	got, err := svc.Get(ctx, summary.ID, BatchSubmitter(signer))
	suite.Require().NoError(err)
	suite.Equal(BatchStatusProcessing, got.Status)
	suite.Equal(2, got.Succeeded)
	suite.Equal(3, got.Pending)

	suite.Require().NoError(svc.ProcessChunk(ctx, summary.ID, 1, false))
	suite.Require().NoError(svc.ProcessChunk(ctx, summary.ID, 2, false))
	got, err = svc.Get(ctx, summary.ID, BatchSubmitter(signer))
	suite.Require().NoError(err)
	suite.Equal(BatchStatusCompleted, got.Status)
	suite.Equal(4, got.Succeeded)
	suite.Equal(1, got.Failed)
	suite.Equal(relational.EvidenceBatchItemFailed, got.Items[2].Status)
	suite.Equal("title c is not allowed", got.Items[2].Error)
	suite.Nil(got.Items[2].EvidenceID)
	suite.EqualValues(4, suite.evidenceCount())

	// Payloads are dropped once processed.
	var withPayload int64
	suite.Require().NoError(suite.DB.Model(&relational.EvidenceBatchItem{}).
		Where("batch_id = ? AND payload IS NOT NULL", summary.ID).Count(&withPayload).Error)
	suite.Zero(withPayload)

	// A retried job stores nothing twice.
	suite.Require().NoError(svc.ProcessChunk(ctx, summary.ID, 0, false))
	suite.EqualValues(4, suite.evidenceCount())

	// Evidence is signed as the submitter, dated when the batch was accepted.
	evidenceService := NewEvidenceService(suite.DB, zap.NewNop().Sugar(), suite.Config, nil)
	for _, item := range got.Items {
		if item.EvidenceID == nil {
			continue
		}
		result, err := evidenceService.VerifyByID(*item.EvidenceID)
		suite.Require().NoError(err)
		suite.True(result.IsValid, "item %d: %v", item.Index, result.Errors)
		suite.Require().NotNil(result.SignedAt)
		suite.True(summary.AcceptedAt.Equal(*result.SignedAt), "signed at %s, accepted at %s", result.SignedAt, summary.AcceptedAt)
		suite.Equal("submitter@example.com", result.Signer.Email)
	}
}

func (suite *EvidenceBatchIntegrationSuite) TestProcessChunkRetriesTransientFailuresThenFailsOnFinalAttempt() {
	suite.Require().NoError(suite.Migrator.Refresh())
	transientFailures := 2
	svc := suite.newBatchService(&fakeBatchEnqueuer{}, 3, &transientFailures)
	signer := userSigner("submitter@example.com")
	ctx := context.Background()

	summary, err := svc.Submit(ctx, batchPayloads(
		testBatchPayload{Title: "a"}, testBatchPayload{Title: "b", Transient: true}, testBatchPayload{Title: "c"},
	), signer)
	suite.Require().NoError(err)

	// A transient failure stops the chunk so the job is retried; earlier items are kept.
	suite.Require().Error(svc.ProcessChunk(ctx, summary.ID, 0, false))
	got, err := svc.Get(ctx, summary.ID, BatchSubmitter(signer))
	suite.Require().NoError(err)
	suite.Equal(1, got.Succeeded)
	suite.Equal(2, got.Pending)

	// On the final attempt the failing item fails, without its cause, and the rest go on.
	suite.Require().NoError(svc.ProcessChunk(ctx, summary.ID, 0, true))
	got, err = svc.Get(ctx, summary.ID, BatchSubmitter(signer))
	suite.Require().NoError(err)
	suite.Equal(BatchStatusCompleted, got.Status)
	suite.Equal(2, got.Succeeded)
	suite.Equal(1, got.Failed)
	suite.Equal(batchItemInternalError, got.Items[1].Error)
}

func (suite *EvidenceBatchIntegrationSuite) TestConcurrentDuplicateChunkJobsStoreEachItemOnce() {
	suite.Require().NoError(suite.Migrator.Refresh())
	svc := suite.newBatchService(&fakeBatchEnqueuer{}, 20, nil)
	signer := userSigner("submitter@example.com")
	ctx := context.Background()

	items := make([]testBatchPayload, 20)
	for i := range items {
		items[i] = testBatchPayload{Title: "dup"}
	}
	summary, err := svc.Submit(ctx, batchPayloads(items...), signer)
	suite.Require().NoError(err)

	var wg sync.WaitGroup
	errs := make([]error, 3)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = svc.ProcessChunk(ctx, summary.ID, 0, false)
		}(i)
	}
	wg.Wait()
	for _, err := range errs {
		suite.Require().NoError(err)
	}

	suite.EqualValues(20, suite.evidenceCount())
	got, err := svc.Get(ctx, summary.ID, BatchSubmitter(signer))
	suite.Require().NoError(err)
	suite.Equal(20, got.Succeeded)
}

func (suite *EvidenceBatchIntegrationSuite) TestProcessChunkOfDeletedBatchIsANoop() {
	suite.Require().NoError(suite.Migrator.Refresh())
	svc := suite.newBatchService(&fakeBatchEnqueuer{}, 2, nil)
	suite.NoError(svc.ProcessChunk(context.Background(), uuid.New(), 0, false))
}

func (suite *EvidenceBatchIntegrationSuite) TestUnreadableSignerFailsChunkOnFinalAttempt() {
	suite.Require().NoError(suite.Migrator.Refresh())
	svc := suite.newBatchService(&fakeBatchEnqueuer{}, 2, nil)
	signer := userSigner("submitter@example.com")
	ctx := context.Background()

	summary, err := svc.Submit(ctx, batchPayloads(testBatchPayload{Title: "a"}, testBatchPayload{Title: "b"}, testBatchPayload{Title: "c"}), signer)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.DB.Model(&relational.EvidenceBatch{}).
		Where("id = ?", summary.ID).Update("signer", datatypes.JSON(`"not a signer"`)).Error)

	suite.Require().Error(svc.ProcessChunk(ctx, summary.ID, 0, false), "retried while attempts remain")
	suite.Require().NoError(svc.ProcessChunk(ctx, summary.ID, 0, true))

	got, err := svc.Get(ctx, summary.ID, BatchSubmitter(signer))
	suite.Require().NoError(err)
	suite.Equal(2, got.Failed, "the chunk's items fail on the final attempt")
	suite.Equal(1, got.Pending, "other chunks are untouched")
	suite.Equal(batchItemInternalError, got.Items[0].Error)
	suite.Zero(suite.evidenceCount())
}

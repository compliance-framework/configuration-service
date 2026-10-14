package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/compliance-framework/api/sdk/types"
	"github.com/google/uuid"
)

func TestEvidenceCreateBatchSendsEvidenceList(t *testing.T) {
	batchID := uuid.New()
	client := artifactTestClient(t, func(r *http.Request) *http.Response {
		if r.Method != http.MethodPost || r.URL.Path != "/api/evidence/batches" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body struct {
			Evidence []types.Evidence `json:"evidence"`
		}
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("decode body %s: %v", raw, err)
		}
		if len(body.Evidence) != 2 || body.Evidence[1].Title != "second" {
			t.Fatalf("unexpected evidence: %s", raw)
		}
		return response(http.StatusAccepted, "application/json", `{"data":{"id":"`+batchID.String()+`","status":"pending","total":2,"pending":2,"items":[]}}`)
	})

	batch, err := client.Evidence.CreateBatch(context.Background(), []types.Evidence{{Title: "first"}, {Title: "second"}})
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}
	if batch.ID != batchID || batch.Status != types.EvidenceBatchPending || batch.Total != 2 {
		t.Fatalf("unexpected batch: %+v", batch)
	}
}

func TestEvidenceCreateBatchReportsStatus(t *testing.T) {
	client := artifactTestClient(t, func(*http.Request) *http.Response {
		return response(http.StatusServiceUnavailable, "application/json", `{"errors":{"body":"evidence batches are not available"}}`)
	})

	_, err := client.Evidence.CreateBatch(context.Background(), []types.Evidence{{Title: "e"}})
	var statusErr *APIStatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected a 503 APIStatusError, got %v", err)
	}
}

func TestEvidenceWaitForBatchPollsUntilCompleted(t *testing.T) {
	batchID := uuid.New()
	var calls atomic.Int32
	client := artifactTestClient(t, func(r *http.Request) *http.Response {
		if r.Method != http.MethodGet || r.URL.Path != "/api/evidence/batches/"+batchID.String() {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if calls.Add(1) < 3 {
			return response(http.StatusOK, "application/json", `{"data":{"id":"`+batchID.String()+`","status":"processing","total":1,"pending":1,"items":[]}}`)
		}
		return response(http.StatusOK, "application/json", `{"data":{"id":"`+batchID.String()+`","status":"completed","total":1,"succeeded":1,"items":[{"index":0,"status":"succeeded","evidence-id":"`+batchID.String()+`"}]}}`)
	})

	batch, err := client.Evidence.WaitForBatch(context.Background(), batchID, time.Millisecond)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if batch.Status != types.EvidenceBatchCompleted || batch.Succeeded != 1 || calls.Load() != 3 {
		t.Fatalf("unexpected batch %+v after %d calls", batch, calls.Load())
	}
	if len(batch.Items) != 1 || batch.Items[0].EvidenceID == nil || *batch.Items[0].EvidenceID != batchID {
		t.Fatalf("unexpected items: %+v", batch.Items)
	}
}

func TestEvidenceWaitForBatchStopsWithContext(t *testing.T) {
	batchID := uuid.New()
	client := artifactTestClient(t, func(*http.Request) *http.Response {
		return response(http.StatusOK, "application/json", `{"data":{"id":"`+batchID.String()+`","status":"pending","total":1,"pending":1,"items":[]}}`)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	batch, err := client.Evidence.WaitForBatch(ctx, batchID, 5*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected the context's error, got %v", err)
	}
	if batch == nil || batch.Status != types.EvidenceBatchPending {
		t.Fatalf("expected the last batch read, got %+v", batch)
	}

	if _, err := client.Evidence.WaitForBatch(context.Background(), batchID, 0); err == nil {
		t.Fatal("expected an error for a zero interval")
	}
}

func TestEvidenceGetBatchReportsStatus(t *testing.T) {
	client := artifactTestClient(t, func(*http.Request) *http.Response {
		return response(http.StatusNotFound, "application/json", `{"errors":{"body":"resource not found"}}`)
	})

	_, err := client.Evidence.GetBatch(context.Background(), uuid.New())
	var statusErr *APIStatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected a 404 APIStatusError, got %v", err)
	}
}

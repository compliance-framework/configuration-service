package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/compliance-framework/api/sdk/types"
	"github.com/google/uuid"
)

type evidenceClient struct {
	client *Client
}

func (r *evidenceClient) Create(ctx context.Context, evidence ...types.Evidence) error {
	for _, evid := range evidence {
		response, err := r.client.doJSONRequest(ctx, http.MethodPost, "/api/evidence", evid)
		if err != nil {
			return err
		}
		closeResponseBody(response, r.client.config.Logger)

		if response.StatusCode != http.StatusCreated {
			return fmt.Errorf("unexpected api response status code: %d", response.StatusCode)
		}
	}

	return nil
}

// CreateBatch submits evidence to be stored asynchronously (POST /api/evidence/batches)
// and returns the accepted, pending batch. Poll it with GetBatch or WaitForBatch: each
// evidence succeeds or fails on its own. A batch is limited in size by the API
// (CCF_EVIDENCE_BATCH_MAX_ITEMS); an API that predates batches, or runs without its
// worker service, answers with an *APIStatusError (404, 405 or 503).
func (r *evidenceClient) CreateBatch(ctx context.Context, evidence []types.Evidence) (*types.EvidenceBatch, error) {
	if evidence == nil {
		evidence = []types.Evidence{}
	}
	response, err := r.client.doJSONRequest(ctx, http.MethodPost, "/api/evidence/batches", map[string]any{"evidence": evidence})
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(response, r.client.config.Logger)

	if response.StatusCode != http.StatusAccepted {
		return nil, newAPIStatusError(response)
	}
	return decodeEvidenceBatch(response)
}

// GetBatch returns a batch this client submitted, with each evidence's outcome.
func (r *evidenceClient) GetBatch(ctx context.Context, id uuid.UUID) (*types.EvidenceBatch, error) {
	response, err := r.client.doRequest(ctx, http.MethodGet, "/api/evidence/batches/"+id.String(), nil)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(response, r.client.config.Logger)

	if response.StatusCode != http.StatusOK {
		return nil, newAPIStatusError(response)
	}
	return decodeEvidenceBatch(response)
}

// WaitForBatch polls GetBatch every interval until the batch is completed or ctx ends.
func (r *evidenceClient) WaitForBatch(ctx context.Context, id uuid.UUID, interval time.Duration) (*types.EvidenceBatch, error) {
	if interval <= 0 {
		return nil, errors.New("sdk: WaitForBatch needs a positive interval")
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		batch, err := r.GetBatch(ctx, id)
		if err != nil {
			return nil, err
		}
		if batch.Status == types.EvidenceBatchCompleted {
			return batch, nil
		}
		select {
		case <-ctx.Done():
			return batch, ctx.Err()
		case <-ticker.C:
		}
	}
}

func decodeEvidenceBatch(response *http.Response) (*types.EvidenceBatch, error) {
	var body struct {
		Data *types.EvidenceBatch `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("sdk: decode evidence batch response: %w", err)
	}
	if body.Data == nil {
		return nil, errors.New("sdk: evidence batch response missing data")
	}
	return body.Data, nil
}

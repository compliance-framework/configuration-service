//go:build integration

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/compliance-framework/api/internal/api"
	"github.com/compliance-framework/api/internal/config"
	"github.com/compliance-framework/api/internal/service/relational"
	evidencesvc "github.com/compliance-framework/api/internal/service/relational/evidence"
	"github.com/compliance-framework/api/internal/service/worker"
	"github.com/compliance-framework/api/internal/tests"
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

func TestEvidenceBatchApi(t *testing.T) {
	suite.Run(t, new(EvidenceBatchApiIntegrationSuite))
}

type EvidenceBatchApiIntegrationSuite struct {
	tests.IntegrationTestSuite
}

// inlineBatchQueue records the chunks a batch enqueues and runs them on demand, standing
// in for the River worker.
type inlineBatchQueue struct {
	mu        sync.Mutex
	processor worker.EvidenceBatchProcessor
	chunks    map[uuid.UUID]int
}

func (q *inlineBatchQueue) EvidenceBatchesAvailable() error { return nil }

func (q *inlineBatchQueue) EnqueueEvidenceBatchChunks(_ context.Context, batchID uuid.UUID, chunks int) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.chunks == nil {
		q.chunks = map[uuid.UUID]int{}
	}
	q.chunks[batchID] = chunks
	return nil
}

func (q *inlineBatchQueue) SetEvidenceBatchProcessor(processor worker.EvidenceBatchProcessor) {
	q.processor = processor
}

func (q *inlineBatchQueue) runAll(ctx context.Context) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	for batchID, chunks := range q.chunks {
		for chunk := 0; chunk < chunks; chunk++ {
			if err := q.processor.ProcessChunk(ctx, batchID, chunk, true); err != nil {
				return err
			}
		}
		delete(q.chunks, batchID)
	}
	return nil
}

func (suite *EvidenceBatchApiIntegrationSuite) setupServer(queue EvidenceBatchQueue) *api.Server {
	logger := zap.NewNop().Sugar()
	metrics := api.NewMetricsHandler(context.Background(), logger)
	server := api.NewServer(context.Background(), logger, suite.Config, metrics)
	services := &APIServices{
		EvidenceService: evidencesvc.NewEvidenceService(suite.DB, logger, suite.Config, nil),
	}
	if queue != nil {
		services.EvidenceBatchQueue = queue
	}
	RegisterHandlers(server, logger, suite.DB, suite.Config, services)
	return server
}

func (suite *EvidenceBatchApiIntegrationSuite) agentToken(name string) string {
	agent, err := suite.CreateAgent(name)
	suite.Require().NoError(err)
	key, _, err := suite.CreateAgentKey(agent, name+"-key")
	suite.Require().NoError(err)
	token, err := suite.GetAgentToken(agent, key)
	suite.Require().NoError(err)
	return *token
}

func (suite *EvidenceBatchApiIntegrationSuite) userToken() string {
	token, err := suite.GetAuthToken()
	suite.Require().NoError(err)
	return *token
}

func (suite *EvidenceBatchApiIntegrationSuite) do(server *api.Server, method, path, token string, body any) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	switch b := body.(type) {
	case nil:
		reader = bytes.NewReader(nil)
	case []byte:
		reader = bytes.NewReader(b)
	default:
		raw, err := json.Marshal(b)
		suite.Require().NoError(err)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if token != "" {
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	server.E().ServeHTTP(rec, req)
	return rec
}

func validBatchEvidence(title string) EvidenceCreateRequest {
	return EvidenceCreateRequest{
		UUID:   uuid.New(),
		Title:  title,
		Start:  time.Now().Add(-time.Hour),
		End:    time.Now().Add(-time.Minute),
		Labels: map[string]string{"source": "batch"},
		Status: oscalTypes_1_1_3.ObjectiveStatus{State: "satisfied"},
	}
}

func decodeBatchResponse(suite *EvidenceBatchApiIntegrationSuite, rec *httptest.ResponseRecorder) EvidenceBatchResponse {
	var resp GenericDataResponse[EvidenceBatchResponse]
	suite.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &resp), rec.Body.String())
	return resp.Data
}

func (suite *EvidenceBatchApiIntegrationSuite) TestBatchStoresEvidenceAsItsSubmitter() {
	suite.Require().NoError(suite.Migrator.Refresh())
	suite.Config.EvidenceBatch = &config.EvidenceBatchConfig{MaxItems: 10, ChunkSize: 2}
	defer func() { suite.Config.EvidenceBatch = nil }()
	queue := &inlineBatchQueue{}
	server := suite.setupServer(queue)
	token := suite.agentToken("batch-agent")

	unknownSubject := validBatchEvidence("unknown subject")
	unknownSubjectID := uuid.New()
	unknownSubject.Subjects = []EvidenceSubject{{SubjectUUID: &unknownSubjectID}}
	missingArtifact := validBatchEvidence("missing artifact")
	missingArtifact.PolicyArtifacts = &EvidencePolicyArtifacts{BundleDigest: "sha256:" + fmt.Sprintf("%064d", 1), InputDigest: "sha256:" + fmt.Sprintf("%064d", 2)}

	rec := suite.do(server, http.MethodPost, "/api/evidence/batches", token, EvidenceBatchCreateRequest{Evidence: []EvidenceCreateRequest{
		validBatchEvidence("first"), unknownSubject, validBatchEvidence("third"), missingArtifact,
	}})
	suite.Require().Equal(http.StatusAccepted, rec.Code, rec.Body.String())
	accepted := decodeBatchResponse(suite, rec)
	suite.Equal("pending", accepted.Status)
	suite.Equal(4, accepted.Total)
	suite.Equal(4, accepted.Pending)
	suite.Equal("/api/evidence/batches/"+accepted.ID.String(), rec.Header().Get(echo.HeaderLocation))
	suite.Equal(2, queue.chunks[accepted.ID], "4 items in chunks of 2")

	suite.Require().NoError(queue.runAll(context.Background()))

	rec = suite.do(server, http.MethodGet, "/api/evidence/batches/"+accepted.ID.String(), token, nil)
	suite.Require().Equal(http.StatusOK, rec.Code, rec.Body.String())
	batch := decodeBatchResponse(suite, rec)
	suite.Equal("completed", batch.Status)
	suite.Equal(2, batch.Succeeded)
	suite.Equal(2, batch.Failed)
	suite.Require().Len(batch.Items, 4)
	suite.Equal("succeeded", batch.Items[0].Status)
	suite.Equal("failed", batch.Items[1].Status)
	suite.Contains(batch.Items[1].Error, "unknown subject")
	suite.Equal("succeeded", batch.Items[2].Status)
	suite.Equal("failed", batch.Items[3].Status)
	suite.Contains(batch.Items[3].Error, "is not a stored artifact")

	// The stored evidence is signed as the agent, like POST /evidence, and readable.
	suite.Require().NotNil(batch.Items[0].EvidenceID)
	var stored relational.Evidence
	suite.Require().NoError(suite.DB.Preload("Labels").First(&stored, "id = ?", *batch.Items[0].EvidenceID).Error)
	suite.Equal("first", stored.Title)
	suite.Require().NotNil(stored.Signature)
	suite.Equal(relational.AgentAuthMethodServiceAccount, stored.Signature.Data().Claims.AuthMethod)
	suite.True(accepted.AcceptedAt.Equal(stored.Signature.Data().SignedAt))
	rec = suite.do(server, http.MethodGet, "/api/evidence/"+batch.Items[0].EvidenceID.String(), token, nil)
	suite.Equal(http.StatusOK, rec.Code, rec.Body.String())

	suite.Equal(batch.SubmittedBy, accepted.SubmittedBy)
	suite.True(strings.HasPrefix(batch.SubmittedBy, "agent:"), batch.SubmittedBy)

	// Another agent holds no evidence:inspect, so the batch doesn't exist for it.
	rec = suite.do(server, http.MethodGet, "/api/evidence/batches/"+accepted.ID.String(), suite.agentToken("other-agent"), nil)
	suite.Equal(http.StatusNotFound, rec.Code)
	// An admin (the test user, under the builtin driver without SSO) holds evidence:inspect
	// and can troubleshoot the agent's batch.
	rec = suite.do(server, http.MethodGet, "/api/evidence/batches/"+accepted.ID.String(), suite.userToken(), nil)
	suite.Require().Equal(http.StatusOK, rec.Code, rec.Body.String())
	inspected := decodeBatchResponse(suite, rec)
	suite.Equal(batch.SubmittedBy, inspected.SubmittedBy)
	suite.Equal(batch.Items, inspected.Items)
	// Inspecting a batch that doesn't exist is still a 404.
	rec = suite.do(server, http.MethodGet, "/api/evidence/batches/"+uuid.NewString(), suite.userToken(), nil)
	suite.Equal(http.StatusNotFound, rec.Code)
	rec = suite.do(server, http.MethodGet, "/api/evidence/batches/not-a-uuid", token, nil)
	suite.Equal(http.StatusBadRequest, rec.Code)
}

func (suite *EvidenceBatchApiIntegrationSuite) TestBatchRejectsInvalidRequestsUpFront() {
	suite.Require().NoError(suite.Migrator.Refresh())
	suite.Config.EvidenceBatch = &config.EvidenceBatchConfig{MaxItems: 4, ChunkSize: 2}
	defer func() { suite.Config.EvidenceBatch = nil }()
	queue := &inlineBatchQueue{}
	server := suite.setupServer(queue)
	token := suite.userToken()

	rec := suite.do(server, http.MethodPost, "/api/evidence/batches", token, EvidenceBatchCreateRequest{})
	suite.Equal(http.StatusBadRequest, rec.Code, "an empty batch")

	tooMany := make([]EvidenceCreateRequest, 5)
	for i := range tooMany {
		tooMany[i] = validBatchEvidence("e")
	}
	rec = suite.do(server, http.MethodPost, "/api/evidence/batches", token, EvidenceBatchCreateRequest{Evidence: tooMany})
	suite.Equal(http.StatusBadRequest, rec.Code, "more than CCF_EVIDENCE_BATCH_MAX_ITEMS")
	suite.Contains(rec.Body.String(), "at most 4")

	future := validBatchEvidence("future")
	future.End = time.Now().Add(time.Hour)
	reserved := validBatchEvidence("reserved prop")
	reserved.Props = []oscalTypes_1_1_3.Property{{Name: "_policy_bundle_digest", Value: "sha256:x"}}
	valid, err := json.Marshal(validBatchEvidence("valid"))
	suite.Require().NoError(err)
	futureRaw, err := json.Marshal(future)
	suite.Require().NoError(err)
	reservedRaw, err := json.Marshal(reserved)
	suite.Require().NoError(err)
	body := []byte(fmt.Sprintf(`{"evidence":[%s,%s,%s,null]}`, valid, futureRaw, reservedRaw))

	rec = suite.do(server, http.MethodPost, "/api/evidence/batches", token, body)
	suite.Require().Equal(http.StatusBadRequest, rec.Code, rec.Body.String())
	var invalid api.Error
	suite.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &invalid))
	suite.NotContains(invalid.Errors, "evidence[0]")
	suite.Contains(invalid.Errors["evidence[1]"], "must not be in the future")
	suite.Contains(invalid.Errors["evidence[2]"], "_policy_bundle_digest")
	suite.Contains(invalid.Errors["evidence[3]"], "must be an object")

	var batches int64
	suite.Require().NoError(suite.DB.Model(&relational.EvidenceBatch{}).Count(&batches).Error)
	suite.Zero(batches, "a rejected request stores nothing")
	suite.Empty(queue.chunks)
}

func (suite *EvidenceBatchApiIntegrationSuite) TestBatchNeedsAnIdentifiedCaller() {
	suite.Require().NoError(suite.Migrator.Refresh())
	// Even where single evidence ingest is public, batches are not.
	suite.Config.StrictDisablePublicAgentEndpoints = false
	server := suite.setupServer(&inlineBatchQueue{})

	rec := suite.do(server, http.MethodPost, "/api/evidence/batches", "", EvidenceBatchCreateRequest{Evidence: []EvidenceCreateRequest{validBatchEvidence("e")}})
	suite.Equal(http.StatusUnauthorized, rec.Code)
	rec = suite.do(server, http.MethodGet, "/api/evidence/batches/"+uuid.NewString(), "", nil)
	suite.Equal(http.StatusUnauthorized, rec.Code)
}

func (suite *EvidenceBatchApiIntegrationSuite) TestBatchUnavailableWithoutWorker() {
	suite.Require().NoError(suite.Migrator.Refresh())
	server := suite.setupServer(nil)
	token := suite.userToken()

	rec := suite.do(server, http.MethodPost, "/api/evidence/batches", token, EvidenceBatchCreateRequest{Evidence: []EvidenceCreateRequest{validBatchEvidence("e")}})
	suite.Equal(http.StatusServiceUnavailable, rec.Code)
	rec = suite.do(server, http.MethodGet, "/api/evidence/batches/"+uuid.NewString(), token, nil)
	suite.Equal(http.StatusServiceUnavailable, rec.Code)
}

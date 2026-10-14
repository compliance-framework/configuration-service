//go:build integration

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/compliance-framework/api/internal/api"
	evidencesvc "github.com/compliance-framework/api/internal/service/relational/evidence"
	"github.com/compliance-framework/api/internal/service/relational/templates"
	"github.com/compliance-framework/api/internal/tests"
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// evidenceCreateStatementBudget is how many SQL statements POST /api/evidence may run for
// the agent-shaped evidence below: 2 for agent auth, 1 subject template lookup, 5
// multi-row pre-creates (7 with nested steps, implemented components and subject
// selections), 2 inserts (subject references, evidence), 5 join-table inserts, 17 reads
// of the stored evidence to sign it, and the signature update. Raise it only on purpose.
const evidenceCreateStatementBudget = 36

func TestEvidenceQueryBudget(t *testing.T) {
	suite.Run(t, new(EvidenceQueryBudgetSuite))
}

type EvidenceQueryBudgetSuite struct {
	tests.IntegrationTestSuite
}

// statementCounter records every SQL statement GORM runs on a database.
type statementCounter struct {
	mu         sync.Mutex
	statements []string
}

func (c *statementCounter) record(db *gorm.DB) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.statements = append(c.statements, db.Statement.SQL.String())
}

func (c *statementCounter) reset() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := c.statements
	c.statements = nil
	return out
}

// register records db's statements until t ends. Sessions share their parent's
// callbacks, so they are removed again rather than left on the suite's database.
func (c *statementCounter) register(t testing.TB, db *gorm.DB) {
	callbacks := db.Callback()
	require.NoError(t, callbacks.Create().After("gorm:create").Register("budget:create", c.record))
	require.NoError(t, callbacks.Query().After("gorm:query").Register("budget:query", c.record))
	require.NoError(t, callbacks.Update().After("gorm:update").Register("budget:update", c.record))
	require.NoError(t, callbacks.Delete().After("gorm:delete").Register("budget:delete", c.record))
	require.NoError(t, callbacks.Row().After("gorm:row").Register("budget:row", c.record))
	require.NoError(t, callbacks.Raw().After("gorm:raw").Register("budget:raw", c.record))
	t.Cleanup(func() {
		_ = callbacks.Create().Remove("budget:create")
		_ = callbacks.Query().Remove("budget:query")
		_ = callbacks.Update().Remove("budget:update")
		_ = callbacks.Delete().Remove("budget:delete")
		_ = callbacks.Row().Remove("budget:row")
		_ = callbacks.Raw().Remove("budget:raw")
	})
}

func (suite *EvidenceQueryBudgetSuite) TestCreateStaysWithinStatementBudget() {
	suite.Require().NoError(suite.Migrator.Refresh())
	db := suite.DB.Session(&gorm.Session{NewDB: true})
	counter := &statementCounter{}
	counter.register(suite.T(), db)

	logger := zap.NewNop().Sugar()
	server := api.NewServer(context.Background(), logger, suite.Config, api.NewMetricsHandler(context.Background(), logger))
	evidenceService := evidencesvc.NewEvidenceService(db, logger, suite.Config, nil,
		evidencesvc.WithComponentDefinitionResolver(templates.NewSubjectTemplateService(db)))
	RegisterHandlers(server, logger, db, suite.Config, &APIServices{EvidenceService: evidenceService})

	agent, err := suite.CreateAgent("budget-agent")
	suite.Require().NoError(err)
	key, _, err := suite.CreateAgentKey(agent, "budget-key")
	suite.Require().NoError(err)
	token, err := suite.GetAgentToken(agent, key)
	suite.Require().NoError(err)

	instance := "web-server/ec2/i-000001"
	body, err := json.Marshal(map[string]any{
		"uuid":   uuid.NewSHA1(uuid.NameSpaceURL, []byte(instance)),
		"title":  "IMDSv2 enforced",
		"start":  time.Now().Add(-time.Hour),
		"end":    time.Now().Add(-time.Minute),
		"labels": map[string]string{"_plugin": "budget", "_policy": "p", "provider": "aws", "service": "ec2", "region": "eu-west-1", "account": "a"},
		"props":  []map[string]string{{"name": "imds", "value": "v2"}},
		"activities": []map[string]any{{"uuid": uuid.New(), "title": "Evaluate", "steps": []map[string]any{
			{"uuid": uuid.New(), "title": "Describe"}, {"uuid": uuid.New(), "title": "Evaluate"},
		}}},
		"components":      []map[string]any{{"identifier": "components/common/aws-ec2", "type": "service", "title": "EC2"}},
		"inventory-items": []map[string]any{{"identifier": instance, "type": "web-server", "implemented-components": []map[string]string{{"identifier": "components/common/aws-ec2"}}}},
		"subjects":        []map[string]any{{"identifier": instance, "type": "inventory-item"}},
		"status":          oscalTypes_1_1_3.ObjectiveStatus{State: "satisfied"},
	})
	suite.Require().NoError(err)

	// Twice: the second time every related row already exists.
	for round := 0; round < 2; round++ {
		counter.reset()
		req := httptest.NewRequest(http.MethodPost, "/api/evidence", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+*token)
		rec := httptest.NewRecorder()
		server.E().ServeHTTP(rec, req)
		suite.Require().Equal(http.StatusCreated, rec.Code, rec.Body.String())
		statements := counter.reset()
		suite.LessOrEqual(len(statements), evidenceCreateStatementBudget, "round %d ran:\n%s", round, strings.Join(statements, "\n"))

		// The response still carries every linked entity, and the signature verifies.
		var created GenericDataResponse[CreatedEvidenceResponse]
		suite.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &created))
		suite.Len(created.Data.Labels, 6)
		suite.Len(created.Data.Components, 1)
		suite.Len(created.Data.InventoryItems, 1)
		suite.Len(created.Data.Activities, 1)
		suite.Len(created.Data.Subjects, 1)
		suite.Require().NotNil(created.Data.Signature)

		result, err := evidenceService.VerifyByID(*created.Data.ID)
		suite.Require().NoError(err)
		suite.True(result.IsValid, "%v", result.Errors)
	}
}

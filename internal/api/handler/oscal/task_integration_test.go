//go:build integration

package oscal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/compliance-framework/configuration-service/internal/api"
	"github.com/compliance-framework/configuration-service/internal/api/handler"
	"github.com/compliance-framework/configuration-service/internal/tests"
	oscalTypes_1_1_3 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

func TestTaskApi(t *testing.T) {
	suite.Run(t, new(TaskApiIntegrationSuite))
}

type TaskApiIntegrationSuite struct {
	tests.IntegrationTestSuite
	server *api.Server
	logger *zap.SugaredLogger
}

func (suite *TaskApiIntegrationSuite) SetupSuite() {
	suite.IntegrationTestSuite.SetupSuite()

	logger, _ := zap.NewDevelopment()
	suite.logger = logger.Sugar()
	suite.server = api.NewServer(context.Background(), suite.logger, suite.Config)
	RegisterHandlers(suite.server, suite.logger, suite.DB, suite.Config)
}

func (suite *TaskApiIntegrationSuite) SetupTest() {
	err := suite.Migrator.Refresh()
	suite.Require().NoError(err)
}

// Helper method to create a test request with Bearer token authentication
func (suite *TaskApiIntegrationSuite) createRequest(method, path string, body any) (*httptest.ResponseRecorder, *http.Request) {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		suite.Require().NoError(err, "Failed to marshal request body")
	}

	token, err := suite.GetAuthToken()
	suite.Require().NoError(err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderAuthorization, fmt.Sprintf("Bearer %s", *token))

	return rec, req
}

// Helper method to create a test assessment plan (prerequisite for task tests)
func (suite *TaskApiIntegrationSuite) createTestAssessmentPlan() uuid.UUID {
	planID := uuid.New()
	testPlan := &oscalTypes_1_1_3.AssessmentPlan{
		UUID: planID.String(),
		Metadata: oscalTypes_1_1_3.Metadata{
			Title:   "Test Assessment Plan for Tasks",
			Version: "1.0.0",
		},
		ImportSsp: oscalTypes_1_1_3.ImportSsp{
			Href: "test-ssp-reference",
		},
	}

	rec, req := suite.createRequest(http.MethodPost, "/api/oscal/assessment-plans", testPlan)
	suite.server.E().ServeHTTP(rec, req)
	suite.Require().Equal(http.StatusCreated, rec.Code, "Failed to create test assessment plan")

	return planID
}

// Helper method to create test task data
func (suite *TaskApiIntegrationSuite) createTestTaskData() *oscalTypes_1_1_3.Task {
	taskID := uuid.New()
	return &oscalTypes_1_1_3.Task{
		UUID:        taskID.String(),
		Title:       "Test Task",
		Description: "Test task description for integration testing",
		Type:        "milestone",
	}
}

func (suite *TaskApiIntegrationSuite) TestCreateTask() {
	// Create test assessment plan first
	planID := suite.createTestAssessmentPlan()
	testTask := suite.createTestTaskData()

	// Create task
	rec, req := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-plans/%s/tasks", planID), testTask)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusCreated, rec.Code)

	// Verify response
	var response handler.GenericDataResponse[*oscalTypes_1_1_3.Task]
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Equal(testTask.UUID, response.Data.UUID)
	suite.Equal(testTask.Title, response.Data.Title)
	suite.Equal(testTask.Description, response.Data.Description)
	suite.Equal(testTask.Type, response.Data.Type)
}

func (suite *TaskApiIntegrationSuite) TestGetTasks() {
	// Create test assessment plan first
	planID := suite.createTestAssessmentPlan()
	testTask := suite.createTestTaskData()

	// Create task first
	createRec, createReq := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-plans/%s/tasks", planID), testTask)
	suite.server.E().ServeHTTP(createRec, createReq)
	suite.Require().Equal(http.StatusCreated, createRec.Code)

	// Get tasks
	getRec, getReq := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-plans/%s/tasks", planID), nil)
	suite.server.E().ServeHTTP(getRec, getReq)
	suite.Equal(http.StatusOK, getRec.Code)

	// Verify response
	var response handler.GenericDataResponse[[]*oscalTypes_1_1_3.Task]
	err := json.Unmarshal(getRec.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Require().Len(response.Data, 1)
	suite.Equal(testTask.UUID, response.Data[0].UUID)
	suite.Equal(testTask.Title, response.Data[0].Title)
	suite.Equal(testTask.Type, response.Data[0].Type)
}

func (suite *TaskApiIntegrationSuite) TestUpdateTask() {
	// Create test assessment plan first
	planID := suite.createTestAssessmentPlan()
	testTask := suite.createTestTaskData()

	// Create task first
	createRec, createReq := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-plans/%s/tasks", planID), testTask)
	suite.server.E().ServeHTTP(createRec, createReq)
	suite.Require().Equal(http.StatusCreated, createRec.Code)

	// Update task
	testTask.Title = "Updated Test Task"
	testTask.Description = "Updated test task description"
	testTask.Type = "action"

	updateRec, updateReq := suite.createRequest(http.MethodPut, fmt.Sprintf("/api/oscal/assessment-plans/%s/tasks/%s", planID, testTask.UUID), testTask)
	suite.server.E().ServeHTTP(updateRec, updateReq)
	suite.Equal(http.StatusOK, updateRec.Code)

	// Verify response
	var response handler.GenericDataResponse[*oscalTypes_1_1_3.Task]
	err := json.Unmarshal(updateRec.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Equal(testTask.UUID, response.Data.UUID)
	suite.Equal("Updated Test Task", response.Data.Title)
	suite.Equal("Updated test task description", response.Data.Description)
	suite.Equal("action", response.Data.Type)
}

func (suite *TaskApiIntegrationSuite) TestDeleteTask() {
	// Create test assessment plan first
	planID := suite.createTestAssessmentPlan()
	testTask := suite.createTestTaskData()

	// Create task first
	createRec, createReq := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-plans/%s/tasks", planID), testTask)
	suite.server.E().ServeHTTP(createRec, createReq)
	suite.Require().Equal(http.StatusCreated, createRec.Code)

	// Delete task
	deleteRec, deleteReq := suite.createRequest(http.MethodDelete, fmt.Sprintf("/api/oscal/assessment-plans/%s/tasks/%s", planID, testTask.UUID), nil)
	suite.server.E().ServeHTTP(deleteRec, deleteReq)
	suite.Equal(http.StatusNoContent, deleteRec.Code)

	// Verify task is deleted by trying to get tasks
	getRec, getReq := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-plans/%s/tasks", planID), nil)
	suite.server.E().ServeHTTP(getRec, getReq)
	suite.Equal(http.StatusOK, getRec.Code)

	var response handler.GenericDataResponse[[]*oscalTypes_1_1_3.Task]
	err := json.Unmarshal(getRec.Body.Bytes(), &response)
	suite.Require().NoError(err)
	suite.Len(response.Data, 0)
}

func (suite *TaskApiIntegrationSuite) TestTaskValidationErrors() {
	// Create test assessment plan first
	planID := suite.createTestAssessmentPlan()

	// Test with invalid task (missing required fields)
	invalidTask := &oscalTypes_1_1_3.Task{
		UUID: "invalid-uuid",
		// Missing Title and Type which are required
	}

	rec, req := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-plans/%s/tasks", planID), invalidTask)
	suite.server.E().ServeHTTP(rec, req)
	suite.Equal(http.StatusBadRequest, rec.Code)
}

func (suite *TaskApiIntegrationSuite) TestTaskNotFound() {
	// Create test assessment plan first
	planID := suite.createTestAssessmentPlan()
	nonExistentTaskID := uuid.New()

	// Try to update non-existent task
	testTask := suite.createTestTaskData()
	updateRec, updateReq := suite.createRequest(http.MethodPut, fmt.Sprintf("/api/oscal/assessment-plans/%s/tasks/%s", planID, nonExistentTaskID), testTask)
	suite.server.E().ServeHTTP(updateRec, updateReq)
	suite.Equal(http.StatusNotFound, updateRec.Code)

	// Try to delete non-existent task
	deleteRec, deleteReq := suite.createRequest(http.MethodDelete, fmt.Sprintf("/api/oscal/assessment-plans/%s/tasks/%s", planID, nonExistentTaskID), nil)
	suite.server.E().ServeHTTP(deleteRec, deleteReq)
	suite.Equal(http.StatusNotFound, deleteRec.Code)
}

func (suite *TaskApiIntegrationSuite) TestAssessmentPlanNotFound() {
	nonExistentPlanID := uuid.New()
	testTask := suite.createTestTaskData()

	// Try to create task for non-existent assessment plan
	createRec, createReq := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-plans/%s/tasks", nonExistentPlanID), testTask)
	suite.server.E().ServeHTTP(createRec, createReq)
	suite.Equal(http.StatusNotFound, createRec.Code)

	// Try to get tasks for non-existent assessment plan
	getRec, getReq := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-plans/%s/tasks", nonExistentPlanID), nil)
	suite.server.E().ServeHTTP(getRec, getReq)
	suite.Equal(http.StatusNotFound, getRec.Code)
}

func (suite *TaskApiIntegrationSuite) TestTaskInvalidUUIDs() {
	// Create test assessment plan first
	planID := suite.createTestAssessmentPlan()
	testTask := suite.createTestTaskData()

	// Test with invalid assessment plan UUID
	invalidRec, invalidReq := suite.createRequest(http.MethodPost, "/api/oscal/assessment-plans/invalid-uuid/tasks", testTask)
	suite.server.E().ServeHTTP(invalidRec, invalidReq)
	suite.Equal(http.StatusBadRequest, invalidRec.Code)

	// Create a valid task first
	createRec, createReq := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-plans/%s/tasks", planID), testTask)
	suite.server.E().ServeHTTP(createRec, createReq)
	suite.Require().Equal(http.StatusCreated, createRec.Code)

	// Test with invalid task UUID for update
	updateRec, updateReq := suite.createRequest(http.MethodPut, fmt.Sprintf("/api/oscal/assessment-plans/%s/tasks/invalid-uuid", planID), testTask)
	suite.server.E().ServeHTTP(updateRec, updateReq)
	suite.Equal(http.StatusBadRequest, updateRec.Code)

	// Test with invalid task UUID for delete
	deleteRec, deleteReq := suite.createRequest(http.MethodDelete, fmt.Sprintf("/api/oscal/assessment-plans/%s/tasks/invalid-uuid", planID), nil)
	suite.server.E().ServeHTTP(deleteRec, deleteReq)
	suite.Equal(http.StatusBadRequest, deleteRec.Code)
}

func (suite *TaskApiIntegrationSuite) TestTaskDependencyRemoval() {
	// Create test assessment plan first
	planID := suite.createTestAssessmentPlan()

	// Create two tasks - one will depend on the other
	task1ID := uuid.New()
	task1 := &oscalTypes_1_1_3.Task{
		UUID:        task1ID.String(),
		Title:       "Task 1",
		Description: "First task for dependency testing",
		Type:        "milestone",
	}

	task2ID := uuid.New()
	task2 := &oscalTypes_1_1_3.Task{
		UUID:        task2ID.String(),
		Title:       "Task 2",
		Description: "Second task that will depend on Task 1",
		Type:        "action",
	}

	// Create both tasks
	createRec1, createReq1 := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-plans/%s/tasks", planID), task1)
	suite.server.E().ServeHTTP(createRec1, createReq1)
	suite.Require().Equal(http.StatusCreated, createRec1.Code)

	createRec2, createReq2 := suite.createRequest(http.MethodPost, fmt.Sprintf("/api/oscal/assessment-plans/%s/tasks", planID), task2)
	suite.server.E().ServeHTTP(createRec2, createReq2)
	suite.Require().Equal(http.StatusCreated, createRec2.Code)

	// Add dependency: task2 depends on task1
	dependencies := []oscalTypes_1_1_3.TaskDependency{
		{
			TaskUuid: task1ID.String(),
			Remarks:  "Task 2 depends on Task 1 completion",
		},
	}
	task2.Dependencies = &dependencies

	// Update task2 with dependency
	updateRec, updateReq := suite.createRequest(http.MethodPut, fmt.Sprintf("/api/oscal/assessment-plans/%s/tasks/%s", planID, task2ID), task2)
	suite.server.E().ServeHTTP(updateRec, updateReq)
	suite.Equal(http.StatusOK, updateRec.Code)

	// Verify dependency was added
	var updateResponse handler.GenericDataResponse[*oscalTypes_1_1_3.Task]
	err := json.Unmarshal(updateRec.Body.Bytes(), &updateResponse)
	suite.Require().NoError(err)
	suite.Require().NotNil(updateResponse.Data.Dependencies)
	suite.Require().Len(*updateResponse.Data.Dependencies, 1)
	suite.Equal(task1ID.String(), (*updateResponse.Data.Dependencies)[0].TaskUuid)
	suite.Equal("Task 2 depends on Task 1 completion", (*updateResponse.Data.Dependencies)[0].Remarks)

	// Now remove the dependency by updating task2 with empty dependencies
	task2.Dependencies = &[]oscalTypes_1_1_3.TaskDependency{} // Empty slice

	removeRec, removeReq := suite.createRequest(http.MethodPut, fmt.Sprintf("/api/oscal/assessment-plans/%s/tasks/%s", planID, task2ID), task2)
	suite.server.E().ServeHTTP(removeRec, removeReq)
	suite.Equal(http.StatusOK, removeRec.Code)

	// Verify dependency was removed
	var removeResponse handler.GenericDataResponse[*oscalTypes_1_1_3.Task]
	err = json.Unmarshal(removeRec.Body.Bytes(), &removeResponse)
	suite.Require().NoError(err)

	// Dependencies should be empty or nil
	if removeResponse.Data.Dependencies != nil {
		suite.Len(*removeResponse.Data.Dependencies, 0, "Dependencies should be empty after removal")
	}

	// Verify by getting the task again
	getRec, getReq := suite.createRequest(http.MethodGet, fmt.Sprintf("/api/oscal/assessment-plans/%s/tasks", planID), nil)
	suite.server.E().ServeHTTP(getRec, getReq)
	suite.Equal(http.StatusOK, getRec.Code)

	var getResponse handler.GenericDataResponse[[]*oscalTypes_1_1_3.Task]
	err = json.Unmarshal(getRec.Body.Bytes(), &getResponse)
	suite.Require().NoError(err)
	suite.Require().Len(getResponse.Data, 2)

	// Find task2 in the response and verify no dependencies
	var foundTask2 *oscalTypes_1_1_3.Task
	for _, task := range getResponse.Data {
		if task.UUID == task2ID.String() {
			foundTask2 = task
			break
		}
	}
	suite.Require().NotNil(foundTask2, "Task 2 should be found in response")

	// Verify dependencies are truly removed
	if foundTask2.Dependencies != nil {
		suite.Len(*foundTask2.Dependencies, 0, "Task 2 should have no dependencies after removal")
	}
}

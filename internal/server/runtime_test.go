package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	storeschema "github.com/compliance-framework/configuration-service/internal/stores/schema"
	echo "github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestRegisterRuntime(t *testing.T) {
	drv := FakeDriver{
		GetFn: func(id string, object interface{}) error {
			if strings.Contains(id, "err") {
				return fmt.Errorf("boom")
			}
			if strings.Contains(id, "123") {
				return nil
			}
			return storeschema.NotFoundErr{}
		},
	}
	s := &Server{Driver: drv}
	p := echo.New()
	err := s.RegisterRuntime(p)
	assert.Nil(t, err)
	expected := map[string]bool{
		"GET/runtime/configurations/:uuid":    false,
		"POST/runtime/configurations":         false,
		"PUT/runtime/configurations/:uuid":    false,
		"DELETE/runtime/configurations/:uuid": false,
	}
	for _, routes := range p.Routes() {
		t := fmt.Sprintf("%s%s", routes.Method, routes.Path)
		if _, ok := expected[t]; ok {
			expected[t] = true
		}
	}
	for k, v := range expected {
		assert.True(t, v, fmt.Sprintf("expected route %s not found", k))
	}
}

type TestCase struct {
	name         string
	getFn        func(id string, object interface{}) error
	updateFn     func(id string, object interface{}) error
	deleteFn     func(id string) error
	postFn       func(id string, object interface{}) error
	path         string
	params       map[string]string
	method       string
	requestPath  string
	expectedCode int
}

func TestGetConfiguration(t *testing.T) {
	tc := []TestCase{
		{
			name: "success",
			getFn: func(id string, object interface{}) error {
				return nil
			},
			path:   "/runtime/configurations/:uuid",
			method: http.MethodGet,
			params: map[string]string{
				"uuid": "123",
			},
			requestPath:  "/runtime/configurations/123",
			expectedCode: http.StatusOK,
		},
		{
			name: "driver-not-found",
			getFn: func(id string, object interface{}) error {
				return storeschema.NotFoundErr{}
			},
			path:   "/runtime/configurations/:uuid",
			method: http.MethodGet,
			params: map[string]string{
				"uuid": "123",
			},
			requestPath:  "/runtime/configurations/123",
			expectedCode: http.StatusNotFound,
		},
		{
			name: "server-error",
			getFn: func(id string, object interface{}) error {
				return fmt.Errorf("boom")
			},
			path:   "/runtime/configurations/:uuid",
			method: http.MethodGet,
			params: map[string]string{
				"uuid": "123",
			},
			requestPath:  "/runtime/configurations/123",
			expectedCode: http.StatusInternalServerError,
		},
	}
	for idx, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			drv := FakeDriver{GetFn: tc[idx].getFn}
			s := &Server{Driver: drv}
			e := echo.New()
			req := httptest.NewRequest(tc[idx].method, tc[idx].requestPath, nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath(tc[idx].path)
			for k, v := range tc[idx].params {
				c.SetParamNames(k)
				c.SetParamValues(v)
			}
			err := s.getConfiguration(c)
			assert.NoError(t, err)
			assert.Equal(t, tc[idx].expectedCode, rec.Result().StatusCode)

		})
	}
}

func TestDeleteConfiugration(t *testing.T) {
	tc := []TestCase{
		{
			name: "success",
			deleteFn: func(id string) error {
				return nil
			},
			path: "/runtime/configurations/:uuid",
			params: map[string]string{
				"uuid": "123",
			},
			requestPath:  "/runtime/configurations/123",
			expectedCode: http.StatusOK,
		},
		{
			name: "driver-not-found",
			deleteFn: func(id string) error {
				return storeschema.NotFoundErr{}
			},
			path: "/runtime/configurations/:uuid",
			params: map[string]string{
				"uuid": "123",
			},
			requestPath:  "/runtime/configurations/123",
			expectedCode: http.StatusNotFound,
		},
		{
			name: "server-error",
			deleteFn: func(id string) error {
				return fmt.Errorf("boom")
			},
			path: "/runtime/configurations/:uuid",
			params: map[string]string{
				"uuid": "123",
			},
			requestPath:  "/runtime/configurations/123",
			expectedCode: http.StatusInternalServerError,
		},
	}
	for idx, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			drv := FakeDriver{GetFn: tc[idx].getFn, DeleteFn: tc[idx].deleteFn}
			s := &Server{Driver: drv}
			e := echo.New()
			req := httptest.NewRequest(http.MethodDelete, tc[idx].requestPath, nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath(tc[idx].path)
			for k, v := range tc[idx].params {
				c.SetParamNames(k)
				c.SetParamValues(v)
			}
			err := s.deleteConfiguration(c)
			assert.NoError(t, err)
			assert.Equal(t, tc[idx].expectedCode, rec.Result().StatusCode)
		})
	}
}

func TestUpdateConfiugration(t *testing.T) {
	tc := []TestCase{
		{
			name: "success",
			updateFn: func(id string, _ interface{}) error {
				return nil
			},
			path: "/runtime/configurations/:uuid",
			params: map[string]string{
				"uuid": "123",
			},
			requestPath:  "/runtime/configurations/123",
			expectedCode: http.StatusOK,
		},
		{
			name: "driver-not-found",
			updateFn: func(id string, _ interface{}) error {
				return storeschema.NotFoundErr{}
			},
			path: "/runtime/configurations/:uuid",
			params: map[string]string{
				"uuid": "123",
			},
			requestPath:  "/runtime/configurations/123",
			expectedCode: http.StatusNotFound,
		},
		{
			name: "server-error",
			updateFn: func(id string, _ interface{}) error {
				return fmt.Errorf("boom")
			},
			path: "/runtime/configurations/:uuid",
			params: map[string]string{
				"uuid": "123",
			},
			requestPath:  "/runtime/configurations/123",
			expectedCode: http.StatusInternalServerError,
		},
	}
	for idx, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			drv := FakeDriver{GetFn: tc[idx].getFn, DeleteFn: tc[idx].deleteFn, UpdateFn: tc[idx].updateFn}
			s := &Server{Driver: drv}
			e := echo.New()
			req := httptest.NewRequest(http.MethodPut, tc[idx].requestPath, nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath(tc[idx].path)
			for k, v := range tc[idx].params {
				c.SetParamNames(k)
				c.SetParamValues(v)
			}
			err := s.putConfiguration(c)
			assert.NoError(t, err)
			assert.Equal(t, tc[idx].expectedCode, rec.Result().StatusCode)
		})
	}
}

func TestPostConfiugration(t *testing.T) {
	tc := []TestCase{
		{
			name: "success",
			postFn: func(id string, _ interface{}) error {
				return nil
			},
			path:         "/runtime/configurations",
			requestPath:  "/runtime/configurations",
			expectedCode: http.StatusCreated,
		},
		{
			name: "server-error",
			postFn: func(id string, _ interface{}) error {
				return fmt.Errorf("boom")
			},
			path:         "/runtime/configurations",
			requestPath:  "/runtime/configurations",
			expectedCode: http.StatusInternalServerError,
		},
	}
	for idx, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			drv := FakeDriver{GetFn: tc[idx].getFn, DeleteFn: tc[idx].deleteFn, UpdateFn: tc[idx].updateFn, CreateFn: tc[idx].postFn}
			s := &Server{Driver: drv}
			e := echo.New()
			req := httptest.NewRequest(http.MethodPut, tc[idx].requestPath, nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath(tc[idx].path)
			for k, v := range tc[idx].params {
				c.SetParamNames(k)
				c.SetParamValues(v)
			}
			err := s.postConfiguration(c)
			assert.NoError(t, err)
			assert.Equal(t, tc[idx].expectedCode, rec.Result().StatusCode)
		})
	}
}

func TestGetJobs(t *testing.T) {
	tc := []TestCase{
		{
			name: "success",
			getFn: func(id string, _ interface{}) error {
				return nil
			},
			params: map[string]string{
				"uuid": "123",
			},
			path:         "/runtime/jobs/:uuid",
			requestPath:  "/runtime/jobs/123",
			expectedCode: http.StatusOK,
		},
		{
			name: "server-error",
			postFn: func(id string, _ interface{}) error {
				return fmt.Errorf("boom")
			},
			getFn: func(id string, _ interface{}) error {
				return fmt.Errorf("boom")
			},
			params: map[string]string{
				"uuid": "123",
			},
			path:         "/runtime/jobs/:uuid",
			requestPath:  "/runtime/jobs/123",
			expectedCode: http.StatusInternalServerError,
		},
	}
	for idx, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			drv := FakeDriver{GetFn: tc[idx].getFn, DeleteFn: tc[idx].deleteFn, UpdateFn: tc[idx].updateFn, CreateFn: tc[idx].postFn}
			s := &Server{Driver: drv}
			e := echo.New()
			req := httptest.NewRequest(http.MethodPut, tc[idx].requestPath, nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath(tc[idx].path)
			for k, v := range tc[idx].params {
				c.SetParamNames(k)
				c.SetParamValues(v)
			}
			err := s.getJob(c)
			assert.NoError(t, err)
			assert.Equal(t, tc[idx].expectedCode, rec.Result().StatusCode)
		})
	}
}
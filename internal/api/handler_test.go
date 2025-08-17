package api

import (
	"devlab/internal/types"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStartScenarioREST(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    string
		mockResponse   *types.StartScenarioResponse
		mockError      error
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:        "successful_start",
			requestBody: `{"user_id": "test-user", "scenario_type": "go"}`,
			mockResponse: &types.StartScenarioResponse{
				ScenarioID: "scn-123",
				Status:     "provisioning",
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"scenario_id": "scn-123",
				"status":      "provisioning",
			},
		},
		{
			name:           "missing_user_id",
			requestBody:    `{"scenario_type": "go"}`,
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error":   "User ID is required",
				"code":    "MISSING_USER_ID",
				"message": "user_id field cannot be empty",
			},
		},
		{
			name:           "missing_scenario_type",
			requestBody:    `{"user_id": "test-user"}`,
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error":   "Scenario type is required",
				"code":    "MISSING_SCENARIO_TYPE",
				"message": "scenario_type field cannot be empty",
			},
		},
		{
			name:           "invalid_json",
			requestBody:    `{"user_id": "test-user", "scenario_type": "go"`,
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "Invalid request format",
				"code":  "INVALID_REQUEST",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock scenario manager
			mockManager := new(MockScenarioManager)
			if tt.mockResponse != nil || tt.mockError != nil {
				mockManager.On("StartScenario", mock.Anything, mock.Anything).Return(tt.mockResponse, tt.mockError)
			}

			// Create handler
			handler := &Handler{
				Scenario: mockManager,
			}

			// Create router
			router := gin.New()
			router.POST("/scenarios/start", handler.StartScenarioREST)

			// Create request
			req, _ := http.NewRequest("POST", "/scenarios/start", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Check expected fields
			for key, expectedValue := range tt.expectedBody {
				assert.Equal(t, expectedValue, response[key], "Field %s should match", key)
			}

			// Verify mock expectations
			if tt.mockResponse != nil || tt.mockError != nil {
				mockManager.AssertExpectations(t)
			}
		})
	}
}

func TestGetScenarioStatusREST(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		scenarioID     string
		mockResponse   *types.ScenarioStatusResponse
		mockError      error
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:       "successful_status",
			scenarioID: "scn-123",
			mockResponse: &types.ScenarioStatusResponse{
				ScenarioID: "scn-123",
				UserID:     "test-user",
				Status:     "running",
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"scenario_id": "scn-123",
				"user_id":     "test-user",
				"status":      "running",
			},
		},
		{
			name:           "empty_scenario_id",
			scenarioID:     "",
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "scenario ID cannot be empty",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock scenario manager
			mockManager := new(MockScenarioManager)
			if tt.mockResponse != nil || tt.mockError != nil {
				mockManager.On("GetScenarioStatus", mock.Anything, tt.scenarioID).Return(tt.mockResponse, tt.mockError)
			}

			// Create handler
			handler := &Handler{
				Scenario: mockManager,
			}

			// Create router
			router := gin.New()
			router.GET("/scenarios/:id/status", handler.GetScenarioStatusREST)

			// Create request
			req, _ := http.NewRequest("GET", "/scenarios/"+tt.scenarioID+"/status", nil)
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Check expected fields
			for key, expectedValue := range tt.expectedBody {
				assert.Equal(t, expectedValue, response[key], "Field %s should match", key)
			}

			// Verify mock expectations
			if tt.mockResponse != nil || tt.mockError != nil {
				mockManager.AssertExpectations(t)
			}
		})
	}
}

func TestStopScenarioREST(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		scenarioID     string
		mockError      error
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:           "successful_stop",
			scenarioID:     "scn-123",
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"error":   "",
				"code":    "SUCCESS",
				"message": "Scenario stopped successfully",
			},
		},
		{
			name:           "empty_scenario_id",
			scenarioID:     "",
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "scenario ID cannot be empty",
			},
		},
		{
			name:           "scenario_not_found",
			scenarioID:     "scn-456",
			mockError:      errors.New("scenario not found"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error":   "Failed to stop scenario",
				"code":    "INTERNAL_ERROR",
				"message": "scenario not found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock scenario manager
			mockManager := new(MockScenarioManager)
			if tt.mockError != nil {
				mockManager.On("StopScenario", mock.Anything, tt.scenarioID).Return(tt.mockError)
			} else {
				mockManager.On("StopScenario", mock.Anything, tt.scenarioID).Return(nil)
			}

			// Create handler
			handler := &Handler{
				Scenario: mockManager,
			}

			// Create router
			router := gin.New()
			router.DELETE("/scenarios/:id", handler.StopScenarioREST)

			// Create request
			req, _ := http.NewRequest("DELETE", "/scenarios/"+tt.scenarioID, nil)
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Check expected fields
			for key, expectedValue := range tt.expectedBody {
				assert.Equal(t, expectedValue, response[key])
			}

			// Verify mock expectations
			mockManager.AssertExpectations(t)
		})
	}
}

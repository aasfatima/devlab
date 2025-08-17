package api

import (
	context "context"
	"devlab/internal/docker"
	"devlab/internal/scenario"
	"devlab/internal/types"
	pb "devlab/proto"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ScenarioManager interface {
	StartScenario(ctx context.Context, req *types.StartScenarioRequest) (*types.StartScenarioResponse, error)
	GetScenarioStatus(ctx context.Context, scenarioID string) (*types.ScenarioStatusResponse, error)
	GetTerminalURL(ctx context.Context, scenarioID string) (string, error)
	StopScenario(ctx context.Context, scenarioID string) error
	GetDirectoryStructure(ctx context.Context, scenarioID string) (*types.DirectoryStructureResponse, error)
}

// REST handler
type Handler struct {
	Scenario ScenarioManager
}

// StartScenarioREST godoc
// @Summary Start a new scenario
// @Description Launch a new coding environment (container) for a user
// @Tags scenarios
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body types.StartScenarioRequest true "Scenario start request"
// @Success 200 {object} types.StartScenarioResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 401 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /scenarios/start [post]
func (h *Handler) StartScenarioREST(c *gin.Context) {
	var req types.StartScenarioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "Invalid request format",
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	// Validate required fields
	if strings.TrimSpace(req.UserID) == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "User ID is required",
			Code:    "MISSING_USER_ID",
			Message: "user_id field cannot be empty",
		})
		return
	}

	if strings.TrimSpace(req.ScenarioType) == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "Scenario type is required",
			Code:    "MISSING_SCENARIO_TYPE",
			Message: "scenario_type field cannot be empty",
		})
		return
	}

	resp, err := h.Scenario.StartScenario(c.Request.Context(), &req)
	if err != nil {
		// Determine appropriate HTTP status code based on error type
		statusCode := http.StatusInternalServerError
		errorCode := "INTERNAL_ERROR"

		if errors.Is(err, docker.ErrInvalidScenarioType) {
			statusCode = http.StatusBadRequest
			errorCode = "INVALID_SCENARIO_TYPE"
		} else if errors.Is(err, docker.ErrPortUnavailable) {
			statusCode = http.StatusServiceUnavailable
			errorCode = "PORT_UNAVAILABLE"
		} else if errors.Is(err, docker.ErrTTYDFailedToStart) {
			statusCode = http.StatusInternalServerError
			errorCode = "TTYD_FAILED"
		} else if errors.Is(err, docker.ErrDockerDaemonUnavailable) {
			statusCode = http.StatusServiceUnavailable
			errorCode = "DOCKER_UNAVAILABLE"
		}

		c.JSON(statusCode, types.ErrorResponse{
			Error:   "Failed to start scenario",
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetScenarioStatusREST godoc
// @Summary Get scenario status
// @Description Get the current status of a scenario
// @Tags scenarios
// @Produce json
// @Security BearerAuth
// @Param id path string true "Scenario ID"
// @Success 200 {object} types.ScenarioStatusResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 401 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Router /scenarios/{id}/status [get]
func (h *Handler) GetScenarioStatusREST(c *gin.Context) {
	scenarioID := c.Param("id")
	if scenarioID == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "Scenario ID is required",
			Code:    "MISSING_SCENARIO_ID",
			Message: "scenario ID parameter cannot be empty",
		})
		return
	}

	resp, err := h.Scenario.GetScenarioStatus(c.Request.Context(), scenarioID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "INTERNAL_ERROR"

		if errors.Is(err, scenario.ErrScenarioNotFound) {
			statusCode = http.StatusNotFound
			errorCode = "SCENARIO_NOT_FOUND"
		} else if errors.Is(err, scenario.ErrInvalidScenarioID) {
			statusCode = http.StatusBadRequest
			errorCode = "INVALID_SCENARIO_ID"
		}

		c.JSON(statusCode, types.ErrorResponse{
			Error:   "Failed to get scenario status",
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetTerminalURLREST godoc
// @Summary Get terminal URL
// @Description Get the web terminal URL for a scenario
// @Tags scenarios
// @Produce json
// @Security BearerAuth
// @Param id path string true "Scenario ID"
// @Success 200 {object} types.TerminalURLResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 401 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Router /scenarios/{id}/terminal [get]
func (h *Handler) GetTerminalURLREST(c *gin.Context) {
	scenarioID := c.Param("id")
	if scenarioID == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "Scenario ID is required",
			Code:    "MISSING_SCENARIO_ID",
			Message: "scenario ID parameter cannot be empty",
		})
		return
	}

	terminalURL, err := h.Scenario.GetTerminalURL(c.Request.Context(), scenarioID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "INTERNAL_ERROR"

		if errors.Is(err, scenario.ErrScenarioNotFound) {
			statusCode = http.StatusNotFound
			errorCode = "SCENARIO_NOT_FOUND"
		} else if errors.Is(err, scenario.ErrScenarioNotRunning) {
			statusCode = http.StatusConflict
			errorCode = "SCENARIO_NOT_RUNNING"
		} else if errors.Is(err, docker.ErrContainerNotFound) {
			statusCode = http.StatusNotFound
			errorCode = "CONTAINER_NOT_FOUND"
		} else if errors.Is(err, docker.ErrContainerNotRunning) {
			statusCode = http.StatusConflict
			errorCode = "CONTAINER_NOT_RUNNING"
		} else if errors.Is(err, scenario.ErrInvalidScenarioID) {
			statusCode = http.StatusBadRequest
			errorCode = "INVALID_SCENARIO_ID"
		}

		c.JSON(statusCode, types.ErrorResponse{
			Error:   "Failed to get terminal URL",
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}

	resp := &types.TerminalURLResponse{
		ScenarioID: scenarioID,
		URL:        terminalURL,
		Message:    "Terminal URL retrieved successfully",
	}
	c.JSON(http.StatusOK, resp)
}

// StopScenarioREST godoc
// @Summary Stop a scenario
// @Description Stop and clean up a running scenario
// @Tags scenarios
// @Security BearerAuth
// @Param id path string true "Scenario ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} types.ErrorResponse
// @Failure 401 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Router /scenarios/{id} [delete]
func (h *Handler) StopScenarioREST(c *gin.Context) {
	scenarioID := c.Param("id")
	if scenarioID == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "Scenario ID is required",
			Code:    "MISSING_SCENARIO_ID",
			Message: "scenario ID parameter cannot be empty",
		})
		return
	}

	err := h.Scenario.StopScenario(c.Request.Context(), scenarioID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "INTERNAL_ERROR"

		if errors.Is(err, scenario.ErrScenarioNotFound) {
			statusCode = http.StatusNotFound
			errorCode = "SCENARIO_NOT_FOUND"
		} else if errors.Is(err, scenario.ErrScenarioAlreadyStopped) {
			statusCode = http.StatusConflict
			errorCode = "SCENARIO_ALREADY_STOPPED"
		} else if errors.Is(err, scenario.ErrInvalidScenarioID) {
			statusCode = http.StatusBadRequest
			errorCode = "INVALID_SCENARIO_ID"
		} else if errors.Is(err, docker.ErrContainerNotFound) {
			// Container not found is not an error for stopping
			statusCode = http.StatusOK
			errorCode = "CONTAINER_ALREADY_STOPPED"
		}

		c.JSON(statusCode, types.ErrorResponse{
			Error:   "Failed to stop scenario",
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.ErrorResponse{
		Error:   "",
		Code:    "SUCCESS",
		Message: "Scenario stopped successfully",
	})
}

// GetDirectoryStructureREST godoc
// @Summary Get directory structure
// @Description Get the file and directory structure for a scenario
// @Tags scenarios
// @Produce json
// @Security BearerAuth
// @Param id path string true "Scenario ID"
// @Success 200 {object} types.DirectoryStructureResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 401 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Router /scenarios/{id}/directory [get]
func (h *Handler) GetDirectoryStructureREST(c *gin.Context) {
	scenarioID := c.Param("id")
	if scenarioID == "" {
		c.JSON(400, gin.H{
			"error": "scenario ID cannot be empty",
		})
		return
	}

	resp, err := h.Scenario.GetDirectoryStructure(c.Request.Context(), scenarioID)
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, resp)
}

// GetScenarioTypesREST returns information about available scenario types
func (h *Handler) GetScenarioTypesREST(c *gin.Context) {
	scenarioTypes := []map[string]interface{}{
		{
			"type":             "go",
			"description":      "Go development environment with Go tools",
			"image":            "devlab-go:latest",
			"tools":            []string{"go", "git", "vim", "nano"},
			"example_commands": []string{"go run main.go", "go mod init myapp", "go test ./..."},
			"status":           "production-ready",
			"test_coverage":    "comprehensive",
		},
		{
			"type":             "docker",
			"description":      "Docker-in-Docker environment for container development",
			"image":            "devlab-docker:latest",
			"tools":            []string{"docker", "docker-compose"},
			"example_commands": []string{"docker run hello-world", "docker build .", "docker-compose up"},
			"status":           "production-ready",
			"test_coverage":    "good",
		},
		{
			"type":             "k8s",
			"description":      "Kubernetes environment with kubectl and k3s",
			"image":            "devlab-k8s:latest",
			"tools":            []string{"kubectl", "k3s"},
			"example_commands": []string{"kubectl get pods", "kubectl apply -f deployment.yaml", "k3s kubectl get nodes"},
			"status":           "production-ready",
			"test_coverage":    "good",
		},
		{
			"type":             "python",
			"description":      "Python development environment with Python tools",
			"image":            "devlab-python:latest",
			"tools":            []string{"python3", "pip", "flask"},
			"example_commands": []string{"python3 app.py", "pip install requests", "flask run"},
			"status":           "beta",
			"test_coverage":    "limited",
		},
		{
			"type":             "go-k8s",
			"description":      "Go development with Kubernetes tools",
			"image":            "devlab-go-k8s:latest",
			"tools":            []string{"go", "kubectl", "k3s"},
			"example_commands": []string{"go run main.go", "kubectl get deployments", "go test ./..."},
			"status":           "beta",
			"test_coverage":    "limited",
		},
		{
			"type":             "python-k8s",
			"description":      "Python development with Kubernetes tools",
			"image":            "devlab-python-k8s:latest",
			"tools":            []string{"python3", "kubectl", "k3s"},
			"example_commands": []string{"python3 app.py", "kubectl get services", "pip install kubernetes"},
			"status":           "beta",
			"test_coverage":    "limited",
		},
	}

	c.JSON(200, gin.H{
		"scenario_types":   scenarioTypes,
		"message":          "Available scenario types retrieved successfully",
		"total_count":      len(scenarioTypes),
		"production_ready": []string{"go", "docker", "k8s"},
		"beta":             []string{"python", "go-k8s", "python-k8s"},
	})
}

// gRPC server

type GRPCServer struct {
	pb.UnimplementedScenarioServiceServer
	Scenario ScenarioManager
}

func (s *GRPCServer) StartScenario(ctx context.Context, req *pb.StartScenarioRequest) (*pb.StartScenarioResponse, error) {
	internalReq := &types.StartScenarioRequest{
		UserID:       req.UserId,
		ScenarioType: req.ScenarioType,
		Script:       req.Script,
	}
	resp, err := s.Scenario.StartScenario(ctx, internalReq)
	if err != nil {
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "invalid scenario type"):
			return nil, status.Errorf(codes.InvalidArgument, errMsg)
		case strings.Contains(errMsg, "port already in use"):
			return nil, status.Errorf(codes.Internal, errMsg)
		case strings.Contains(errMsg, "container not found"):
			return nil, status.Errorf(codes.Internal, errMsg)
		case strings.Contains(errMsg, "database connection failed"):
			return nil, status.Errorf(codes.Internal, errMsg)
		default:
			return nil, status.Errorf(codes.Internal, errMsg)
		}
	}
	return &pb.StartScenarioResponse{
		ScenarioId: resp.ScenarioID,
		Status:     resp.Status,
	}, nil
}

func (s *GRPCServer) GetScenarioStatus(ctx context.Context, req *pb.GetScenarioStatusRequest) (*pb.GetScenarioStatusResponse, error) {
	resp, err := s.Scenario.GetScenarioStatus(ctx, req.ScenarioId)
	if err != nil {
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "scenario not found"):
			return nil, status.Errorf(codes.NotFound, errMsg)
		case strings.Contains(errMsg, "database connection failed"):
			return nil, status.Errorf(codes.Internal, errMsg)
		default:
			return nil, status.Errorf(codes.Internal, errMsg)
		}
	}
	return &pb.GetScenarioStatusResponse{
		ScenarioId:      resp.ScenarioID,
		UserId:          resp.UserID,
		ScenarioType:    resp.ScenarioType,
		ContainerId:     resp.ContainerID,
		Status:          resp.Status,
		ContainerStatus: resp.ContainerStatus,
		Message:         resp.Message,
	}, nil
}

func (s *GRPCServer) GetTerminalURL(ctx context.Context, req *pb.GetTerminalURLRequest) (*pb.GetTerminalURLResponse, error) {
	terminalURL, err := s.Scenario.GetTerminalURL(ctx, req.ScenarioId)
	if err != nil {
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "scenario not found"):
			return nil, status.Errorf(codes.NotFound, errMsg)
		case strings.Contains(errMsg, "container not running"):
			return nil, status.Errorf(codes.FailedPrecondition, errMsg)
		default:
			return nil, status.Errorf(codes.Internal, errMsg)
		}
	}
	return &pb.GetTerminalURLResponse{
		ScenarioId: req.ScenarioId,
		Url:        terminalURL,
		Message:    "Terminal URL retrieved successfully",
	}, nil
}

func (s *GRPCServer) StopScenario(ctx context.Context, req *pb.StopScenarioRequest) (*pb.StopScenarioResponse, error) {
	if req.ScenarioId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "scenario ID cannot be empty")
	}

	err := s.Scenario.StopScenario(ctx, req.ScenarioId)
	if err != nil {
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "scenario not found"):
			return nil, status.Errorf(codes.NotFound, errMsg)
		case strings.Contains(errMsg, "scenario already stopped"):
			return nil, status.Errorf(codes.FailedPrecondition, errMsg)
		default:
			return nil, status.Errorf(codes.Internal, errMsg)
		}
	}

	return &pb.StopScenarioResponse{
		Message: "Scenario stopped successfully",
	}, nil
}

func (s *GRPCServer) GetDirectoryStructure(ctx context.Context, req *pb.GetDirectoryStructureRequest) (*pb.GetDirectoryStructureResponse, error) {
	resp, err := s.Scenario.GetDirectoryStructure(ctx, req.ScenarioId)
	if err != nil {
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "scenario not found"):
			return nil, status.Errorf(codes.NotFound, errMsg)
		default:
			return nil, status.Errorf(codes.Internal, errMsg)
		}
	}

	// Map internal FileNode to proto FileNode
	var protoStructure []*pb.FileNode
	for _, node := range resp.Structure {
		protoNode := &pb.FileNode{
			Path:     node.Path,
			Type:     node.Type,
			IsRoot:   node.IsRoot,
			Children: node.Children,
			Content:  node.Content,
			IsOpen:   node.IsOpen,
			IsSaved:  node.IsSaved,
		}
		protoStructure = append(protoStructure, protoNode)
	}

	return &pb.GetDirectoryStructureResponse{
		ScenarioId: req.ScenarioId,
		Path:       resp.Path,
		Structure:  protoStructure,
		Message:    resp.Message,
	}, nil
}

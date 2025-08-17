package api

import (
	"context"
	"devlab/internal/types"

	"github.com/stretchr/testify/mock"
)

// MockScenarioManager is a consolidated mock for all API tests
type MockScenarioManager struct {
	mock.Mock
}

func (m *MockScenarioManager) StartScenario(ctx context.Context, req *types.StartScenarioRequest) (*types.StartScenarioResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.StartScenarioResponse), args.Error(1)
}

func (m *MockScenarioManager) GetScenarioStatus(ctx context.Context, scenarioID string) (*types.ScenarioStatusResponse, error) {
	args := m.Called(ctx, scenarioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.ScenarioStatusResponse), args.Error(1)
}

func (m *MockScenarioManager) GetTerminalURL(ctx context.Context, scenarioID string) (string, error) {
	args := m.Called(ctx, scenarioID)
	return args.String(0), args.Error(1)
}

func (m *MockScenarioManager) StopScenario(ctx context.Context, scenarioID string) error {
	args := m.Called(ctx, scenarioID)
	return args.Error(0)
}

func (m *MockScenarioManager) GetDirectoryStructure(ctx context.Context, scenarioID string) (*types.DirectoryStructureResponse, error) {
	args := m.Called(ctx, scenarioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.DirectoryStructureResponse), args.Error(1)
}

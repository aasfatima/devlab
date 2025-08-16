package types

// Shared request and response types to avoid circular imports

type StartScenarioRequest struct {
	UserID       string `json:"user_id"`
	ScenarioType string `json:"scenario_type"`
	Script       string `json:"script"`
}

type StartScenarioResponse struct {
	ScenarioID string `json:"scenario_id"`
	Status     string `json:"status"`
}

type ScenarioStatusResponse struct {
	ScenarioID      string `json:"scenario_id"`
	UserID          string `json:"user_id"`
	ScenarioType    string `json:"scenario_type"`
	ContainerID     string `json:"container_id"`
	Status          string `json:"status"`
	ContainerStatus string `json:"container_status,omitempty"`
	Message         string `json:"message"`
}

type TerminalURLResponse struct {
	ScenarioID string `json:"scenario_id"`
	URL        string `json:"url"`
	Message    string `json:"message"`
}

// FileNode represents a file or directory in the file tree
type FileNode struct {
	Path     string   `json:"path"`
	Type     string   `json:"type"` // "file" or "folder"
	IsRoot   bool     `json:"isRoot"`
	Children []string `json:"children,omitempty"`
	Content  string   `json:"content,omitempty"`
	IsOpen   bool     `json:"isOpen"`
	IsSaved  bool     `json:"isSaved"`
}

// DirectoryStructureResponse represents the response for directory structure endpoint
type DirectoryStructureResponse struct {
	ScenarioID string     `json:"scenario_id"`
	Path       string     `json:"path"`
	Structure  []FileNode `json:"structure"`
	Message    string     `json:"message"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

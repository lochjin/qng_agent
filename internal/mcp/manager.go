package mcp

import (
	"context"
	"fmt"
	"qng_agent/internal/config"
	"sync"
)

// MCPManager 接口，定义MCP管理器的通用方法
type MCPManager interface {
	CallTool(ctx context.Context, serverName, method string, params map[string]any) (any, error)
	CallQNGWorkflow(ctx context.Context, message string) (string, error)
	GetQNGWorkflowStatus(ctx context.Context, workflowID string) (*WorkflowStatus, error)
	SubmitWorkflowSignature(ctx context.Context, workflowID, signature string) (any, error)
	GetAllCapabilities() map[string][]Capability
}

type Manager struct {
	servers map[string]Server
	config  config.MCPConfig
	mu      sync.RWMutex
}

type Server interface {
	Call(ctx context.Context, method string, params map[string]any) (any, error)
	GetCapabilities() []Capability
	Start() error
	Stop() error
}

type Capability struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  []Parameter `json:"parameters"`
}

type Parameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

type WorkflowStatus struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	Message  string `json:"message"`
	Result   any    `json:"result,omitempty"`
}

func NewManager(config config.MCPConfig) *Manager {
	return &Manager{
		servers: make(map[string]Server),
		config:  config,
	}
}

func (m *Manager) RegisterServer(name string, server Server) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.servers[name] = server

	// 启动服务器
	if err := server.Start(); err != nil {
		fmt.Printf("Failed to start MCP server %s: %v\n", name, err)
	}
}

func (m *Manager) CallTool(ctx context.Context, serverName, method string, params map[string]any) (any, error) {
	m.mu.RLock()
	server, exists := m.servers[serverName]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("server %s not found", serverName)
	}

	return server.Call(ctx, method, params)
}

func (m *Manager) CallQNGWorkflow(ctx context.Context, message string) (string, error) {
	m.mu.RLock()
	qngServer, exists := m.servers["qng"]
	m.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("QNG server not found")
	}

	result, err := qngServer.Call(ctx, "start_workflow", map[string]any{
		"message": message,
	})
	if err != nil {
		return "", err
	}

	workflowID, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("invalid workflow ID type")
	}

	return workflowID, nil
}

func (m *Manager) GetQNGWorkflowStatus(ctx context.Context, workflowID string) (*WorkflowStatus, error) {
	m.mu.RLock()
	qngServer, exists := m.servers["qng"]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("QNG server not found")
	}

	result, err := qngServer.Call(ctx, "get_workflow_status", map[string]any{
		"workflow_id": workflowID,
	})
	if err != nil {
		return nil, err
	}

	status, ok := result.(*WorkflowStatus)
	if !ok {
		return nil, fmt.Errorf("invalid workflow status type")
	}

	return status, nil
}

func (m *Manager) SubmitWorkflowSignature(ctx context.Context, workflowID, signature string) (any, error) {
	m.mu.RLock()
	qngServer, exists := m.servers["qng"]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("QNG server not found")
	}

	result, err := qngServer.Call(ctx, "submit_signature", map[string]any{
		"workflow_id": workflowID,
		"signature":   signature,
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (m *Manager) GetAllCapabilities() map[string][]Capability {
	m.mu.RLock()
	defer m.mu.RUnlock()

	capabilities := make(map[string][]Capability)
	for name, server := range m.servers {
		capabilities[name] = server.GetCapabilities()
	}

	return capabilities
}

func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, server := range m.servers {
		if err := server.Stop(); err != nil {
			fmt.Printf("Failed to stop MCP server %s: %v\n", name, err)
		}
	}
}

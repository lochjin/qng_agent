package agent

import (
	"context"
	"fmt"
	"log"
	"qng_agent/internal/config"
	"qng_agent/internal/llm"
	"qng_agent/internal/mcp"
	"strings"
	"time"
)

type Manager struct {
	mcpServer *mcp.Server
	llmClient llm.Client
	sessions  map[string]*Session
}

type Session struct {
	ID           string
	Messages     []Message
	CurrentState string
	CreatedAt    time.Time
}

type Message struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type ProcessRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

type ProcessResponse struct {
	Response   string `json:"response"`
	NeedAction bool   `json:"need_action"`
	ActionType string `json:"action_type,omitempty"`
	ActionData any    `json:"action_data,omitempty"`
	WorkflowID string `json:"workflow_id,omitempty"`
}

func NewManager(mcpServer *mcp.Server, llmConfig config.LLMConfig) *Manager {
	llmClient, err := llm.NewClient(llmConfig)
	if err != nil {
		log.Fatal("Failed to create LLM client:", err)
	}

	return &Manager{
		mcpServer: mcpServer,
		llmClient: llmClient,
		sessions:  make(map[string]*Session),
	}
}

func (m *Manager) ProcessMessage(ctx context.Context, req ProcessRequest) (*ProcessResponse, error) {
	session := m.getOrCreateSession(req.SessionID)

	// 添加用户消息到会话
	userMsg := Message{
		Role:      "user",
		Content:   req.Message,
		Timestamp: time.Now(),
	}
	session.Messages = append(session.Messages, userMsg)

	// 检查是否需要工具调用
	needsTools, toolInfo := m.analyzeMessageForTools(req.Message)

	if !needsTools {
		// 直接调用LLM
		response, err := m.llmClient.Chat(ctx, m.buildLLMMessages(session))
		if err != nil {
			return nil, fmt.Errorf("LLM call failed: %w", err)
		}

		// 添加助手回复到会话
		assistantMsg := Message{
			Role:      "assistant",
			Content:   response,
			Timestamp: time.Now(),
		}
		session.Messages = append(session.Messages, assistantMsg)

		return &ProcessResponse{
			Response: response,
		}, nil
	}

	// 需要工具调用
	if toolInfo.IsQNGWorkflow {
		// 调用QNG工作流
		log.Printf("调用QNG工作流，消息: %s", req.Message)
		result, err := m.mcpServer.Call(ctx, "qng", "execute_workflow", map[string]any{
			"message": req.Message,
		})
		if err != nil {
			log.Printf("QNG工作流调用失败: %v", err)
			return nil, fmt.Errorf("QNG workflow call failed: %w", err)
		}
		resMap, _ := result.(map[string]any)
		workflowID, _ := resMap["workflow_id"].(string)
		log.Printf("QNG工作流启动成功，ID: %s", workflowID)
		return &ProcessResponse{
			Response:   "任务正在执行中，请等待...",
			NeedAction: true,
			ActionType: "workflow_running",
			WorkflowID: workflowID,
		}, nil
	}

	// 调用其它MCP工具
	log.Printf("调用MCP工具，服务器: %s, 工具: %s", toolInfo.ServerName, toolInfo.ToolName)
	result, err := m.mcpServer.Call(ctx, toolInfo.ServerName, toolInfo.ToolName, toolInfo.Parameters)
	if err != nil {
		log.Printf("MCP工具调用失败: %v", err)
		return nil, fmt.Errorf("MCP tool call failed: %w", err)
	}

	// 将结果传给LLM格式化
	llmMessages := m.buildLLMMessages(session)
	llmMessages = append(llmMessages, llm.Message{
		Role:    "system",
		Content: fmt.Sprintf("Tool result: %s", result),
	})

	response, err := m.llmClient.Chat(ctx, llmMessages)
	if err != nil {
		return nil, fmt.Errorf("LLM formatting failed: %w", err)
	}

	assistantMsg := Message{
		Role:      "assistant",
		Content:   response,
		Timestamp: time.Now(),
	}
	session.Messages = append(session.Messages, assistantMsg)

	return &ProcessResponse{
		Response: response,
	}, nil
}

type ToolInfo struct {
	IsQNGWorkflow bool
	ServerName    string
	ToolName      string
	Parameters    map[string]any
}

func (m *Manager) analyzeMessageForTools(message string) (bool, ToolInfo) {
	lowerMsg := strings.ToLower(message)

	// 检查是否是工作流相关消息
	workflowKeywords := []string{
		"兑换", "质押", "交易", "swap", "stake",
		"transfer", "转账", "usdt", "btc", "eth",
	}

	for _, keyword := range workflowKeywords {
		if strings.Contains(lowerMsg, keyword) {
			return true, ToolInfo{
				IsQNGWorkflow: true,
			}
		}
	}

	// 检查MetaMask相关操作
	if strings.Contains(lowerMsg, "钱包") || strings.Contains(lowerMsg, "metamask") ||
		strings.Contains(lowerMsg, "连接") || strings.Contains(lowerMsg, "签名") {
		return true, ToolInfo{
			ServerName: "metamask",
			ToolName:   "connect_wallet",
		}
	}

	return false, ToolInfo{}
}

func (m *Manager) getOrCreateSession(sessionID string) *Session {
	if session, exists := m.sessions[sessionID]; exists {
		return session
	}

	session := &Session{
		ID:           sessionID,
		Messages:     make([]Message, 0),
		CurrentState: "active",
		CreatedAt:    time.Now(),
	}
	m.sessions[sessionID] = session
	return session
}

func (m *Manager) buildLLMMessages(session *Session) []llm.Message {
	messages := make([]llm.Message, 0, len(session.Messages))

	// 添加系统提示
	messages = append(messages, llm.Message{
		Role: "system",
		Content: `你是一个智能区块链助手，可以帮助用户进行各种DeFi操作。
你可以调用以下工具：
1. QNG工作流 - 处理复杂的DeFi操作流程
2. MetaMask - 钱包连接和签名操作

请根据用户需求提供准确的帮助。`,
	})

	// 转换会话消息
	for _, msg := range session.Messages {
		messages = append(messages, llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return messages
}

func (m *Manager) GetWorkflowStatus(ctx context.Context, workflowID string) (*mcp.WorkflowStatus, error) {
	result, err := m.mcpServer.Call(ctx, "qng", "get_session_status", map[string]any{"session_id": workflowID})
	if err != nil {
		return nil, err
	}
	status, ok := result.(*mcp.WorkflowStatus)
	if !ok {
		return nil, fmt.Errorf("invalid workflow status type: %T", result)
	}
	return status, nil
}

func (m *Manager) ContinueWorkflowWithSignature(ctx context.Context, workflowID, signature string) (any, error) {
	return m.mcpServer.Call(ctx, "qng", "submit_signature", map[string]any{"session_id": workflowID, "signature": signature})
}

func (m *Manager) GetCapabilities() map[string]any {
	return map[string]any{
		"llm": map[string]any{
			"enabled":   true,
			"providers": []string{"openai", "anthropic"},
		},
		"mcp_servers": map[string]any{
			"qng": map[string]any{
				"enabled":     true,
				"workflows":   []string{"swap", "stake", "transfer"},
				"description": "QNG blockchain workflow execution",
			},
			"metamask": map[string]any{
				"enabled":     true,
				"tools":       []string{"connect_wallet", "sign_transaction", "get_balance"},
				"description": "MetaMask wallet integration",
			},
		},
		"features": []string{
			"natural_language_processing",
			"workflow_execution",
			"wallet_integration",
			"transaction_signing",
		},
	}
}

package mcp

import "time"

// Capability 表示MCP服务器的能力
type Capability struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  []Parameter `json:"parameters"`
}

// Parameter 表示方法参数
type Parameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// WorkflowStatus 工作流状态
type WorkflowStatus struct {
	Status       string                 `json:"status"`
	Progress     int                    `json:"progress"`
	Message      string                 `json:"message"`
	SessionID    string                 `json:"session_id"`
	NeedSignature bool                  `json:"need_signature"`
	SignatureRequest *SignatureRequest  `json:"signature_request,omitempty"`
	Result       map[string]interface{} `json:"result,omitempty"`
	Error        string                 `json:"error,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// SignatureRequest 签名请求
type SignatureRequest struct {
	Action      string `json:"action"`
	FromToken   string `json:"from_token"`
	ToToken     string `json:"to_token"`
	Amount      string `json:"amount"`
	ToAddress   string `json:"to_address"`
	Value       string `json:"value"`
	Data        string `json:"data"`
	GasLimit    string `json:"gas_limit"`
	GasPrice    string `json:"gas_price"`
	GasFee      string `json:"gas_fee"`
	Slippage    string `json:"slippage"`
}

// Session 表示会话信息
type Session struct {
	ID               string                 `json:"id"`
	WorkflowID       string                 `json:"workflow_id"`
	Status           string                 `json:"status"` // pending, running, waiting_signature, completed, failed
	Message          string                 `json:"message"`
	Result           any                    `json:"result,omitempty"`
	Context          any                    `json:"context,omitempty"`
	SignatureRequest *SignatureRequest      `json:"signature_request,omitempty"`
	CreatedAt        string                 `json:"created_at"`
	UpdatedAt        string                 `json:"updated_at"`
	PollingChan      chan *SessionUpdate    `json:"-"`
	CancelChan       chan bool              `json:"-"`
}

// SessionUpdate 表示会话更新
type SessionUpdate struct {
	Type    string `json:"type"` // status_update, signature_request, result
	Data    any    `json:"data"`
	Session *Session `json:"session"`
} 
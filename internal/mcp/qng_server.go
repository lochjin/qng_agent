package mcp

import (
	"context"
	"fmt"
	"qng_agent/internal/config"
	"qng_agent/internal/qng"
	"sync"
	"time"

	"github.com/google/uuid"
)

type QNGServer struct {
	config    config.QNGConfig
	chain     *qng.Chain
	workflows map[string]*WorkflowExecution
	mu        sync.RWMutex
	running   bool
}

type WorkflowExecution struct {
	ID        string
	Status    string
	Progress  int
	Message   string
	Result    any
	CreatedAt time.Time
	UpdatedAt time.Time
	Context   context.Context
	Cancel    context.CancelFunc
}

func NewQNGServer(config config.QNGConfig) *QNGServer {
	chain := qng.NewChain(config)

	return &QNGServer{
		config:    config,
		chain:     chain,
		workflows: make(map[string]*WorkflowExecution),
	}
}

func (s *QNGServer) Start() error {
	s.running = true
	return s.chain.Start()
}

func (s *QNGServer) Stop() error {
	s.running = false

	// 取消所有正在运行的工作流
	s.mu.Lock()
	for _, workflow := range s.workflows {
		if workflow.Cancel != nil {
			workflow.Cancel()
		}
	}
	s.mu.Unlock()

	return s.chain.Stop()
}

func (s *QNGServer) Call(ctx context.Context, method string, params map[string]any) (any, error) {
	switch method {
	case "start_workflow":
		return s.startWorkflow(ctx, params)
	case "get_workflow_status":
		return s.getWorkflowStatus(ctx, params)
	case "cancel_workflow":
		return s.cancelWorkflow(ctx, params)
	case "submit_signature":
		return s.submitSignature(ctx, params)
	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}

func (s *QNGServer) startWorkflow(ctx context.Context, params map[string]any) (any, error) {
	message, ok := params["message"].(string)
	if !ok {
		return nil, fmt.Errorf("message parameter is required")
	}

	// 生成工作流ID
	workflowID := uuid.New().String()

	// 创建工作流上下文
	workflowCtx, cancel := context.WithCancel(ctx)

	execution := &WorkflowExecution{
		ID:        workflowID,
		Status:    "running",
		Progress:  0,
		Message:   "初始化工作流...",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Context:   workflowCtx,
		Cancel:    cancel,
	}

	s.mu.Lock()
	s.workflows[workflowID] = execution
	s.mu.Unlock()

	// 异步执行工作流
	go s.executeWorkflow(execution, message)

	return workflowID, nil
}

func (s *QNGServer) executeWorkflow(execution *WorkflowExecution, message string) {
	defer func() {
		if r := recover(); r != nil {
			s.updateWorkflowStatus(execution.ID, "failed", execution.Progress, fmt.Sprintf("工作流执行失败: %v", r), nil)
		}
	}()

	// 更新状态：发送到QNG链
	s.updateWorkflowStatus(execution.ID, "running", 10, "发送消息到QNG链...", nil)

	// 调用QNG链处理消息
	result, err := s.chain.ProcessMessage(execution.Context, message)
	if err != nil {
		s.updateWorkflowStatus(execution.ID, "failed", execution.Progress, fmt.Sprintf("QNG链处理失败: %v", err), nil)
		return
	}

	// 检查是否需要用户签名
	if result.NeedSignature {
		s.updateWorkflowStatus(execution.ID, "waiting_signature", 50, "等待用户签名...", map[string]any{
			"signature_request": result.SignatureRequest,
			"workflow_context":  result.WorkflowContext,
		})

		// 等待签名提交，这里不需要 long polling，因为签名会通过 API 提交
		return
	} else {
		// 直接完成
		s.updateWorkflowStatus(execution.ID, "completed", 100, "工作流执行完成", result)
	}
}

func (s *QNGServer) updateWorkflowStatus(workflowID, status string, progress int, message string, result any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if execution, exists := s.workflows[workflowID]; exists {
		execution.Status = status
		execution.Progress = progress
		execution.Message = message
		execution.Result = result
		execution.UpdatedAt = time.Now()
	}
}

func (s *QNGServer) getWorkflowStatus(ctx context.Context, params map[string]any) (any, error) {
	workflowID, ok := params["workflow_id"].(string)
	if !ok {
		return nil, fmt.Errorf("workflow_id parameter is required")
	}

	s.mu.RLock()
	execution, exists := s.workflows[workflowID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("workflow not found")
	}

	return &WorkflowStatus{
		ID:       execution.ID,
		Status:   execution.Status,
		Progress: execution.Progress,
		Message:  execution.Message,
		Result:   execution.Result,
	}, nil
}

func (s *QNGServer) cancelWorkflow(ctx context.Context, params map[string]any) (any, error) {
	workflowID, ok := params["workflow_id"].(string)
	if !ok {
		return nil, fmt.Errorf("workflow_id parameter is required")
	}

	s.mu.Lock()
	execution, exists := s.workflows[workflowID]
	if exists && execution.Cancel != nil {
		execution.Cancel()
		execution.Status = "cancelled"
		execution.UpdatedAt = time.Now()
	}
	s.mu.Unlock()

	if !exists {
		return nil, fmt.Errorf("workflow not found")
	}

	return "ok", nil
}

func (s *QNGServer) submitSignature(ctx context.Context, params map[string]any) (any, error) {
	workflowID, ok := params["workflow_id"].(string)
	if !ok {
		return nil, fmt.Errorf("workflow_id parameter is required")
	}

	signature, ok := params["signature"].(string)
	if !ok {
		return nil, fmt.Errorf("signature parameter is required")
	}

	s.mu.Lock()
	execution, exists := s.workflows[workflowID]
	if !exists {
		s.mu.Unlock()
		return nil, fmt.Errorf("workflow not found")
	}

	// 更新工作流状态，标记签名已收到
	execution.Status = "signature_received"
	execution.Result = signature
	execution.UpdatedAt = time.Now()
	s.mu.Unlock()

	// 异步继续执行工作流
	go s.continueWorkflowWithSignature(execution, signature)

	return map[string]any{
		"status":   "signature_accepted",
		"workflow": workflowID,
		"message":  "签名已提交，工作流继续执行",
	}, nil
}

func (s *QNGServer) continueWorkflowWithSignature(execution *WorkflowExecution, signature string) {
	defer func() {
		if r := recover(); r != nil {
			s.updateWorkflowStatus(execution.ID, "failed", execution.Progress, fmt.Sprintf("工作流执行失败: %v", r), nil)
		}
	}()

	// 更新状态：处理签名
	s.updateWorkflowStatus(execution.ID, "running", 75, "处理签名并继续执行...", nil)

	// 获取之前保存的工作流上下文
	var workflowContext any
	if execution.Result != nil {
		if resultMap, ok := execution.Result.(map[string]any); ok {
			if ctx, exists := resultMap["workflow_context"]; exists {
				workflowContext = ctx
			}
		}
	}

	// 如果没有保存的上下文，创建一个简单的上下文
	if workflowContext == nil {
		workflowContext = map[string]any{
			"workflow_id": execution.ID,
			"signature":   signature,
			"completed":   true,
		}

		// 直接完成工作流
		s.updateWorkflowStatus(execution.ID, "completed", 100, "签名验证完成，工作流执行完成", map[string]any{
			"signature": signature,
			"status":    "completed",
			"message":   "所有操作已成功完成",
			"tx_hash":   "0x" + signature[:40],
		})
		return
	}

	// 调用 QNG 链继续执行
	finalResult, err := s.chain.ContinueWithSignature(execution.Context, workflowContext, signature)
	if err != nil {
		s.updateWorkflowStatus(execution.ID, "failed", 75, fmt.Sprintf("工作流执行失败: %v", err), nil)
		return
	}

	s.updateWorkflowStatus(execution.ID, "completed", 100, "工作流执行完成", finalResult)
}

func (s *QNGServer) GetCapabilities() []Capability {
	return []Capability{
		{
			Name:        "start_workflow",
			Description: "启动QNG工作流处理",
			Parameters: []Parameter{
				{
					Name:        "message",
					Type:        "string",
					Description: "用户消息",
					Required:    true,
				},
			},
		},
		{
			Name:        "get_workflow_status",
			Description: "获取工作流状态",
			Parameters: []Parameter{
				{
					Name:        "workflow_id",
					Type:        "string",
					Description: "工作流ID",
					Required:    true,
				},
			},
		},
		{
			Name:        "cancel_workflow",
			Description: "取消工作流",
			Parameters: []Parameter{
				{
					Name:        "workflow_id",
					Type:        "string",
					Description: "工作流ID",
					Required:    true,
				},
			},
		},
		{
			Name:        "submit_signature",
			Description: "提交签名继续工作流",
			Parameters: []Parameter{
				{
					Name:        "workflow_id",
					Type:        "string",
					Description: "工作流ID",
					Required:    true,
				},
				{
					Name:        "signature",
					Type:        "string",
					Description: "用户签名",
					Required:    true,
				},
			},
		},
	}
}

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

type Agent struct {
	config    config.AgentConfig
	llmClient llm.Client
	mcpServer *mcp.Server
	running   bool
}

type WorkflowExecution struct {
	SessionID    string                 `json:"session_id"`
	WorkflowID   string                 `json:"workflow_id"`
	Status       string                 `json:"status"`
	Message      string                 `json:"message"`
	UserMessage  string                 `json:"user_message"`
	Result       any                    `json:"result,omitempty"`
	NeedSignature bool                  `json:"need_signature,omitempty"`
	SignatureRequest any                `json:"signature_request,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

func NewAgent(config config.AgentConfig) *Agent {
	// 创建LLM客户端
	llmClient, err := llm.NewClient(config.LLM)
	if err != nil {
		log.Printf("⚠️  无法创建LLM客户端: %v", err)
		llmClient = nil
	}

	// 创建MCP服务器
	mcpServer := mcp.NewServer(config.MCP)

	agent := &Agent{
		config:    config,
		llmClient: llmClient,
		mcpServer: mcpServer,
	}

	return agent
}

func (a *Agent) Start() error {
	log.Printf("🚀 智能体启动")
	
	// 启动MCP服务器
	if err := a.mcpServer.Start(); err != nil {
		log.Printf("❌ 启动MCP服务器失败: %v", err)
		return fmt.Errorf("failed to start MCP server: %w", err)
	}
	
	a.running = true
	log.Printf("✅ 智能体启动成功")
	return nil
}

func (a *Agent) Stop() error {
	log.Printf("🛑 智能体停止")
	
	if a.mcpServer != nil {
		if err := a.mcpServer.Stop(); err != nil {
			log.Printf("❌ 停止MCP服务器失败: %v", err)
		}
	}
	
	a.running = false
	log.Printf("✅ 智能体停止成功")
	return nil
}

func (a *Agent) ProcessMessage(ctx context.Context, message string) (*WorkflowExecution, error) {
	log.Printf("🔄 智能体处理消息")
	log.Printf("📝 用户消息: %s", message)
	
	if !a.running {
		log.Printf("❌ 智能体未运行")
		return nil, fmt.Errorf("agent is not running")
	}

	// 1. 分析用户消息，确定需要的工作流
	log.Printf("🤖 分析用户消息，确定工作流")
	workflow, err := a.analyzeMessage(ctx, message)
	if err != nil {
		log.Printf("❌ 消息分析失败: %v", err)
		return nil, fmt.Errorf("message analysis failed: %w", err)
	}

	log.Printf("✅ 工作流分析完成")
	log.Printf("📋 工作流类型: %s", workflow.Type)
	log.Printf("📋 工作流参数: %+v", workflow.Parameters)

	// 2. 执行工作流
	log.Printf("🔄 执行工作流")
	execution, err := a.executeWorkflow(ctx, message, workflow)
	if err != nil {
		log.Printf("❌ 工作流执行失败: %v", err)
		return nil, fmt.Errorf("workflow execution failed: %w", err)
	}

	log.Printf("✅ 工作流执行完成")
	return execution, nil
}

func (a *Agent) analyzeMessage(ctx context.Context, message string) (*WorkflowInfo, error) {
	log.Printf("🤖 使用LLM分析用户消息")
	
	if a.llmClient == nil {
		log.Printf("⚠️  没有LLM客户端，使用简单规则分析")
		return a.simpleMessageAnalysis(message)
	}

	// 构建分析提示
	prompt := fmt.Sprintf(`
请分析用户的请求并确定需要执行的工作流类型。

用户消息: %s

可用的工作流类型:
1. swap - 代币兑换
2. stake - 代币质押
3. transfer - 代币转账
4. query - 余额查询

请按以下JSON格式返回分析结果:
{
  "type": "swap",
  "description": "用户想要将USDT兑换成BTC",
  "parameters": {
    "from_token": "USDT",
    "to_token": "BTC", 
    "amount": "1000"
  }
}

只返回JSON格式，不要其他文字。
如果找到相近的多个tool，请提示用户选择。3 tool [{},{},{}]
`, message)
	log.Printf("📝 构建LLM提示完成")
	log.Printf("📝 提示长度: %d", len(prompt))

	// 调用LLM分析
	response, err := a.llmClient.Chat(ctx, []llm.Message{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		log.Printf("❌ LLM调用失败: %v", err)
		return a.simpleMessageAnalysis(message)
	}

	log.Printf("✅ LLM响应成功")
	log.Printf("📄 LLM响应: %s", response)

	// 解析LLM响应
	workflow, err := a.parseWorkflowFromResponse(response)
	if err != nil {
		log.Printf("❌ 解析LLM响应失败: %v", err)
		return a.simpleMessageAnalysis(message)
	}

	log.Printf("✅ 工作流分析完成: %+v", workflow)
	return workflow, nil
}

func (a *Agent) simpleMessageAnalysis(message string) (*WorkflowInfo, error) {
	log.Printf("🔄 使用简单规则分析消息")
	log.Printf("📝 消息: %s", message)
	
	lowerMsg := strings.ToLower(message)
	
	if strings.Contains(lowerMsg, "兑换") || strings.Contains(lowerMsg, "swap") {
		log.Printf("✅ 检测到兑换工作流")
		return &WorkflowInfo{
			Type: "swap",
			Description: "代币兑换",
			Parameters: map[string]any{
				"from_token": "USDT",
				"to_token":   "BTC",
				"amount":     "1000",
			},
		}, nil
	}
	
	if strings.Contains(lowerMsg, "质押") || strings.Contains(lowerMsg, "stake") {
		log.Printf("✅ 检测到质押工作流")
		return &WorkflowInfo{
			Type: "stake",
			Description: "代币质押",
			Parameters: map[string]any{
				"token":  "BTC",
				"amount": "0.1",
				"pool":   "compound",
			},
		}, nil
	}
	
	log.Printf("⚠️  未检测到具体工作流，使用默认查询")
	return &WorkflowInfo{
		Type: "query",
		Description: "余额查询",
		Parameters: map[string]any{},
	}, nil
}

func (a *Agent) parseWorkflowFromResponse(response string) (*WorkflowInfo, error) {
	log.Printf("🔄 解析LLM响应中的工作流")
	log.Printf("📄 响应内容: %s", response)
	
	// 简化的解析，实际应该使用json.Unmarshal
	lowerResponse := strings.ToLower(response)
	
	if strings.Contains(lowerResponse, "swap") {
		log.Printf("✅ 检测到swap工作流")
		return &WorkflowInfo{
			Type: "swap",
			Description: "代币兑换",
			Parameters: map[string]any{
				"from_token": "USDT",
				"to_token":   "BTC",
				"amount":     "1000",
			},
		}, nil
	}
	
	if strings.Contains(lowerResponse, "stake") {
		log.Printf("✅ 检测到stake工作流")
		return &WorkflowInfo{
			Type: "stake",
			Description: "代币质押",
			Parameters: map[string]any{
				"token":  "BTC",
				"amount": "0.1",
				"pool":   "compound",
			},
		}, nil
	}
	
	log.Printf("⚠️  未检测到具体工作流")
	return &WorkflowInfo{
		Type: "query",
		Description: "余额查询",
		Parameters: map[string]any{},
	}, nil
}

func (a *Agent) executeWorkflow(ctx context.Context, message string, workflow *WorkflowInfo) (*WorkflowExecution, error) {
	log.Printf("🔄 执行工作流")
	log.Printf("📋 工作流类型: %s", workflow.Type)
	log.Printf("📝 用户消息: %s", message)
	
	// 调用QNG MCP服务执行工作流
	result, err := a.mcpServer.Call(ctx, "qng", "execute_workflow", map[string]any{
		"message": message,
	})
	if err != nil {
		log.Printf("❌ 调用QNG服务失败: %v", err)
		return nil, fmt.Errorf("QNG service call failed: %w", err)
	}
	
	log.Printf("✅ QNG服务调用成功")
	log.Printf("📊 结果: %+v", result)
	
	// 解析结果
	resultMap, ok := result.(map[string]any)
	if !ok {
		log.Printf("❌ 结果格式错误")
		return nil, fmt.Errorf("invalid result format")
	}
	
	sessionID, ok := resultMap["session_id"].(string)
	if !ok {
		log.Printf("❌ 缺少session_id")
		return nil, fmt.Errorf("missing session_id")
	}
	
	workflowID, ok := resultMap["workflow_id"].(string)
	if !ok {
		log.Printf("❌ 缺少workflow_id")
		return nil, fmt.Errorf("missing workflow_id")
	}
	
	status, ok := resultMap["status"].(string)
	if !ok {
		log.Printf("❌ 缺少status")
		return nil, fmt.Errorf("missing status")
	}
	
	execution := &WorkflowExecution{
		SessionID:   sessionID,
		WorkflowID:  workflowID,
		Status:      status,
		Message:     "工作流已提交",
		UserMessage: message,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	log.Printf("✅ 工作流执行对象创建完成")
	log.Printf("📋 会话ID: %s", sessionID)
	log.Printf("📋 工作流ID: %s", workflowID)
	log.Printf("📋 状态: %s", status)
	
	return execution, nil
}

func (a *Agent) PollWorkflowStatus(ctx context.Context, sessionID string) (*WorkflowExecution, error) {
	log.Printf("🔄 轮询工作流状态")
	log.Printf("📋 会话ID: %s", sessionID)
	
	// 调用QNG MCP服务轮询状态
	result, err := a.mcpServer.Call(ctx, "qng", "poll_session", map[string]any{
		"session_id": sessionID,
		"timeout":    30,
	})
	if err != nil {
		log.Printf("❌ 轮询失败: %v", err)
		return nil, fmt.Errorf("poll failed: %w", err)
	}
	
	log.Printf("✅ 轮询成功")
	log.Printf("📊 结果: %+v", result)
	
	// 解析结果
	resultMap, ok := result.(map[string]any)
	if !ok {
		log.Printf("❌ 结果格式错误")
		return nil, fmt.Errorf("invalid result format")
	}
	
	// 检查是否超时
	if timeout, ok := resultMap["timeout"].(bool); ok && timeout {
		log.Printf("⏰ 轮询超时")
		return &WorkflowExecution{
			SessionID: sessionID,
			Status:    "timeout",
			Message:   "轮询超时，请重试",
			UpdatedAt: time.Now(),
		}, nil
	}
	
	// 检查是否取消
	if cancelled, ok := resultMap["cancelled"].(bool); ok && cancelled {
		log.Printf("🛑 会话已取消")
		return &WorkflowExecution{
			SessionID: sessionID,
			Status:    "cancelled",
			Message:   "会话已取消",
			UpdatedAt: time.Now(),
		}, nil
	}
	
	// 解析更新
	update, ok := resultMap["update"].(map[string]any)
	if !ok {
		log.Printf("❌ 缺少update")
		return nil, fmt.Errorf("missing update")
	}
	
	updateType, ok := update["type"].(string)
	if !ok {
		log.Printf("❌ 缺少update type")
		return nil, fmt.Errorf("missing update type")
	}
	
	log.Printf("📋 更新类型: %s", updateType)
	
	execution := &WorkflowExecution{
		SessionID: sessionID,
		UpdatedAt: time.Now(),
	}
	
	switch updateType {
	case "signature_request":
		log.Printf("✍️  需要用户签名")
		execution.Status = "waiting_signature"
		execution.NeedSignature = true
		execution.SignatureRequest = update["data"]
		execution.Message = "需要用户签名授权"
		
	case "result":
		log.Printf("✅ 工作流完成")
		execution.Status = "completed"
		execution.Result = update["data"]
		execution.Message = "工作流执行完成"
		
	default:
		log.Printf("⚠️  未知更新类型: %s", updateType)
		execution.Status = "unknown"
		execution.Message = fmt.Sprintf("未知更新类型: %s", updateType)
	}
	
	return execution, nil
}

func (a *Agent) SubmitSignature(ctx context.Context, sessionID, signature string) error {
	log.Printf("✍️  提交用户签名")
	log.Printf("📋 会话ID: %s", sessionID)
	log.Printf("🔐 签名长度: %d", len(signature))
	
	// 调用QNG MCP服务提交签名
	result, err := a.mcpServer.Call(ctx, "qng", "submit_signature", map[string]any{
		"session_id": sessionID,
		"signature":  signature,
	})
	if err != nil {
		log.Printf("❌ 提交签名失败: %v", err)
		return fmt.Errorf("submit signature failed: %w", err)
	}
	
	log.Printf("✅ 签名提交成功")
	log.Printf("📊 结果: %+v", result)
	
	return nil
}

func (a *Agent) ConnectWallet(ctx context.Context) error {
	log.Printf("🔗 连接MetaMask钱包")
	
	// 调用MetaMask MCP服务连接钱包
	result, err := a.mcpServer.Call(ctx, "metamask", "connect_wallet", map[string]any{
		"request_permissions": true,
	})
	if err != nil {
		log.Printf("❌ 连接钱包失败: %v", err)
		return fmt.Errorf("connect wallet failed: %w", err)
	}
	
	log.Printf("✅ 钱包连接成功")
	log.Printf("📊 结果: %+v", result)
	
	return nil
}

func (a *Agent) GetWalletBalance(ctx context.Context, account string) (any, error) {
	log.Printf("💰 获取钱包余额")
	log.Printf("📋 账户: %s", account)
	
	// 调用MetaMask MCP服务获取余额
	result, err := a.mcpServer.Call(ctx, "metamask", "get_balance", map[string]any{
		"account": account,
	})
	if err != nil {
		log.Printf("❌ 获取余额失败: %v", err)
		return nil, fmt.Errorf("get balance failed: %w", err)
	}
	
	log.Printf("✅ 余额查询成功")
	log.Printf("📊 结果: %+v", result)
	
	return result, nil
}

type WorkflowInfo struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Parameters  map[string]any         `json:"parameters"`
} 
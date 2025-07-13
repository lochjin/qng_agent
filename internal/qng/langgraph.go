package qng

import (
	"context"
	"fmt"
	"log"
	"qng_agent/internal/llm"
	"time"
)

// LangGraph 节点系统
type LangGraph struct {
	nodes map[string]Node
	edges map[string][]string
	llm   llm.Client
}

// Node 节点接口
type Node interface {
	Execute(ctx context.Context, input NodeInput) (*NodeOutput, error)
	GetName() string
	GetType() string
}

// NodeInput 节点输入
type NodeInput struct {
	Data    map[string]any `json:"data"`
	Context map[string]any `json:"context"`
}

// NodeOutput 节点输出
type NodeOutput struct {
	Data         map[string]any `json:"data"`
	NextNodes    []string       `json:"next_nodes"`
	NeedUserAuth bool           `json:"need_user_auth"`
	AuthRequest  any            `json:"auth_request,omitempty"`
	Completed    bool           `json:"completed"`
}

// NewLangGraph 创建LangGraph实例
func NewLangGraph(llmClient llm.Client) *LangGraph {
	lg := &LangGraph{
		nodes: make(map[string]Node),
		edges: make(map[string][]string),
		llm:   llmClient,
	}

	// 注册节点
	lg.registerNodes()
	
	// 构建图结构
	lg.buildGraph()

	return lg
}

// registerNodes 注册所有节点
func (lg *LangGraph) registerNodes() {
	// 任务分解节点
	lg.nodes["task_decomposer"] = NewTaskDecomposerNode(lg.llm)
	
	// 交易执行节点
	lg.nodes["swap_executor"] = NewSwapExecutorNode()
	
	// 质押执行节点
	lg.nodes["stake_executor"] = NewStakeExecutorNode()
	
	// 签名验证节点
	lg.nodes["signature_validator"] = NewSignatureValidatorNode()
	
	// 结果聚合节点
	lg.nodes["result_aggregator"] = NewResultAggregatorNode()
}

// buildGraph 构建图结构
func (lg *LangGraph) buildGraph() {
	lg.edges = map[string][]string{
		"task_decomposer":     {"swap_executor", "stake_executor"},
		"swap_executor":       {"signature_validator"},
		"stake_executor":      {"signature_validator"},
		"signature_validator": {"result_aggregator"},
		"result_aggregator":   {}, // 终止节点
	}
}

// ExecuteWorkflow 执行工作流
func (lg *LangGraph) ExecuteWorkflow(ctx context.Context, message string) (*ProcessResult, error) {
	log.Printf("🔄 LangGraph开始执行工作流")
	log.Printf("📝 用户消息: %s", message)

	// 初始化输入
	input := NodeInput{
		Data: map[string]any{
			"user_message": message,
			"timestamp":    time.Now(),
		},
		Context: map[string]any{
			"workflow_id": ctx.Value("workflow_id"),
			"session_id":  ctx.Value("session_id"),
		},
	}

	// 从任务分解节点开始执行
	return lg.executeNode(ctx, "task_decomposer", input)
}

// executeNode 执行单个节点
func (lg *LangGraph) executeNode(ctx context.Context, nodeName string, input NodeInput) (*ProcessResult, error) {
	log.Printf("🔄 执行节点: %s", nodeName)
	
	node, exists := lg.nodes[nodeName]
	if !exists {
		log.Printf("❌ 节点不存在: %s", nodeName)
		return nil, fmt.Errorf("node %s not found", nodeName)
	}

	log.Printf("✅ 找到节点: %s (类型: %s)", nodeName, node.GetType())

	// 执行节点
	output, err := node.Execute(ctx, input)
	if err != nil {
		log.Printf("❌ 节点执行失败: %v", err)
		return nil, fmt.Errorf("node %s execution failed: %w", nodeName, err)
	}

	log.Printf("✅ 节点执行成功")
	log.Printf("📊 输出数据: %+v", output.Data)
	log.Printf("➡️  下一个节点: %v", output.NextNodes)
	log.Printf("🔐 需要用户授权: %v", output.NeedUserAuth)
	log.Printf("✅ 是否完成: %v", output.Completed)

	// 检查是否需要用户授权
	if output.NeedUserAuth {
		log.Printf("✍️  需要用户签名授权")
		log.Printf("📋 授权请求: %+v", output.AuthRequest)
		
		return &ProcessResult{
			NeedSignature:    true,
			SignatureRequest: output.AuthRequest,
			WorkflowContext: map[string]any{
				"current_node": nodeName,
				"node_output":  output,
				"input":        input,
			},
		}, nil
	}

	// 检查是否已完成
	if output.Completed {
		log.Printf("✅ 工作流执行完成")
		return &ProcessResult{
			FinalResult: output.Data,
		}, nil
	}

	// 继续执行下一个节点
	if len(output.NextNodes) > 0 {
		nextNode := output.NextNodes[0] // 简化处理，取第一个
		log.Printf("➡️  继续执行下一个节点: %s", nextNode)
		
		nextInput := NodeInput{
			Data:    output.Data,
			Context: input.Context,
		}

		return lg.executeNode(ctx, nextNode, nextInput)
	}

	// 没有下一个节点，工作流完成
	log.Printf("✅ 没有下一个节点，工作流完成")
	return &ProcessResult{
		FinalResult: output.Data,
	}, nil
}

// ContinueWithSignature 使用签名继续工作流
func (lg *LangGraph) ContinueWithSignature(ctx context.Context, workflowContext any, signature string) (any, error) {
	log.Printf("🔄 使用签名继续工作流")
	log.Printf("🔐 签名长度: %d", len(signature))
	
	// 从工作流上下文恢复执行状态
	contextMap, ok := workflowContext.(map[string]any)
	if !ok {
		log.Printf("❌ 无效的工作流上下文类型")
		return nil, fmt.Errorf("invalid workflow context")
	}

	currentNode, ok := contextMap["current_node"].(string)
	if !ok {
		log.Printf("❌ 上下文中缺少当前节点信息")
		return nil, fmt.Errorf("invalid current node in context")
	}

	log.Printf("🔄 从节点恢复: %s", currentNode)

	nodeOutput, ok := contextMap["node_output"].(*NodeOutput)
	if !ok {
		log.Printf("❌ 上下文中缺少节点输出信息")
		return nil, fmt.Errorf("invalid node output in context")
	}

	nodeInput, ok := contextMap["input"].(NodeInput)
	if !ok {
		log.Printf("❌ 上下文中缺少节点输入信息")
		return nil, fmt.Errorf("invalid node input in context")
	}

	// 将签名添加到数据中
	log.Printf("🔐 将签名添加到节点数据中")
	nodeOutput.Data["signature"] = signature
	nodeOutput.NeedUserAuth = false

	// 继续执行下一个节点
	if len(nodeOutput.NextNodes) > 0 {
		nextNode := nodeOutput.NextNodes[0]
		log.Printf("➡️  继续执行下一个节点: %s", nextNode)
		
		nextInput := NodeInput{
			Data:    nodeOutput.Data,
			Context: nodeInput.Context,
		}

		result, err := lg.executeNode(ctx, nextNode, nextInput)
		if err != nil {
			log.Printf("❌ 继续执行失败: %v", err)
			return nil, err
		}

		log.Printf("✅ 继续执行成功")
		return result.FinalResult, nil
	}

	log.Printf("✅ 没有下一个节点，返回当前数据")
	return nodeOutput.Data, nil
} 
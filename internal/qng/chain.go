package qng

import (
	"context"
	"fmt"
	"qng_agent/internal/config"
	"qng_agent/internal/llm"
	"sync"
	"time"
)

type Chain struct {
	config    config.QNGConfig
	llmClient llm.Client
	graph     *Graph
	nodes     map[string]Node
	mu        sync.RWMutex
	running   bool
}

type ProcessResult struct {
	NeedSignature    bool `json:"need_signature"`
	SignatureRequest any  `json:"signature_request,omitempty"`
	WorkflowContext  any  `json:"workflow_context,omitempty"`
	FinalResult      any  `json:"final_result,omitempty"`
}

type Graph struct {
	Nodes map[string]Node
	Edges map[string][]string // node -> next nodes
	Start string
}

type Node interface {
	Execute(ctx context.Context, input NodeInput) (*NodeOutput, error)
	GetName() string
	GetType() string
}

type NodeInput struct {
	Data    map[string]any `json:"data"`
	Context map[string]any `json:"context"`
}

type NodeOutput struct {
	Data         map[string]any `json:"data"`
	NextNodes    []string       `json:"next_nodes"`
	NeedUserAuth bool           `json:"need_user_auth"`
	AuthRequest  any            `json:"auth_request,omitempty"`
	Completed    bool           `json:"completed"`
}

func NewChain(config config.QNGConfig) *Chain {
	// 创建LLM客户端
	// TODO: Initialize LLM client from config

	chain := &Chain{
		config: config,
		nodes:  make(map[string]Node),
	}

	// 初始化图结构
	chain.initializeGraph()

	return chain
}

func (c *Chain) initializeGraph() {
	// 创建任务分解节点
	taskDecomposer := NewTaskDecomposerNode(c.llmClient)
	c.nodes["task_decomposer"] = taskDecomposer

	// 创建交易执行节点
	swapNode := NewSwapNode()
	c.nodes["swap_executor"] = swapNode

	// 创建质押节点
	stakeNode := NewStakeNode()
	c.nodes["stake_executor"] = stakeNode

	// 创建签名验证节点
	signatureNode := NewSignatureNode()
	c.nodes["signature_validator"] = signatureNode

	// 创建结果整合节点
	aggregatorNode := NewResultAggregatorNode()
	c.nodes["result_aggregator"] = aggregatorNode

	// 构建图结构
	c.graph = &Graph{
		Nodes: c.nodes,
		Edges: map[string][]string{
			"task_decomposer":     {"swap_executor", "stake_executor"},
			"swap_executor":       {"signature_validator", "stake_executor"},
			"stake_executor":      {"signature_validator", "result_aggregator"},
			"signature_validator": {"swap_executor", "stake_executor", "result_aggregator"},
			"result_aggregator":   {}, // 终止节点
		},
		Start: "task_decomposer",
	}
}

func (c *Chain) Start() error {
	c.running = true
	return nil
}

func (c *Chain) Stop() error {
	c.running = false
	return nil
}

func (c *Chain) ProcessMessage(ctx context.Context, message string) (*ProcessResult, error) {
	if !c.running {
		return nil, fmt.Errorf("chain is not running")
	}

	// 初始化执行上下文
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
	return c.executeNode(ctx, c.graph.Start, input)
}

func (c *Chain) executeNode(ctx context.Context, nodeName string, input NodeInput) (*ProcessResult, error) {
	c.mu.RLock()
	node, exists := c.nodes[nodeName]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("node %s not found", nodeName)
	}

	// 执行节点
	output, err := node.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("node %s execution failed: %w", nodeName, err)
	}

	// 检查是否需要用户授权
	if output.NeedUserAuth {
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
		return &ProcessResult{
			FinalResult: output.Data,
		}, nil
	}

	// 继续执行下一个节点
	if len(output.NextNodes) > 0 {
		nextNode := output.NextNodes[0] // 简化处理，取第一个
		nextInput := NodeInput{
			Data:    output.Data,
			Context: input.Context,
		}

		return c.executeNode(ctx, nextNode, nextInput)
	}

	// 没有下一个节点，工作流完成
	return &ProcessResult{
		FinalResult: output.Data,
	}, nil
}

func (c *Chain) ContinueWithSignature(ctx context.Context, workflowContext any, signature string) (any, error) {
	// 从工作流上下文恢复执行状态
	contextMap, ok := workflowContext.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid workflow context")
	}

	_, ok = contextMap["current_node"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid current node in context")
	}

	nodeOutput, ok := contextMap["node_output"].(*NodeOutput)
	if !ok {
		return nil, fmt.Errorf("invalid node output in context")
	}

	nodeInput, ok := contextMap["input"].(NodeInput)
	if !ok {
		return nil, fmt.Errorf("invalid node input in context")
	}

	// 将签名添加到数据中
	nodeOutput.Data["signature"] = signature
	nodeOutput.NeedUserAuth = false

	// 继续执行下一个节点
	if len(nodeOutput.NextNodes) > 0 {
		nextNode := nodeOutput.NextNodes[0]
		nextInput := NodeInput{
			Data:    nodeOutput.Data,
			Context: nodeInput.Context,
		}

		result, err := c.executeNode(ctx, nextNode, nextInput)
		if err != nil {
			return nil, err
		}

		return result.FinalResult, nil
	}

	return nodeOutput.Data, nil
}

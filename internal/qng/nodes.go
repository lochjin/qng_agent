package qng

import (
	"context"
	"fmt"
	"qng_agent/internal/llm"
	"strings"
)

// TaskDecomposerNode 任务分解节点
type TaskDecomposerNode struct {
	llmClient llm.Client
}

func NewTaskDecomposerNode(llmClient llm.Client) *TaskDecomposerNode {
	return &TaskDecomposerNode{
		llmClient: llmClient,
	}
}

func (n *TaskDecomposerNode) GetName() string {
	return "task_decomposer"
}

func (n *TaskDecomposerNode) GetType() string {
	return "llm_processor"
}

func (n *TaskDecomposerNode) Execute(ctx context.Context, input NodeInput) (*NodeOutput, error) {
	userMessage, ok := input.Data["user_message"].(string)
	if !ok {
		return nil, fmt.Errorf("user_message not found in input")
	}

	// 构建LLM提示
	prompt := fmt.Sprintf(`
请分析用户的请求并分解为具体的执行步骤。

用户请求: %s

请按以下格式返回分解结果：
{
  "tasks": [
    {
      "type": "swap",
      "from_token": "USDT", 
      "to_token": "BTC",
      "amount": "1000"
    },
    {
      "type": "stake",
      "token": "BTC",
      "amount": "0.1",
      "pool": "compound"
    }
  ]
}

只返回JSON格式，不要其他文字。
`, userMessage)

	// 调用LLM进行任务分解
	if n.llmClient != nil {
		response, err := n.llmClient.Chat(ctx, []llm.Message{
			{Role: "user", Content: prompt},
		})
		if err != nil {
			return nil, fmt.Errorf("LLM call failed: %w", err)
		}

		// 解析LLM响应
		tasks := n.parseTasksFromResponse(response)

		// 决定下一个执行节点
		nextNodes := n.determineNextNodes(tasks)

		return &NodeOutput{
			Data: map[string]any{
				"tasks":         tasks,
				"user_message":  userMessage,
				"decomposed_at": input.Data["timestamp"],
			},
			NextNodes: nextNodes,
			Completed: false,
		}, nil
	}

	// 如果没有LLM客户端，使用简单的规则分解
	tasks := n.simpleTaskDecomposition(userMessage)
	nextNodes := n.determineNextNodes(tasks)

	return &NodeOutput{
		Data: map[string]any{
			"tasks":        tasks,
			"user_message": userMessage,
		},
		NextNodes: nextNodes,
		Completed: false,
	}, nil
}

func (n *TaskDecomposerNode) parseTasksFromResponse(response string) []map[string]any {
	// 简化的JSON解析，实际应该使用json.Unmarshal
	if strings.Contains(strings.ToLower(response), "swap") {
		return []map[string]any{
			{
				"type":       "swap",
				"from_token": "USDT",
				"to_token":   "BTC",
				"amount":     "1000",
			},
		}
	}

	return []map[string]any{}
}

func (n *TaskDecomposerNode) simpleTaskDecomposition(message string) []map[string]any {
	lowerMsg := strings.ToLower(message)
	tasks := make([]map[string]any, 0)

	if strings.Contains(lowerMsg, "兑换") || strings.Contains(lowerMsg, "swap") {
		tasks = append(tasks, map[string]any{
			"type":       "swap",
			"from_token": "USDT",
			"to_token":   "BTC",
			"amount":     "1000",
		})
	}

	if strings.Contains(lowerMsg, "质押") || strings.Contains(lowerMsg, "stake") {
		tasks = append(tasks, map[string]any{
			"type":   "stake",
			"token":  "BTC",
			"amount": "0.1",
			"pool":   "compound",
		})
	}

	return tasks
}

func (n *TaskDecomposerNode) determineNextNodes(tasks []map[string]any) []string {
	for _, task := range tasks {
		if taskType, ok := task["type"].(string); ok {
			switch taskType {
			case "swap":
				return []string{"swap_executor"}
			case "stake":
				return []string{"stake_executor"}
			}
		}
	}
	return []string{"result_aggregator"}
}

// SwapNode 交易执行节点
type SwapNode struct{}

func NewSwapNode() *SwapNode {
	return &SwapNode{}
}

func (n *SwapNode) GetName() string {
	return "swap_executor"
}

func (n *SwapNode) GetType() string {
	return "transaction_executor"
}

func (n *SwapNode) Execute(ctx context.Context, input NodeInput) (*NodeOutput, error) {
	tasks, ok := input.Data["tasks"].([]map[string]any)
	if !ok {
		return nil, fmt.Errorf("tasks not found in input")
	}

	// 查找swap任务
	var swapTask map[string]any
	for _, task := range tasks {
		if taskType, ok := task["type"].(string); ok && taskType == "swap" {
			swapTask = task
			break
		}
	}

	if swapTask == nil {
		// 没有swap任务，跳到下一个节点
		return &NodeOutput{
			Data:      input.Data,
			NextNodes: []string{"stake_executor"},
			Completed: false,
		}, nil
	}

	// 需要用户签名授权交易
	authRequest := map[string]any{
		"type":       "transaction_signature",
		"action":     "swap",
		"from_token": swapTask["from_token"],
		"to_token":   swapTask["to_token"],
		"amount":     swapTask["amount"],
		"gas_fee":    "0.001 ETH",
		"slippage":   "0.5%",
	}

	return &NodeOutput{
		Data:         input.Data,
		NextNodes:    []string{"signature_validator"},
		NeedUserAuth: true,
		AuthRequest:  authRequest,
		Completed:    false,
	}, nil
}

// StakeNode 质押执行节点
type StakeNode struct{}

func NewStakeNode() *StakeNode {
	return &StakeNode{}
}

func (n *StakeNode) GetName() string {
	return "stake_executor"
}

func (n *StakeNode) GetType() string {
	return "transaction_executor"
}

func (n *StakeNode) Execute(ctx context.Context, input NodeInput) (*NodeOutput, error) {
	tasks, ok := input.Data["tasks"].([]map[string]any)
	if !ok {
		return nil, fmt.Errorf("tasks not found in input")
	}

	// 查找stake任务
	var stakeTask map[string]any
	for _, task := range tasks {
		if taskType, ok := task["type"].(string); ok && taskType == "stake" {
			stakeTask = task
			break
		}
	}

	if stakeTask == nil {
		// 没有stake任务，结束流程
		return &NodeOutput{
			Data:      input.Data,
			NextNodes: []string{"result_aggregator"},
			Completed: false,
		}, nil
	}

	// 需要用户签名授权质押
	authRequest := map[string]any{
		"type":    "transaction_signature",
		"action":  "stake",
		"token":   stakeTask["token"],
		"amount":  stakeTask["amount"],
		"pool":    stakeTask["pool"],
		"gas_fee": "0.001 ETH",
		"apy":     "8.5%",
	}

	return &NodeOutput{
		Data:         input.Data,
		NextNodes:    []string{"signature_validator"},
		NeedUserAuth: true,
		AuthRequest:  authRequest,
		Completed:    false,
	}, nil
}

// SignatureNode 签名验证节点
type SignatureNode struct{}

func NewSignatureNode() *SignatureNode {
	return &SignatureNode{}
}

func (n *SignatureNode) GetName() string {
	return "signature_validator"
}

func (n *SignatureNode) GetType() string {
	return "validator"
}

func (n *SignatureNode) Execute(ctx context.Context, input NodeInput) (*NodeOutput, error) {
	signature, ok := input.Data["signature"].(string)
	if !ok || signature == "" {
		return nil, fmt.Errorf("signature not found in input")
	}

	// 验证签名（简化处理）
	if len(signature) < 10 {
		return nil, fmt.Errorf("invalid signature")
	}

	// 签名验证成功，继续下一步
	input.Data["signature_verified"] = true
	input.Data["transaction_hash"] = "0x" + signature[:40] // 模拟交易哈希

	return &NodeOutput{
		Data:      input.Data,
		NextNodes: []string{"result_aggregator"},
		Completed: false,
	}, nil
}

// ResultAggregatorNode 结果整合节点
type ResultAggregatorNode struct{}

func NewResultAggregatorNode() *ResultAggregatorNode {
	return &ResultAggregatorNode{}
}

func (n *ResultAggregatorNode) GetName() string {
	return "result_aggregator"
}

func (n *ResultAggregatorNode) GetType() string {
	return "aggregator"
}

func (n *ResultAggregatorNode) Execute(ctx context.Context, input NodeInput) (*NodeOutput, error) {
	// 整合执行结果
	result := map[string]any{
		"status":     "completed",
		"message":    "所有操作已成功完成",
		"tasks":      input.Data["tasks"],
		"signatures": input.Data["signature"],
		"tx_hash":    input.Data["transaction_hash"],
	}

	if verified, ok := input.Data["signature_verified"].(bool); ok && verified {
		result["verification"] = "success"
	}

	return &NodeOutput{
		Data:      result,
		NextNodes: []string{}, // 终止节点
		Completed: true,
	}, nil
}

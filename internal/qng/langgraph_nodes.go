package qng

import (
	"context"
	"fmt"
	"log"
	"qng_agent/internal/llm"
	"strings"
	"time"
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
	log.Printf("🔄 任务分解节点开始执行")
	
	userMessage, ok := input.Data["user_message"].(string)
	if !ok {
		log.Printf("❌ 输入中缺少user_message")
		return nil, fmt.Errorf("user_message not found in input")
	}

	log.Printf("📝 用户消息: %s", userMessage)

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

	log.Printf("📋 构建LLM提示完成")
	log.Printf("📝 提示长度: %d", len(prompt))

	// 调用LLM进行任务分解
	if n.llmClient != nil {
		log.Printf("🤖 调用LLM进行任务分解...")
		response, err := n.llmClient.Chat(ctx, []llm.Message{
			{Role: "user", Content: prompt},
		})
		if err != nil {
			log.Printf("❌ LLM调用失败: %v", err)
			return nil, fmt.Errorf("LLM call failed: %w", err)
		}

		log.Printf("✅ LLM响应成功")
		log.Printf("📄 LLM响应: %s", response)

		// 解析LLM响应
		log.Printf("🔄 解析LLM响应...")
		tasks := n.parseTasksFromResponse(response)
		log.Printf("📋 解析出 %d 个任务", len(tasks))

		// 决定下一个执行节点
		nextNodes := n.determineNextNodes(tasks)
		log.Printf("➡️  下一个节点: %v", nextNodes)

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
	log.Printf("⚠️  没有LLM客户端，使用简单规则分解")
	tasks := n.simpleTaskDecomposition(userMessage)
	log.Printf("📋 简单分解出 %d 个任务", len(tasks))
	
	nextNodes := n.determineNextNodes(tasks)
	log.Printf("➡️  下一个节点: %v", nextNodes)

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
	log.Printf("🔄 解析LLM响应中的任务")
	log.Printf("📄 响应内容: %s", response)
	
	// 简化的JSON解析，实际应该使用json.Unmarshal
	if strings.Contains(strings.ToLower(response), "swap") {
		log.Printf("✅ 检测到swap任务")
		return []map[string]any{
			{
				"type":       "swap",
				"from_token": "USDT",
				"to_token":   "BTC",
				"amount":     "1000",
			},
		}
	}

	log.Printf("⚠️  未检测到具体任务")
	return []map[string]any{}
}

func (n *TaskDecomposerNode) simpleTaskDecomposition(message string) []map[string]any {
	log.Printf("🔄 使用简单规则分解任务")
	log.Printf("📝 消息: %s", message)
	
	lowerMsg := strings.ToLower(message)
	tasks := make([]map[string]any, 0)

	if strings.Contains(lowerMsg, "兑换") || strings.Contains(lowerMsg, "swap") {
		log.Printf("✅ 检测到兑换/swap任务")
		tasks = append(tasks, map[string]any{
			"type":       "swap",
			"from_token": "USDT",
			"to_token":   "BTC",
			"amount":     "1000",
		})
	}

	if strings.Contains(lowerMsg, "质押") || strings.Contains(lowerMsg, "stake") {
		log.Printf("✅ 检测到质押/stake任务")
		tasks = append(tasks, map[string]any{
			"type":   "stake",
			"token":  "BTC",
			"amount": "0.1",
			"pool":   "compound",
		})
	}

	log.Printf("📋 简单分解完成，共 %d 个任务", len(tasks))
	return tasks
}

func (n *TaskDecomposerNode) determineNextNodes(tasks []map[string]any) []string {
	log.Printf("🔄 确定下一个执行节点")
	log.Printf("📋 任务数量: %d", len(tasks))
	
	for i, task := range tasks {
		log.Printf("📋 任务[%d]: %+v", i, task)
		if taskType, ok := task["type"].(string); ok {
			log.Printf("🔄 任务类型: %s", taskType)
			switch taskType {
			case "swap":
				log.Printf("➡️  选择swap_executor节点")
				return []string{"swap_executor"}
			case "stake":
				log.Printf("➡️  选择stake_executor节点")
				return []string{"stake_executor"}
			}
		}
	}
	
	log.Printf("➡️  选择result_aggregator节点")
	return []string{"result_aggregator"}
}

// SwapExecutorNode 交易执行节点
type SwapExecutorNode struct{}

func NewSwapExecutorNode() *SwapExecutorNode {
	return &SwapExecutorNode{}
}

func (n *SwapExecutorNode) GetName() string {
	return "swap_executor"
}

func (n *SwapExecutorNode) GetType() string {
	return "transaction_executor"
}

func (n *SwapExecutorNode) Execute(ctx context.Context, input NodeInput) (*NodeOutput, error) {
	log.Printf("🔄 交易执行节点开始执行")
	
	tasks, ok := input.Data["tasks"].([]map[string]any)
	if !ok {
		log.Printf("❌ 输入中缺少tasks")
		return nil, fmt.Errorf("tasks not found in input")
	}

	log.Printf("📋 任务数量: %d", len(tasks))

	// 查找swap任务
	var swapTask map[string]any
	for i, task := range tasks {
		log.Printf("📋 检查任务[%d]: %+v", i, task)
		if taskType, ok := task["type"].(string); ok && taskType == "swap" {
			log.Printf("✅ 找到swap任务: %+v", task)
			swapTask = task
			break
		}
	}

	if swapTask == nil {
		log.Printf("⚠️  没有swap任务，跳到下一个节点")
		// 没有swap任务，跳到下一个节点
		return &NodeOutput{
			Data:      input.Data,
			NextNodes: []string{"stake_executor"},
			Completed: false,
		}, nil
	}

	// 需要用户签名授权交易
	log.Printf("✍️  需要用户签名授权交易")
	authRequest := map[string]any{
		"type":       "transaction_signature",
		"action":     "swap",
		"from_token": swapTask["from_token"],
		"to_token":   swapTask["to_token"],
		"amount":     swapTask["amount"],
		"gas_fee":    "0.001 ETH",
		"slippage":   "0.5%",
	}

	log.Printf("📋 授权请求: %+v", authRequest)

	return &NodeOutput{
		Data:         input.Data,
		NextNodes:    []string{"signature_validator"},
		NeedUserAuth: true,
		AuthRequest:  authRequest,
		Completed:    false,
	}, nil
}

// StakeExecutorNode 质押执行节点
type StakeExecutorNode struct{}

func NewStakeExecutorNode() *StakeExecutorNode {
	return &StakeExecutorNode{}
}

func (n *StakeExecutorNode) GetName() string {
	return "stake_executor"
}

func (n *StakeExecutorNode) GetType() string {
	return "transaction_executor"
}

func (n *StakeExecutorNode) Execute(ctx context.Context, input NodeInput) (*NodeOutput, error) {
	log.Printf("🔄 质押执行节点开始执行")
	
	tasks, ok := input.Data["tasks"].([]map[string]any)
	if !ok {
		log.Printf("❌ 输入中缺少tasks")
		return nil, fmt.Errorf("tasks not found in input")
	}

	log.Printf("📋 任务数量: %d", len(tasks))

	// 查找stake任务
	var stakeTask map[string]any
	for i, task := range tasks {
		log.Printf("📋 检查任务[%d]: %+v", i, task)
		if taskType, ok := task["type"].(string); ok && taskType == "stake" {
			log.Printf("✅ 找到stake任务: %+v", task)
			stakeTask = task
			break
		}
	}

	if stakeTask == nil {
		log.Printf("⚠️  没有stake任务，结束流程")
		// 没有stake任务，结束流程
		return &NodeOutput{
			Data:      input.Data,
			NextNodes: []string{"result_aggregator"},
			Completed: false,
		}, nil
	}

	// 需要用户签名授权质押
	log.Printf("✍️  需要用户签名授权质押")
	authRequest := map[string]any{
		"type":    "transaction_signature",
		"action":  "stake",
		"token":   stakeTask["token"],
		"amount":  stakeTask["amount"],
		"pool":    stakeTask["pool"],
		"gas_fee": "0.001 ETH",
		"apy":     "8.5%",
	}

	log.Printf("📋 授权请求: %+v", authRequest)

	return &NodeOutput{
		Data:         input.Data,
		NextNodes:    []string{"signature_validator"},
		NeedUserAuth: true,
		AuthRequest:  authRequest,
		Completed:    false,
	}, nil
}

// SignatureValidatorNode 签名验证节点
type SignatureValidatorNode struct{}

func NewSignatureValidatorNode() *SignatureValidatorNode {
	return &SignatureValidatorNode{}
}

func (n *SignatureValidatorNode) GetName() string {
	return "signature_validator"
}

func (n *SignatureValidatorNode) GetType() string {
	return "validator"
}

func (n *SignatureValidatorNode) Execute(ctx context.Context, input NodeInput) (*NodeOutput, error) {
	log.Printf("🔄 签名验证节点开始执行")
	
	signature, ok := input.Data["signature"].(string)
	if !ok || signature == "" {
		log.Printf("❌ 输入中缺少签名")
		return nil, fmt.Errorf("signature not found in input")
	}

	log.Printf("🔐 收到签名，长度: %d", len(signature))
	log.Printf("🔐 签名内容: %s", signature[:llm.Min(len(signature), 50)])

	// 验证签名（简化处理）
	if len(signature) < 10 {
		log.Printf("❌ 签名长度不足: %d", len(signature))
		return nil, fmt.Errorf("invalid signature")
	}

	log.Printf("✅ 签名验证成功")

	// 签名验证成功，继续下一步
	input.Data["signature_verified"] = true
	input.Data["transaction_hash"] = "0x" + signature[:40] // 模拟交易哈希

	log.Printf("📊 更新数据:")
	log.Printf("  - signature_verified: true")
	log.Printf("  - transaction_hash: %s", input.Data["transaction_hash"])

	return &NodeOutput{
		Data:      input.Data,
		NextNodes: []string{"result_aggregator"},
		Completed: false,
	}, nil
}

// ResultAggregatorNode 结果聚合节点
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
	log.Printf("🔄 结果聚合节点开始执行")
	log.Printf("📊 输入数据: %+v", input.Data)

	// 聚合所有执行结果
	result := map[string]any{
		"status":      "completed",
		"timestamp":   time.Now(),
		"workflow_id": input.Context["workflow_id"],
		"session_id":  input.Context["session_id"],
		"tasks":       input.Data["tasks"],
		"user_message": input.Data["user_message"],
	}

	// 检查是否有签名验证结果
	if signatureVerified, ok := input.Data["signature_verified"].(bool); ok && signatureVerified {
		log.Printf("✅ 检测到签名验证成功")
		result["signature_verified"] = true
		result["transaction_hash"] = input.Data["transaction_hash"]
	}

	// 检查是否有交易执行结果
	if transactionHash, ok := input.Data["transaction_hash"].(string); ok {
		log.Printf("✅ 检测到交易哈希: %s", transactionHash)
		result["transaction_hash"] = transactionHash
	}

	log.Printf("📊 聚合结果: %+v", result)

	return &NodeOutput{
		Data:      result,
		NextNodes: []string{}, // 终止节点
		Completed: true,
	}, nil
} 
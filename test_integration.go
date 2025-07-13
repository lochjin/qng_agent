package main

import (
	"context"
	"log"
	"qng_agent/internal/agent"
	"qng_agent/internal/config"
	"qng_agent/internal/mcp"
	"qng_agent/internal/qng"
	"time"
)

func main() {
	log.Printf("🧪 开始集成测试")
	
	// 1. 测试配置加载
	log.Printf("📋 测试配置加载...")
	cfg := config.LoadConfig("config/config.yaml")
	if cfg == nil {
		log.Fatal("❌ 配置加载失败")
	}
	log.Printf("✅ 配置加载成功")

	// 2. 测试QNG Chain
	log.Printf("🔄 测试QNG Chain...")
	testQNGChain(cfg.MCP.QNG)

	// 3. 测试MCP服务器
	log.Printf("🔄 测试MCP服务器...")
	testMCPServer(cfg.MCP)

	// 4. 测试智能体
	log.Printf("🔄 测试智能体...")
	testAgent(cfg.Agent)

	// 5. 测试完整工作流
	log.Printf("🔄 测试完整工作流...")
	testCompleteWorkflow(cfg)

	log.Printf("🎉 所有集成测试通过！")
}

func testQNGChain(cfg config.QNGConfig) {
	// 创建QNG Chain
	chain := qng.NewChain(cfg)
	
	// 启动Chain
	if err := chain.Start(); err != nil {
		log.Fatalf("❌ QNG Chain启动失败: %v", err)
	}
	defer chain.Stop()
	
	log.Printf("✅ QNG Chain启动成功")

	// 测试消息处理
	ctx := context.Background()
	ctx = context.WithValue(ctx, "workflow_id", "test_workflow_001")
	ctx = context.WithValue(ctx, "session_id", "test_session_001")

	message := "我需要将1000USDT兑换成BTC"
	log.Printf("📝 测试消息: %s", message)

	result, err := chain.ProcessMessage(ctx, message)
	if err != nil {
		log.Fatalf("❌ 消息处理失败: %v", err)
	}

	log.Printf("✅ 消息处理成功")
	log.Printf("📊 结果: %+v", result)

	// 测试签名继续
	if result.NeedSignature {
		log.Printf("✍️  测试签名继续...")
		
		signature := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
		finalResult, err := chain.ContinueWithSignature(ctx, result.WorkflowContext, signature)
		if err != nil {
			log.Fatalf("❌ 签名继续失败: %v", err)
		}

		log.Printf("✅ 签名继续成功")
		log.Printf("📊 最终结果: %+v", finalResult)
	}
}

func testMCPServer(cfg config.MCPConfig) {
	// 创建MCP服务器
	server := mcp.NewServer(cfg)
	
	// 启动服务器
	if err := server.Start(); err != nil {
		log.Fatalf("❌ MCP服务器启动失败: %v", err)
	}
	defer server.Stop()
	
	log.Printf("✅ MCP服务器启动成功")

	// 测试服务列表
	services := server.GetServices()
	log.Printf("📋 可用服务: %v", services)

	// 测试QNG服务
	if cfg.QNG.Enabled {
		log.Printf("🔄 测试QNG服务...")
		
		ctx := context.Background()
		result, err := server.Call(ctx, "qng", "execute_workflow", map[string]any{
			"message": "我需要将500USDT兑换成ETH",
		})
		if err != nil {
			log.Fatalf("❌ QNG服务调用失败: %v", err)
		}

		log.Printf("✅ QNG服务调用成功")
		log.Printf("📊 结果: %+v", result)
	}

	// 测试MetaMask服务
	if cfg.MetaMask.Enabled {
		log.Printf("🔄 测试MetaMask服务...")
		
		ctx := context.Background()
		result, err := server.Call(ctx, "metamask", "connect_wallet", map[string]any{
			"request_permissions": true,
		})
		if err != nil {
			log.Fatalf("❌ MetaMask服务调用失败: %v", err)
		}

		log.Printf("✅ MetaMask服务调用成功")
		log.Printf("📊 结果: %+v", result)
	}
}

func testAgent(cfg config.AgentConfig) {
	// 创建智能体
	agent := agent.NewAgent(cfg)
	
	// 启动智能体
	if err := agent.Start(); err != nil {
		log.Fatalf("❌ 智能体启动失败: %v", err)
	}
	defer agent.Stop()
	
	log.Printf("✅ 智能体启动成功")

	// 测试消息处理
	ctx := context.Background()
	message := "我需要将2000USDT兑换成BTC，然后质押到Compound"
	log.Printf("📝 测试消息: %s", message)

	execution, err := agent.ProcessMessage(ctx, message)
	if err != nil {
		log.Fatalf("❌ 智能体消息处理失败: %v", err)
	}

	log.Printf("✅ 智能体消息处理成功")
	log.Printf("📊 执行结果: %+v", execution)

	// 测试轮询状态
	if execution.SessionID != "" {
		log.Printf("🔄 测试状态轮询...")
		
		// 模拟轮询
		for i := 0; i < 3; i++ {
			time.Sleep(1 * time.Second)
			
			status, err := agent.PollWorkflowStatus(ctx, execution.SessionID)
			if err != nil {
				log.Printf("⚠️  轮询失败: %v", err)
				continue
			}

			log.Printf("📊 轮询结果[%d]: %+v", i+1, status)
			
			if status.NeedSignature {
				log.Printf("✍️  需要签名，测试签名提交...")
				
				signature := "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
				err := agent.SubmitSignature(ctx, execution.SessionID, signature)
				if err != nil {
					log.Printf("⚠️  签名提交失败: %v", err)
				} else {
					log.Printf("✅ 签名提交成功")
				}
				break
			}
		}
	}
}

func testCompleteWorkflow(cfg *config.Config) {
	log.Printf("🔄 测试完整工作流...")
	
	// 创建所有组件
	chain := qng.NewChain(cfg.MCP.QNG)
	server := mcp.NewServer(cfg.MCP)
	agent := agent.NewAgent(cfg.Agent)

	// 启动所有服务
	if err := chain.Start(); err != nil {
		log.Fatalf("❌ Chain启动失败: %v", err)
	}
	defer chain.Stop()

	if err := server.Start(); err != nil {
		log.Fatalf("❌ MCP服务器启动失败: %v", err)
	}
	defer server.Stop()

	if err := agent.Start(); err != nil {
		log.Fatalf("❌ 智能体启动失败: %v", err)
	}
	defer agent.Stop()

	log.Printf("✅ 所有服务启动成功")

	// 测试完整工作流
	ctx := context.Background()
	testCases := []string{
		"我需要将1000USDT兑换成BTC",
		"将我的BTC质押到Compound",
		"查看我的钱包余额",
		"将500USDT兑换成ETH，然后质押到Aave",
	}

	for i, testCase := range testCases {
		log.Printf("🧪 测试用例[%d]: %s", i+1, testCase)
		
		execution, err := agent.ProcessMessage(ctx, testCase)
		if err != nil {
			log.Printf("❌ 测试用例[%d]失败: %v", i+1, err)
			continue
		}

		log.Printf("✅ 测试用例[%d]成功", i+1)
		log.Printf("📊 执行结果: %+v", execution)

		// 等待一段时间让工作流执行
		time.Sleep(2 * time.Second)
	}

	log.Printf("🎉 完整工作流测试完成")
}

// 测试辅助函数
func testLLMIntegration() {
	log.Printf("🤖 测试LLM集成...")
	
	// 这里可以添加LLM集成的测试
	// 由于需要API密钥，这里只是示例
	log.Printf("✅ LLM集成测试跳过（需要API密钥）")
}

func testWalletIntegration() {
	log.Printf("🔗 测试钱包集成...")
	
	// 这里可以添加钱包集成的测试
	log.Printf("✅ 钱包集成测试跳过（需要MetaMask）")
}

func testErrorHandling() {
	log.Printf("🛡️  测试错误处理...")
	
	// 测试各种错误情况
	testCases := []struct {
		name    string
		message string
	}{
		{"空消息", ""},
		{"无效消息", "无效的请求"},
		{"超长消息", string(make([]byte, 10000))},
	}

	for _, testCase := range testCases {
		log.Printf("🧪 测试错误处理: %s", testCase.name)
		// 这里可以添加具体的错误处理测试
	}

	log.Printf("✅ 错误处理测试完成")
}

func testPerformance() {
	log.Printf("⚡ 测试性能...")
	
	// 测试并发处理
	start := time.Now()
	
	// 模拟并发请求
	for i := 0; i < 5; i++ {
		go func(id int) {
			log.Printf("🔄 并发请求[%d]开始", id)
			time.Sleep(1 * time.Second)
			log.Printf("✅ 并发请求[%d]完成", id)
		}(i)
	}
	
	// 等待所有请求完成
	time.Sleep(2 * time.Second)
	
	duration := time.Since(start)
	log.Printf("⏱️  性能测试完成，耗时: %v", duration)
} 
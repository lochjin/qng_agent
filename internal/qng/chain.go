package qng

import (
	"context"
	"fmt"
	"log"
	"qng_agent/internal/config"
	"qng_agent/internal/llm"
	"qng_agent/internal/contracts"
	"qng_agent/internal/rpc"
	"sync"
)

type Chain struct {
	config          config.QNGConfig
	llmClient       llm.Client
	contractManager *contracts.ContractManager
	rpcClient       *rpc.Client
	langGraph       *LangGraph
	mu              sync.RWMutex
	running         bool
}

type ProcessResult struct {
	NeedSignature    bool `json:"need_signature"`
	SignatureRequest any  `json:"signature_request,omitempty"`
	WorkflowContext  any  `json:"workflow_context,omitempty"`
	FinalResult      any  `json:"final_result,omitempty"`
}

func NewChain(config config.QNGConfig) *Chain {
	// 创建LLM客户端
	var llmClient llm.Client
	var err error
	
	// 从配置中获取LLM配置
	if config.Chain.LLM.Provider != "" {
		llmClient, err = llm.NewClient(config.Chain.LLM)
		if err != nil {
			log.Printf("⚠️  无法创建LLM客户端: %v", err)
			llmClient = nil
		}
	}

	// 创建合约管理器
	contractManager, err := contracts.NewContractManager("config/contracts.json")
	if err != nil {
		log.Printf("⚠️  无法创建合约管理器: %v", err)
		contractManager = nil
	}

	// 创建RPC客户端
	var rpcClient *rpc.Client
	if config.Chain.RPCURL != "" {
		rpcClient = rpc.NewClient(config.Chain.RPCURL)
		log.Printf("✅ RPC客户端已创建: %s", config.Chain.RPCURL)
	} else {
		log.Printf("⚠️  未配置RPC URL，使用模拟确认")
	}

	// 创建LangGraph
	langGraph := NewLangGraph(llmClient, contractManager, rpcClient, config.Chain.Transaction)

	chain := &Chain{
		config:          config,
		llmClient:       llmClient,
		contractManager: contractManager,
		rpcClient:       rpcClient,
		langGraph:       langGraph,
	}

	return chain
}

func (c *Chain) Start() error {
	log.Printf("🚀 QNG Chain启动")
	c.running = true
	return nil
}

func (c *Chain) Stop() error {
	log.Printf("🛑 QNG Chain停止")
	c.running = false
	return nil
}

func (c *Chain) ProcessMessage(ctx context.Context, message string) (*ProcessResult, error) {
	log.Printf("🔄 QNG Chain开始处理消息")
	log.Printf("📝 消息内容: %s", message)
	
	if !c.running {
		log.Printf("❌ Chain未运行")
		return nil, fmt.Errorf("chain is not running")
	}

	// 使用LangGraph执行工作流
	result, err := c.langGraph.ExecuteWorkflow(ctx, message)
	if err != nil {
		log.Printf("❌ LangGraph执行失败: %v", err)
		return nil, fmt.Errorf("langgraph execution failed: %w", err)
	}

	log.Printf("✅ LangGraph执行成功")
	return result, nil
}

func (c *Chain) ContinueWithSignature(ctx context.Context, workflowContext any, signature string) (*ProcessResult, error) {
	log.Printf("🔄 QNG Chain使用签名继续工作流")
	log.Printf("🔐 签名长度: %d", len(signature))
	
	result, err := c.langGraph.ContinueWithSignature(ctx, workflowContext, signature)
	if err != nil {
		log.Printf("❌ 继续执行失败: %v", err)
		return nil, fmt.Errorf("continue with signature failed: %w", err)
	}

	log.Printf("✅ 继续执行成功")
	return result, nil
}

package main

import (
	"log"
	"qng_agent/internal/agent"
	"qng_agent/internal/config"
	"qng_agent/internal/mcp"
	"qng_agent/internal/ui"
)

func main() {
	log.Println("=== QNG Agent 系统启动 ===")

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	log.Println("✅ 配置加载成功")

	// 初始化本地MCP Server管理器
	mcpManager := mcp.NewManager(cfg.MCP)
	log.Println("✅ MCP管理器初始化成功")

	// 注册QNG MCP Server
	qngServer := mcp.NewQNGServer(cfg.QNG)
	mcpManager.RegisterServer("qng", qngServer)
	log.Println("✅ QNG MCP Server 已注册")

	// 注册MetaMask MCP Server
	metamaskServer := mcp.NewMetaMaskServer(cfg.MetaMask)
	mcpManager.RegisterServer("metamask", metamaskServer)
	log.Println("✅ MetaMask MCP Server 已注册")

	// 初始化Agent管理器
	agentManager := agent.NewManager(mcpManager, cfg.LLM)
	log.Println("✅ Agent管理器初始化成功")

	// 启动UI服务器
	uiServer := ui.NewServer(agentManager, cfg.UI)
	log.Println("✅ UI服务器初始化成功")

	log.Println("🚀 启动QNG Agent系统...")
	if err := uiServer.Start(); err != nil {
		log.Fatal("Failed to start UI server:", err)
	}
}

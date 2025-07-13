package main

import (
	"log"
	"qng_agent/internal/agent"
	"qng_agent/internal/config"
	"qng_agent/internal/mcp"
	"qng_agent/internal/ui"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// 初始化MCP Server管理器
	mcpManager := mcp.NewManager(cfg.MCP)

	// 注册QNG MCP Server
	qngServer := mcp.NewQNGServer(cfg.QNG)
	mcpManager.RegisterServer("qng", qngServer)

	// 注册MetaMask MCP Server
	metamaskServer := mcp.NewMetaMaskServer(cfg.MetaMask)
	mcpManager.RegisterServer("metamask", metamaskServer)

	// 初始化Agent管理器
	agentManager := agent.NewManager(mcpManager, cfg.LLM)

	// 启动UI服务器
	uiServer := ui.NewServer(agentManager, cfg.UI)

	log.Println("Starting QNG Agent System...")
	if err := uiServer.Start(); err != nil {
		log.Fatal("Failed to start UI server:", err)
	}
}

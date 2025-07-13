package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"qng_agent/internal/config"
	"qng_agent/internal/mcp"
	"qng_agent/internal/service"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("=== MCP 服务器启动 ===")

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// 获取服务注册中心
	registry := service.GetRegistry()

	// 注册自己为MCP服务
	mcpService := &service.ServiceInfo{
		Name:    "mcp",
		Address: "localhost",
		Port:    9091,
		Endpoints: []string{
			"/api/mcp/call",
			"/api/mcp/capabilities",
			"/api/mcp/servers",
			"/health",
		},
		Metadata: map[string]string{
			"type":    "mcp_manager",
			"version": "1.0.0",
		},
	}

	if err := registry.RegisterService(mcpService); err != nil {
		log.Fatal("Failed to register MCP service:", err)
	}

	// 初始化MCP Server管理器
	mcpManager := mcp.NewManager(cfg.MCP)

	// 注册QNG MCP Server
	qngServer := mcp.NewQNGServer(cfg.QNG)
	mcpManager.RegisterServer("qng", qngServer)
	log.Println("✅ QNG MCP Server 已注册")

	// 注册MetaMask MCP Server
	metamaskServer := mcp.NewMetaMaskServer(cfg.MetaMask)
	mcpManager.RegisterServer("metamask", metamaskServer)
	log.Println("✅ MetaMask MCP Server 已注册")

	// 创建HTTP服务器
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// 健康检查端点
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "mcp",
			"timestamp": time.Now().Unix(),
		})
	})

	// MCP API端点
	api := router.Group("/api/mcp")
	{
		// 调用MCP工具
		api.POST("/call", func(c *gin.Context) {
			var req struct {
				Server string                 `json:"server"`
				Method string                 `json:"method"`
				Params map[string]interface{} `json:"params"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			ctx := context.Background()
			result, err := mcpManager.CallTool(ctx, req.Server, req.Method, req.Params)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"result": result})
		})

		// 获取所有能力
		api.GET("/capabilities", func(c *gin.Context) {
			capabilities := mcpManager.GetAllCapabilities()
			c.JSON(http.StatusOK, gin.H{"capabilities": capabilities})
		})

		// 获取服务器列表
		api.GET("/servers", func(c *gin.Context) {
			servers := make(map[string]interface{})
			capabilities := mcpManager.GetAllCapabilities()

			for name, caps := range capabilities {
				servers[name] = map[string]interface{}{
					"name":         name,
					"capabilities": caps,
					"status":       "running",
				}
			}

			c.JSON(http.StatusOK, gin.H{"servers": servers})
		})

		// QNG工作流相关端点
		api.POST("/qng/workflow", func(c *gin.Context) {
			var req struct {
				Message string `json:"message"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			ctx := context.Background()
			workflowID, err := mcpManager.CallQNGWorkflow(ctx, req.Message)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"workflow_id": workflowID})
		})

		api.GET("/qng/workflow/:id/status", func(c *gin.Context) {
			workflowID := c.Param("id")

			ctx := context.Background()
			status, err := mcpManager.GetQNGWorkflowStatus(ctx, workflowID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, status)
		})

		api.POST("/qng/workflow/:id/signature", func(c *gin.Context) {
			workflowID := c.Param("id")

			var req struct {
				Signature string `json:"signature"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			ctx := context.Background()
			result, err := mcpManager.SubmitWorkflowSignature(ctx, workflowID, req.Signature)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"result": result})
		})
	}

	// 启动HTTP服务器
	server := &http.Server{
		Addr:    ":9091",
		Handler: router,
	}

	go func() {
		log.Printf("🚀 MCP服务启动在端口: %d", 9091)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start MCP server:", err)
		}
	}()

	log.Println("✅ MCP服务器已启动")
	log.Println("📋 已注册的服务器:")
	capabilities := mcpManager.GetAllCapabilities()
	for name, caps := range capabilities {
		log.Printf("  - %s: %d 个能力", name, len(caps))
		for _, cap := range caps {
			capJSON, _ := json.MarshalIndent(cap, "    ", "  ")
			log.Printf("    %s", capJSON)
		}
	}

	// 启动健康检查
	registry.StartHealthCheck()

	// 等待中断信号
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	log.Println("MCP服务器正在运行，按 Ctrl+C 停止")
	<-c

	log.Println("正在关闭MCP服务器...")

	// 注销服务
	registry.UnregisterService("mcp")

	// 关闭MCP管理器
	mcpManager.Stop()

	// 关闭HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("MCP服务关闭失败: %v", err)
	}

	log.Println("MCP服务器已关闭")
}

package main

import (
	"log"
	"net/http"
	"qng_agent/internal/config"
	"qng_agent/internal/mcp"
	"qng_agent/internal/service"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("=== QNG MCP 服务启动 ===")

	// 加载配置
	cfg := config.LoadConfig("config/config.yaml")
	if cfg == nil {
		log.Fatal("Failed to load config")
	}

	// 获取服务注册中心
	registry := service.GetRegistry()

	// 注册MCP服务
	mcpService := &service.ServiceInfo{
		Name:    "mcp",
		Address: "localhost",
		Port:    9091,
		Status:  "running",
		LastSeen: time.Now(),
		Endpoints: []string{
			"/api/mcp/call",
			"/api/mcp/qng/workflow",
			"/api/mcp/capabilities",
		},
		Metadata: map[string]string{
			"type":    "mcp_service",
			"version": "1.0.0",
		},
	}

	if err := registry.RegisterService(mcpService); err != nil {
		log.Fatal("Failed to register MCP service:", err)
	}

	log.Println("✅ MCP服务已注册到服务注册中心")

	// 初始化MCP服务器
	mcpServer := mcp.NewServer(cfg.MCP)
	log.Println("✅ MCP服务器初始化成功")

	// 启动MCP服务器
	if err := mcpServer.Start(); err != nil {
		log.Fatal("Failed to start MCP server:", err)
	}
	defer mcpServer.Stop()

	log.Println("📋 服务架构说明:")
	log.Println("  - MCP服务管理所有子服务")
	log.Println("  - QNG服务内部包含chain功能")
	log.Println("  - 不需要独立等待chain服务")

	// 创建HTTP服务器
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// 添加CORS中间件
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// 健康检查端点
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "mcp",
			"timestamp": time.Now().Unix(),
		})
	})

	// API路由
	api := router.Group("/api/mcp")
	{
		// 通用MCP调用
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

			ctx := c.Request.Context()
			result, err := mcpServer.Call(ctx, req.Server, req.Method, req.Params)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"result": result})
		})

		// QNG工作流
		api.POST("/qng/workflow", func(c *gin.Context) {
			var req struct {
				Message string `json:"message"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			ctx := c.Request.Context()
			result, err := mcpServer.Call(ctx, "qng", "execute_workflow", map[string]any{
				"message": req.Message,
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			resMap, _ := result.(map[string]any)
			workflowID, _ := resMap["workflow_id"].(string)
			c.JSON(http.StatusOK, gin.H{"workflow_id": workflowID})
		})

		// 获取工作流状态
		api.GET("/qng/workflow/:id/status", func(c *gin.Context) {
			workflowID := c.Param("id")

			ctx := c.Request.Context()
			result, err := mcpServer.Call(ctx, "qng", "get_session_status", map[string]any{
				"session_id": workflowID,
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, result)
		})

		// 提交签名
		api.POST("/qng/workflow/:id/signature", func(c *gin.Context) {
			workflowID := c.Param("id")

			var req struct {
				Signature string `json:"signature"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			ctx := c.Request.Context()
			result, err := mcpServer.Call(ctx, "qng", "submit_signature", map[string]any{
				"session_id": workflowID,
				"signature":  req.Signature,
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"result": result})
		})

		// 获取能力
		api.GET("/capabilities", func(c *gin.Context) {
			capabilities := mcpServer.GetCapabilities()
			c.JSON(http.StatusOK, gin.H{"capabilities": capabilities})
		})
	}

	// 启动服务器
	addr := ":9091"
	log.Printf("MCP服务启动在 %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatal("Failed to start MCP server:", err)
	}
}

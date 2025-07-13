package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"qng_agent/internal/agent"
	"qng_agent/internal/config"
	"qng_agent/internal/mcp"
	"qng_agent/internal/service"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocket相关结构
type WebSocketClient struct {
	SessionID string
	Conn      *websocket.Conn
	Send      chan []byte
}

type ChatMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

type ChatResponse struct {
	Type       string `json:"type"`
	SessionID  string `json:"session_id"`
	Response   string `json:"response"`
	NeedAction bool   `json:"need_action"`
	ActionType string `json:"action_type,omitempty"`
	ActionData any    `json:"action_data,omitempty"`
	WorkflowID string `json:"workflow_id,omitempty"`
	Timestamp  int64  `json:"timestamp"`
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 允许跨域
		},
	}
	clients = make(map[string]*WebSocketClient)
)

func main() {
	log.Println("=== QNG Agent 管理器启动 ===")

	// 加载配置
	cfg := config.LoadConfig("config/config.yaml")
	if cfg == nil {
		log.Fatal("Failed to load config")
	}

	// 获取服务注册中心
	registry := service.GetRegistry()

	// 注册MCP服务到注册中心（如果不存在）
	mcpService := &service.ServiceInfo{
		Name:    "mcp",
		Address: "localhost",
		Port:    9091, // MCP服务端口
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

	// 尝试注册MCP服务
	if err := registry.RegisterService(mcpService); err != nil {
		log.Printf("Warning: Failed to register MCP service: %v", err)
	} else {
		log.Println("✅ MCP服务已注册到服务注册中心")
	}

	// 注册自己为Agent服务
	agentService := &service.ServiceInfo{
		Name:    "agent",
		Address: "localhost",
		Port:    9090,
		Endpoints: []string{
			"/api/chat",
			"/api/workflow/:id/status",
			"/api/workflow/:id/signature",
			"/api/capabilities",
			"/ws",
			"/health",
		},
		Metadata: map[string]string{
			"type":    "agent_manager",
			"version": "1.0.0",
		},
	}

	if err := registry.RegisterService(agentService); err != nil {
		log.Fatal("Failed to register agent service:", err)
	}

	// 等待依赖服务启动
	log.Println("⏳ 等待依赖服务启动...")
	log.Println("📋 服务依赖说明:")
	log.Println("  - Agent服务依赖MCP服务")
	log.Println("  - MCP服务内部管理QNG和MetaMask服务")
	log.Println("  - Chain功能由QNG服务提供")
	waitForServices([]string{"mcp"}, registry, 30*time.Second)

	// 创建MCP服务器
	mcpServer := mcp.NewServer(cfg.MCP)

	// 初始化Agent管理器
	agentManager := agent.NewManager(mcpServer, cfg.LLM)

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
			"service":   "agent",
			"timestamp": time.Now().Unix(),
		})
	})

	// 手动设置路由
	// 静态文件服务
	router.Static("/static", cfg.Frontend.BuildDir)
	router.StaticFile("/", cfg.Frontend.BuildDir+"/index.html")

	// WebSocket路由
	router.GET("/ws", func(c *gin.Context) {
		handleWebSocket(c, agentManager)
	})

	// API路由
	api := router.Group("/api")
	{
		api.GET("/capabilities", func(c *gin.Context) {
			capabilities := agentManager.GetCapabilities()
			c.JSON(http.StatusOK, gin.H{
				"capabilities": capabilities,
			})
		})

		// 前端期望的API端点
		api.POST("/agent/process", func(c *gin.Context) {
			var msg struct {
				Message string `json:"message"`
			}

			if err := c.ShouldBindJSON(&msg); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			ctx := context.Background()
			req := agent.ProcessRequest{
				SessionID: uuid.New().String(),
				Message:   msg.Message,
			}

			response, err := agentManager.ProcessMessage(ctx, req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, response)
		})

		api.GET("/agent/poll/:sessionId", func(c *gin.Context) {
			sessionId := c.Param("sessionId")

			ctx := context.Background()
			status, err := agentManager.GetWorkflowStatus(ctx, sessionId)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, status)
		})

		api.POST("/agent/signature", func(c *gin.Context) {
			var req struct {
				SessionID string `json:"session_id"`
				Signature string `json:"signature"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			ctx := context.Background()
			result, err := agentManager.ContinueWorkflowWithSignature(ctx, req.SessionID, req.Signature)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"status":      "signature_submitted",
				"session_id":  req.SessionID,
				"signature":   req.Signature,
				"result":      result,
			})
		})

		api.POST("/chat", func(c *gin.Context) {
			var msg struct {
				SessionID string `json:"session_id"`
				Message   string `json:"message"`
			}

			if err := c.ShouldBindJSON(&msg); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			ctx := context.Background()
			req := agent.ProcessRequest{
				SessionID: msg.SessionID,
				Message:   msg.Message,
			}

			response, err := agentManager.ProcessMessage(ctx, req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, response)
		})

		api.GET("/workflow/:id/status", func(c *gin.Context) {
			workflowID := c.Param("id")

			ctx := context.Background()
			status, err := agentManager.GetWorkflowStatus(ctx, workflowID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, status)
		})

		api.POST("/workflow/:id/signature", func(c *gin.Context) {
			workflowID := c.Param("id")

			var req struct {
				Signature string `json:"signature"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			ctx := context.Background()
			result, err := agentManager.ContinueWorkflowWithSignature(ctx, workflowID, req.Signature)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"status":      "signature_submitted",
				"workflow_id": workflowID,
				"signature":   req.Signature,
				"result":      result,
			})

			// 通知所有连接的客户端工作流状态更新
			broadcastWorkflowUpdate(workflowID, "signature_received", 60, "签名已提交，继续执行工作流...")
		})
	}

	// 启动HTTP服务器
	server := &http.Server{
		Addr:    ":9090",
		Handler: router,
	}

	go func() {
		log.Printf("🚀 Agent服务启动在端口: %d", 9090)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start agent server:", err)
		}
	}()

	// 启动健康检查
	registry.StartHealthCheck()

	// 等待中断信号
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	log.Println("Agent服务正在运行，按 Ctrl+C 停止")
	<-c

	log.Println("正在关闭Agent服务...")

	// 注销服务
	registry.UnregisterService("agent")

	// 关闭HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Agent服务关闭失败: %v", err)
	}

	log.Println("Agent服务已关闭")
}

// WebSocket处理函数
func handleWebSocket(c *gin.Context, agentManager *agent.Manager) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v\n", err)
		return
	}

	sessionID := uuid.New().String()
	client := &WebSocketClient{
		SessionID: sessionID,
		Conn:      conn,
		Send:      make(chan []byte, 256),
	}

	clients[sessionID] = client

	// 启动WebSocket处理goroutine
	go handleWebSocketClient(client, agentManager)
	go writeWebSocketClient(client)
}

func handleWebSocketClient(client *WebSocketClient, agentManager *agent.Manager) {
	defer func() {
		delete(clients, client.SessionID)
		client.Conn.Close()
	}()

	for {
		var msg ChatMessage
		if err := client.Conn.ReadJSON(&msg); err != nil {
			log.Printf("WebSocket read error: %v\n", err)
			break
		}

		msg.SessionID = client.SessionID

		// 处理消息
		req := agent.ProcessRequest{
			SessionID: msg.SessionID,
			Message:   msg.Message,
		}

		ctx := context.Background()
		response, err := agentManager.ProcessMessage(ctx, req)
		if err != nil {
			log.Printf("Agent process error: %v\n", err)
			continue
		}

		chatResponse := ChatResponse{
			Type:       "chat_response",
			SessionID:  msg.SessionID,
			Response:   response.Response,
			NeedAction: response.NeedAction,
			ActionType: response.ActionType,
			ActionData: response.ActionData,
			WorkflowID: response.WorkflowID,
			Timestamp:  time.Now().Unix(),
		}

		// 发送响应
		if err := client.Conn.WriteJSON(chatResponse); err != nil {
			log.Printf("WebSocket write error: %v\n", err)
			break
		}

		// 如果是工作流执行，启动状态监控
		if response.NeedAction && response.WorkflowID != "" {
			go monitorWorkflow(client, response.WorkflowID, agentManager)
		}
	}
}

func writeWebSocketClient(client *WebSocketClient) {
	defer client.Conn.Close()

	for {
		select {
		case message, ok := <-client.Send:
			if !ok {
				return
			}

			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error: %v\n", err)
				return
			}
		}
	}
}

func monitorWorkflow(client *WebSocketClient, workflowID string, agentManager *agent.Manager) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx := context.Background()
			status, err := agentManager.GetWorkflowStatus(ctx, workflowID)
			if err != nil {
				log.Printf("Get workflow status error: %v\n", err)
				continue
			}

			statusUpdate := map[string]any{
				"type":        "workflow_status",
				"workflow_id": workflowID,
				"status":      status.Status,
				"progress":    status.Progress,
				"message":     status.Message,
				"timestamp":   time.Now().Unix(),
			}

			if err := client.Conn.WriteJSON(statusUpdate); err != nil {
				log.Printf("WebSocket write error: %v\n", err)
				return
			}

			// 如果工作流完成或失败，停止监控
			if status.Status == "completed" || status.Status == "failed" || status.Status == "cancelled" {
				return
			}
		}
	}
}

func broadcastWorkflowUpdate(workflowID, status string, progress int, message string) {
	statusUpdate := map[string]any{
		"type":        "workflow_status",
		"workflow_id": workflowID,
		"status":      status,
		"progress":    progress,
		"message":     message,
		"timestamp":   time.Now().Unix(),
	}

	// 向所有连接的客户端广播状态更新
	for _, client := range clients {
		// 直接通过 WebSocket 发送 JSON
		if err := client.Conn.WriteJSON(statusUpdate); err != nil {
			log.Printf("WebSocket broadcast error: %v\n", err)
		}
	}
}

// waitForServices 等待依赖服务启动
func waitForServices(services []string, registry *service.ServiceRegistry, timeout time.Duration) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		allReady := true

		for _, serviceName := range services {
			if _, err := registry.GetService(serviceName); err != nil {
				allReady = false
				log.Printf("⏳ 等待服务: %s", serviceName)
				break
			}
		}

		if allReady {
			log.Println("✅ 所有依赖服务已就绪")
			return
		}

		time.Sleep(2 * time.Second)
	}

	log.Println("⚠️ 部分依赖服务未就绪，继续启动...")
	log.Println("📋 注意: chain服务由mcp服务内部管理，不需要独立等待")
}

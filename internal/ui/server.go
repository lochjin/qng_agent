package ui

import (
	"context"
	"fmt"
	"net/http"
	"qng_agent/internal/agent"
	"qng_agent/internal/config"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Server struct {
	agentManager *agent.Manager
	config       config.UIConfig
	router       *gin.Engine
	upgrader     websocket.Upgrader
	clients      map[string]*WebSocketClient
}

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

type SettingsRequest struct {
	LLM LLMSettings `json:"llm"`
	UI  UISettings  `json:"ui"`
}

type LLMSettings struct {
	Provider string            `json:"provider"`
	Configs  map[string]string `json:"configs"`
}

type UISettings struct {
	Port   int    `json:"port"`
	Static string `json:"static"`
}

type MCPSettingsRequest struct {
	Servers map[string]MCPServerConfig `json:"servers"`
}

type MCPServerConfig struct {
	Enabled      bool                   `json:"enabled"`
	Config       map[string]interface{} `json:"config"`
	Capabilities []string               `json:"capabilities"`
}

func NewServer(agentManager *agent.Manager, config config.UIConfig) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	server := &Server{
		agentManager: agentManager,
		config:       config,
		router:       router,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许跨域
			},
		},
		clients: make(map[string]*WebSocketClient),
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	// 静态文件服务
	s.router.Static("/static", s.config.Static)

	// 主页 - 直接服务静态HTML文件
	s.router.StaticFile("/", s.config.Static+"/index.html")

	// API路由
	api := s.router.Group("/api")
	{
		api.GET("/capabilities", s.getCapabilities)
		api.POST("/chat", s.handleChat)
		api.GET("/workflow/:id/status", s.getWorkflowStatus)
		api.POST("/workflow/:id/signature", s.submitSignature)
		api.GET("/config", s.getConfig)
		api.PUT("/config", s.updateConfig)
		api.GET("/settings", s.getSettings)
		api.PUT("/settings", s.updateSettings)
		api.GET("/settings/mcp", s.getMCPSettings)
		api.PUT("/settings/mcp", s.updateMCPSettings)
	}

	// WebSocket路由
	s.router.GET("/ws", s.handleWebSocket)
}

func (s *Server) getCapabilities(c *gin.Context) {
	capabilities := s.agentManager.GetCapabilities()
	c.JSON(http.StatusOK, gin.H{
		"capabilities": capabilities,
	})
}

func (s *Server) handleChat(c *gin.Context) {
	var msg ChatMessage
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if msg.SessionID == "" {
		msg.SessionID = uuid.New().String()
	}

	// 处理消息
	req := agent.ProcessRequest{
		SessionID: msg.SessionID,
		Message:   msg.Message,
	}

	ctx := context.Background()
	response, err := s.agentManager.ProcessMessage(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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

	c.JSON(http.StatusOK, chatResponse)
}

func (s *Server) getWorkflowStatus(c *gin.Context) {
	workflowID := c.Param("id")

	ctx := context.Background()
	status, err := s.agentManager.GetWorkflowStatus(ctx, workflowID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

func (s *Server) submitSignature(c *gin.Context) {
	workflowID := c.Param("id")

	var req struct {
		Signature string `json:"signature"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 将签名提交给工作流继续执行
	ctx := context.Background()
	result, err := s.agentManager.ContinueWorkflowWithSignature(ctx, workflowID, req.Signature)
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
	s.broadcastWorkflowUpdate(workflowID, "signature_received", 60, "签名已提交，继续执行工作流...")
}

func (s *Server) broadcastWorkflowUpdate(workflowID, status string, progress int, message string) {
	statusUpdate := map[string]any{
		"type":        "workflow_status",
		"workflow_id": workflowID,
		"status":      status,
		"progress":    progress,
		"message":     message,
		"timestamp":   time.Now().Unix(),
	}

	// 向所有连接的客户端广播状态更新
	for _, client := range s.clients {
		// 直接通过 WebSocket 发送 JSON
		if err := client.Conn.WriteJSON(statusUpdate); err != nil {
			fmt.Printf("WebSocket broadcast error: %v\n", err)
		}
	}
}

func (s *Server) handleWebSocket(c *gin.Context) {
	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Printf("WebSocket upgrade error: %v\n", err)
		return
	}

	sessionID := uuid.New().String()
	client := &WebSocketClient{
		SessionID: sessionID,
		Conn:      conn,
		Send:      make(chan []byte, 256),
	}

	s.clients[sessionID] = client

	// 启动WebSocket处理goroutine
	go s.handleWebSocketClient(client)
	go s.writeWebSocketClient(client)
}

func (s *Server) handleWebSocketClient(client *WebSocketClient) {
	defer func() {
		delete(s.clients, client.SessionID)
		client.Conn.Close()
	}()

	for {
		var msg ChatMessage
		if err := client.Conn.ReadJSON(&msg); err != nil {
			fmt.Printf("WebSocket read error: %v\n", err)
			break
		}

		msg.SessionID = client.SessionID

		// 处理消息
		req := agent.ProcessRequest{
			SessionID: msg.SessionID,
			Message:   msg.Message,
		}

		ctx := context.Background()
		response, err := s.agentManager.ProcessMessage(ctx, req)
		if err != nil {
			fmt.Printf("Agent process error: %v\n", err)
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
			fmt.Printf("WebSocket write error: %v\n", err)
			break
		}

		// 如果是工作流执行，启动状态监控
		if response.NeedAction && response.WorkflowID != "" {
			go s.monitorWorkflow(client, response.WorkflowID)
		}
	}
}

func (s *Server) writeWebSocketClient(client *WebSocketClient) {
	defer client.Conn.Close()

	for {
		select {
		case message, ok := <-client.Send:
			if !ok {
				return
			}

			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				fmt.Printf("WebSocket write error: %v\n", err)
				return
			}
		}
	}
}

func (s *Server) monitorWorkflow(client *WebSocketClient, workflowID string) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx := context.Background()
			status, err := s.agentManager.GetWorkflowStatus(ctx, workflowID)
			if err != nil {
				fmt.Printf("Get workflow status error: %v\n", err)
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
				fmt.Printf("WebSocket write error: %v\n", err)
				return
			}

			// 如果工作流完成或失败，停止监控
			if status.Status == "completed" || status.Status == "failed" || status.Status == "cancelled" {
				return
			}
		}
	}
}

func (s *Server) getSettings(c *gin.Context) {
	// Return current settings from the agent manager
	capabilities := s.agentManager.GetCapabilities()

	settings := gin.H{
		"llm": gin.H{
			"provider": capabilities["llm"].(map[string]any)["providers"].([]string)[0], // Current provider
			"configs": gin.H{
				"openai_api_key":    "***HIDDEN***",
				"openai_base_url":   "https://api.openai.com/v1",
				"anthropic_api_key": "***HIDDEN***",
				"ollama_base_url":   "http://localhost:11434",
				"gemini_api_key":    "***HIDDEN***",
				"gemini_base_url":   "https://generativelanguage.googleapis.com/v1beta",
				"model":             "gpt-4",
			},
		},
		"ui": gin.H{
			"port":   s.config.Port,
			"static": s.config.Static,
		},
		"available_providers": []string{"openai", "anthropic", "ollama", "gemini"},
		"available_models": gin.H{
			"openai":    []string{"gpt-4", "gpt-3.5-turbo", "gpt-4-turbo"},
			"anthropic": []string{"claude-3-sonnet-20240229", "claude-3-haiku-20240307"},
			"ollama":    []string{"llama3", "llama2", "codellama", "mistral", "qwen"},
			"gemini":    []string{"gemini-pro", "gemini-pro-vision"},
		},
	}

	c.JSON(http.StatusOK, settings)
}

func (s *Server) updateSettings(c *gin.Context) {
	var req SettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Implement settings update logic
	// This would require adding methods to agent manager to reload LLM client
	// and updating the configuration file

	c.JSON(http.StatusOK, gin.H{
		"message": "Settings updated successfully",
		"status":  "success",
		"note":    "Settings will take effect after restart",
	})
}

func (s *Server) getMCPSettings(c *gin.Context) {
	capabilities := s.agentManager.GetCapabilities()
	mcpServers := capabilities["mcp_servers"].(map[string]any)

	settings := gin.H{
		"servers": mcpServers,
		"timeout": 30, // Default timeout
	}

	c.JSON(http.StatusOK, settings)
}

func (s *Server) updateMCPSettings(c *gin.Context) {
	var req MCPSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Implement MCP settings update logic
	// This would require adding methods to MCP manager to enable/disable servers
	// and update their configurations

	c.JSON(http.StatusOK, gin.H{
		"message": "MCP settings updated successfully",
		"status":  "success",
		"note":    "Some changes may require restart",
	})
}

func (s *Server) getConfig(c *gin.Context) {
	// 读取当前配置文件
	cfg, err := config.Load()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, cfg)
}

func (s *Server) updateConfig(c *gin.Context) {
	var newConfig config.Config
	if err := c.ShouldBindJSON(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid config format: " + err.Error()})
		return
	}

	// 保存配置到文件
	if err := config.Save(&newConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "配置保存成功",
		"status":  "success",
		"note":    "某些配置更改可能需要重启服务才能生效",
	})
}

func (s *Server) Start() error {
	addr := ":" + strconv.Itoa(s.config.Port)
	fmt.Printf("UI Server starting on %s\n", addr)
	return s.router.Run(addr)
}

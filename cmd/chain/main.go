package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"qng_agent/internal/config"
	"qng_agent/internal/qng"
	"qng_agent/internal/service"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("=== QNG Chain 服务启动 ===")

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// 获取服务注册中心
	registry := service.GetRegistry()

	// 注册自己为Chain服务
	chainService := &service.ServiceInfo{
		Name:    "chain",
		Address: "localhost",
		Port:    9092,
		Endpoints: []string{
			"/api/chain/process",
			"/api/chain/status",
			"/api/chain/nodes",
			"/health",
		},
		Metadata: map[string]string{
			"type":    "qng_chain",
			"version": "1.0.0",
		},
	}

	if err := registry.RegisterService(chainService); err != nil {
		log.Fatal("Failed to register Chain service:", err)
	}

	// 初始化QNG Chain
	chain := qng.NewChain(cfg.QNG)
	log.Printf("🔗 初始化QNG链，RPC: %s", cfg.QNG.ChainRPC)

	// 启动Chain服务
	if err := chain.Start(); err != nil {
		log.Fatal("Failed to start chain:", err)
	}
	log.Println("✅ QNG Chain已启动")

	// 创建HTTP服务器
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// 健康检查端点
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "chain",
			"timestamp": time.Now().Unix(),
		})
	})

	// Chain API端点
	api := router.Group("/api/chain")
	{
		// 处理消息
		api.POST("/process", func(c *gin.Context) {
			var req struct {
				Message string `json:"message"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			ctx := context.Background()
			result, err := chain.ProcessMessage(ctx, req.Message)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"result": result})
		})

		// 获取链状态
		api.GET("/status", func(c *gin.Context) {
			status := map[string]interface{}{
				"running":       true,
				"chain_rpc":     cfg.QNG.ChainRPC,
				"graph_nodes":   cfg.QNG.GraphNodes,
				"poll_interval": cfg.QNG.PollInterval,
				"timestamp":     time.Now().Unix(),
			}

			c.JSON(http.StatusOK, gin.H{"status": status})
		})

		// 获取节点信息
		api.GET("/nodes", func(c *gin.Context) {
			nodes := map[string]interface{}{
				"task_decomposer": map[string]interface{}{
					"name":   "task_decomposer",
					"type":   "llm_processor",
					"status": "active",
				},
				"swap_executor": map[string]interface{}{
					"name":   "swap_executor",
					"type":   "transaction_executor",
					"status": "active",
				},
				"stake_executor": map[string]interface{}{
					"name":   "stake_executor",
					"type":   "transaction_executor",
					"status": "active",
				},
				"signature_validator": map[string]interface{}{
					"name":   "signature_validator",
					"type":   "validator",
					"status": "active",
				},
				"result_aggregator": map[string]interface{}{
					"name":   "result_aggregator",
					"type":   "aggregator",
					"status": "active",
				},
			}

			c.JSON(http.StatusOK, gin.H{"nodes": nodes})
		})

		// 继续工作流（带签名）
		api.POST("/continue", func(c *gin.Context) {
			var req struct {
				WorkflowContext interface{} `json:"workflow_context"`
				Signature       string      `json:"signature"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			ctx := context.Background()
			result, err := chain.ContinueWithSignature(ctx, req.WorkflowContext, req.Signature)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"result": result})
		})
	}

	// 启动HTTP服务器
	server := &http.Server{
		Addr:    ":9092",
		Handler: router,
	}

	go func() {
		log.Printf("🚀 Chain服务启动在端口: %d", 9092)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start Chain server:", err)
		}
	}()

	// 启动状态监控
	log.Println("🎯 启动状态监控...")
	go func() {
		ticker := time.NewTicker(time.Duration(cfg.QNG.PollInterval) * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				log.Printf("📊 链状态监控 - 间隔: %dms", cfg.QNG.PollInterval)
				// 这里可以添加更多的监控逻辑
			}
		}
	}()

	log.Println("✅ QNG Chain服务已启动")
	log.Printf("📡 监控间隔: %dms", cfg.QNG.PollInterval)
	log.Printf("🌐 图节点数: %d", cfg.QNG.GraphNodes)

	// 启动健康检查
	registry.StartHealthCheck()

	// 等待中断信号
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	log.Println("Chain服务正在运行，按 Ctrl+C 停止")
	<-c

	log.Println("正在关闭Chain服务...")

	// 注销服务
	registry.UnregisterService("chain")

	// 关闭Chain
	if err := chain.Stop(); err != nil {
		log.Printf("关闭Chain服务时出错: %v", err)
	}

	// 关闭HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Chain服务关闭失败: %v", err)
	}

	log.Println("Chain服务已关闭")
}

package mcp

import (
	"context"
	"fmt"
	"log"
	"qng_agent/internal/config"
	"sync"
)

type Server struct {
	config      config.MCPConfig
	qngServer   *QNGServer
	metamaskServer *MetaMaskServer
	mu          sync.RWMutex
	running     bool
}

func NewServer(config config.MCPConfig) *Server {
	log.Printf("🔧 创建MCP服务器")
	log.Printf("📋 QNG配置: enabled=%v, host=%s, port=%d", config.QNG.Enabled, config.QNG.Host, config.QNG.Port)
	log.Printf("📋 MetaMask配置: enabled=%v, host=%s, port=%d", config.MetaMask.Enabled, config.MetaMask.Host, config.MetaMask.Port)
	
	server := &Server{
		config: config,
	}
	
	// 初始化QNG服务器
	if config.QNG.Enabled {
		log.Printf("🔧 初始化QNG MCP服务器")
		server.qngServer = NewQNGServer(config.QNG)
		log.Printf("✅ QNG服务器初始化完成")
	} else {
		log.Printf("⚠️  QNG服务未启用")
	}
	
	// 初始化MetaMask服务器
	if config.MetaMask.Enabled {
		log.Printf("🔧 初始化MetaMask MCP服务器")
		server.metamaskServer = NewMetaMaskServer(config.MetaMask)
		log.Printf("✅ MetaMask服务器初始化完成")
	} else {
		log.Printf("⚠️  MetaMask服务未启用")
	}
	
	return server
}

func (s *Server) Start() error {
	log.Printf("🚀 MCP服务器启动")
	
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()
	
	// 启动QNG服务器
	if s.qngServer != nil {
		log.Printf("🚀 启动QNG MCP服务器")
		if err := s.qngServer.Start(); err != nil {
			log.Printf("❌ 启动QNG服务器失败: %v", err)
			return fmt.Errorf("failed to start QNG server: %w", err)
		}
		log.Printf("✅ QNG MCP服务器启动成功")
	}
	
	// 启动MetaMask服务器
	if s.metamaskServer != nil {
		log.Printf("🚀 启动MetaMask MCP服务器")
		if err := s.metamaskServer.Start(); err != nil {
			log.Printf("❌ 启动MetaMask服务器失败: %v", err)
			return fmt.Errorf("failed to start MetaMask server: %w", err)
		}
		log.Printf("✅ MetaMask MCP服务器启动成功")
	}
	
	log.Printf("✅ MCP服务器启动完成")
	return nil
}

func (s *Server) Stop() error {
	log.Printf("🛑 MCP服务器停止")
	
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
	
	// 停止QNG服务器
	if s.qngServer != nil {
		log.Printf("🛑 停止QNG MCP服务器")
		if err := s.qngServer.Stop(); err != nil {
			log.Printf("❌ 停止QNG服务器失败: %v", err)
		} else {
			log.Printf("✅ QNG MCP服务器停止成功")
		}
	}
	
	// 停止MetaMask服务器
	if s.metamaskServer != nil {
		log.Printf("🛑 停止MetaMask MCP服务器")
		if err := s.metamaskServer.Stop(); err != nil {
			log.Printf("❌ 停止MetaMask服务器失败: %v", err)
		} else {
			log.Printf("✅ MetaMask MCP服务器停止成功")
		}
	}
	
	log.Printf("✅ MCP服务器停止完成")
	return nil
}

func (s *Server) Call(ctx context.Context, service string, method string, params map[string]any) (any, error) {
	log.Printf("🔄 MCP服务器调用")
	log.Printf("🔧 服务: %s", service)
	log.Printf("🛠️  方法: %s", method)
	log.Printf("📋 参数: %+v", params)
	
	s.mu.RLock()
	if !s.running {
		s.mu.RUnlock()
		log.Printf("❌ MCP服务器未运行")
		return nil, fmt.Errorf("MCP server is not running")
	}
	s.mu.RUnlock()
	
	switch service {
	case "qng":
		if s.qngServer == nil {
			log.Printf("❌ QNG服务未启用")
			return nil, fmt.Errorf("QNG service not enabled")
		}
		log.Printf("🔄 调用QNG服务")
		return s.qngServer.Call(ctx, method, params)
		
	case "metamask":
		if s.metamaskServer == nil {
			log.Printf("❌ MetaMask服务未启用")
			return nil, fmt.Errorf("MetaMask service not enabled")
		}
		log.Printf("🔄 调用MetaMask服务")
		return s.metamaskServer.Call(ctx, method, params)
		
	default:
		log.Printf("❌ 未知服务: %s", service)
		return nil, fmt.Errorf("unknown service: %s", service)
	}
}

func (s *Server) GetCapabilities() map[string][]Capability {
	log.Printf("📋 获取MCP服务器能力")
	
	capabilities := make(map[string][]Capability)
	
	// QNG服务能力
	if s.qngServer != nil {
		log.Printf("📋 获取QNG服务能力")
		capabilities["qng"] = s.qngServer.GetCapabilities()
	}
	
	// MetaMask服务能力
	if s.metamaskServer != nil {
		log.Printf("📋 获取MetaMask服务能力")
		capabilities["metamask"] = s.metamaskServer.GetCapabilities()
	}
	
	log.Printf("✅ 返回 %d 个服务的能力", len(capabilities))
	return capabilities
}

func (s *Server) GetServices() []string {
	log.Printf("📋 获取可用服务列表")
	
	services := make([]string, 0)
	
	if s.qngServer != nil {
		services = append(services, "qng")
		log.Printf("✅ QNG服务可用")
	}
	
	if s.metamaskServer != nil {
		services = append(services, "metamask")
		log.Printf("✅ MetaMask服务可用")
	}
	
	log.Printf("📋 可用服务: %v", services)
	return services
} 
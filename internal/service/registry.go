package service

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// ServiceRegistry 服务注册中心
type ServiceRegistry struct {
	services map[string]*ServiceInfo
	mu       sync.RWMutex
}

// ServiceInfo 服务信息
type ServiceInfo struct {
	Name      string            `json:"name"`
	Address   string            `json:"address"`
	Port      int               `json:"port"`
	Status    string            `json:"status"`
	LastSeen  time.Time         `json:"last_seen"`
	Endpoints []string          `json:"endpoints"`
	Metadata  map[string]string `json:"metadata"`
}

// ServiceClient 服务客户端接口
type ServiceClient interface {
	Call(ctx context.Context, endpoint string, request interface{}) (interface{}, error)
	GetStatus() string
	GetEndpoints() []string
}

var (
	globalRegistry *ServiceRegistry
	once           sync.Once
)

// GetRegistry 获取全局服务注册中心
func GetRegistry() *ServiceRegistry {
	once.Do(func() {
		globalRegistry = &ServiceRegistry{
			services: make(map[string]*ServiceInfo),
		}
	})
	return globalRegistry
}

// RegisterService 注册服务
func (r *ServiceRegistry) RegisterService(service *ServiceInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	service.Status = "running"
	service.LastSeen = time.Now()
	r.services[service.Name] = service

	log.Printf("✅ 服务已注册: %s @ %s:%d", service.Name, service.Address, service.Port)
	return nil
}

// UnregisterService 注销服务
func (r *ServiceRegistry) UnregisterService(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if service, exists := r.services[name]; exists {
		service.Status = "stopped"
		delete(r.services, name)
		log.Printf("❌ 服务已注销: %s", name)
	}
	return nil
}

// GetService 获取服务信息
func (r *ServiceRegistry) GetService(name string) (*ServiceInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if service, exists := r.services[name]; exists {
		return service, nil
	}
	return nil, fmt.Errorf("service %s not found", name)
}

// GetAllServices 获取所有服务
func (r *ServiceRegistry) GetAllServices() map[string]*ServiceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make(map[string]*ServiceInfo)
	for name, service := range r.services {
		services[name] = service
	}
	return services
}

// CallService 调用服务
func (r *ServiceRegistry) CallService(ctx context.Context, serviceName, endpoint string, request interface{}) (interface{}, error) {
	service, err := r.GetService(serviceName)
	if err != nil {
		return nil, err
	}

	// 构建完整的URL
	url := fmt.Sprintf("http://%s:%d%s", service.Address, service.Port, endpoint)

	// 这里可以实现具体的HTTP调用逻辑
	// 为了简化，我们返回一个模拟响应
	return map[string]interface{}{
		"service":  serviceName,
		"endpoint": endpoint,
		"status":   "success",
		"url":      url,
	}, nil
}

// StartHealthCheck 启动健康检查
func (r *ServiceRegistry) StartHealthCheck() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				r.healthCheck()
			}
		}
	}()
}

// healthCheck 健康检查
func (r *ServiceRegistry) healthCheck() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for name, service := range r.services {
		// 检查服务是否响应
		url := fmt.Sprintf("http://%s:%d/health", service.Address, service.Port)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(url)

		if err != nil || resp.StatusCode != http.StatusOK {
			service.Status = "unhealthy"
			log.Printf("⚠️ 服务健康检查失败: %s", name)
		} else {
			service.Status = "healthy"
			service.LastSeen = time.Now()
		}

		if resp != nil {
			resp.Body.Close()
		}
	}
}

// HTTPServiceClient HTTP服务客户端
type HTTPServiceClient struct {
	registry *ServiceRegistry
	service  string
	client   *http.Client
}

// NewHTTPServiceClient 创建HTTP服务客户端
func NewHTTPServiceClient(service string) *HTTPServiceClient {
	return &HTTPServiceClient{
		registry: GetRegistry(),
		service:  service,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// Call 调用服务
func (c *HTTPServiceClient) Call(ctx context.Context, endpoint string, request interface{}) (interface{}, error) {
	return c.registry.CallService(ctx, c.service, endpoint, request)
}

// GetStatus 获取服务状态
func (c *HTTPServiceClient) GetStatus() string {
	service, err := c.registry.GetService(c.service)
	if err != nil {
		return "unknown"
	}
	return service.Status
}

// GetEndpoints 获取服务端点
func (c *HTTPServiceClient) GetEndpoints() []string {
	service, err := c.registry.GetService(c.service)
	if err != nil {
		return []string{}
	}
	return service.Endpoints
}

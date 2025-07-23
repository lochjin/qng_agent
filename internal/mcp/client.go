package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"qng_agent/internal/config"
	"time"
)

// HTTPClient MCP HTTP 客户端
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
	config     config.MCPConfig
}

// MCPRequest MCP 请求结构
type MCPRequest struct {
	Server string                 `json:"server"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

// MCPResponse MCP 响应结构
type MCPResponse struct {
	Result interface{} `json:"result"`
	Error  string      `json:"error,omitempty"`
}

// NewHTTPClient 创建新的 MCP HTTP 客户端
func NewHTTPClient(config config.MCPConfig) *HTTPClient {
	baseURL := fmt.Sprintf("http://%s:%d", config.Host, 9091) // 使用固定的 MCP 服务器端口
	
	log.Printf("🔧 创建MCP HTTP客户端: %s", baseURL)
	
	return &HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
		config: config,
	}
}

// Call 调用 MCP 服务器方法
func (c *HTTPClient) Call(ctx context.Context, server, method string, params map[string]interface{}) (interface{}, error) {
	log.Printf("🔄 MCP服务器调用")
	log.Printf("🔧 服务: %s", server)
	log.Printf("🛠️  方法: %s", method)
	log.Printf("📋 参数: %v", params)
	
	// 检查服务器连接
	if !c.isServerRunning() {
		log.Printf("❌ MCP服务器未运行")
		return nil, fmt.Errorf("MCP server is not running")
	}
	
	// 构建请求
	reqBody := MCPRequest{
		Server: server,
		Method: method,
		Params: params,
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// 发送 HTTP 请求
	url := c.baseURL + "/api/mcp/call"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("❌ HTTP请求失败: %v", err)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ HTTP状态码错误: %d", resp.StatusCode)
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
	
	// 解析响应
	var mcpResp MCPResponse
	if err := json.NewDecoder(resp.Body).Decode(&mcpResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	if mcpResp.Error != "" {
		log.Printf("❌ MCP错误: %s", mcpResp.Error)
		return nil, fmt.Errorf("MCP error: %s", mcpResp.Error)
	}
	
	log.Printf("✅ MCP调用成功")
	return mcpResp.Result, nil
}

// Start 启动客户端（HTTP 客户端不需要启动）
func (c *HTTPClient) Start() error {
	log.Printf("🔗 MCP HTTP客户端已就绪")
	return nil
}

// Stop 停止客户端
func (c *HTTPClient) Stop() error {
	log.Printf("🔌 MCP HTTP客户端已断开")
	return nil
}

// GetCapabilities 获取服务器能力
func (c *HTTPClient) GetCapabilities() map[string]interface{} {
	ctx := context.Background()
	
	url := c.baseURL + "/api/mcp/capabilities"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Printf("❌ 创建能力查询请求失败: %v", err)
		return make(map[string]interface{})
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("❌ 能力查询请求失败: %v", err)
		return make(map[string]interface{})
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ 能力查询状态码错误: %d", resp.StatusCode)
		return make(map[string]interface{})
	}
	
	var response struct {
		Capabilities map[string]interface{} `json:"capabilities"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("❌ 解析能力响应失败: %v", err)
		return make(map[string]interface{})
	}
	
	return response.Capabilities
}

// isServerRunning 检查服务器是否运行
func (c *HTTPClient) isServerRunning() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	url := c.baseURL + "/health"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Printf("❌ 创建健康检查请求失败: %v", err)
		return false
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("❌ 健康检查请求失败: %v", err)
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == http.StatusOK
}

// ServerInterface 定义服务器接口，兼容原有代码
type ServerInterface interface {
	Call(ctx context.Context, server, method string, params map[string]interface{}) (interface{}, error)
	Start() error
	Stop() error
	GetCapabilities() map[string]interface{}
}

// 确保 HTTPClient 实现 ServerInterface
var _ ServerInterface = (*HTTPClient)(nil) 
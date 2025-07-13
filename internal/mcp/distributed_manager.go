package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"qng_agent/internal/service"
	"time"
)

// DistributedManager 分布式MCP管理器，通过HTTP与远程MCP服务通信
type DistributedManager struct {
	mcpClient *service.HTTPServiceClient
	client    *http.Client
}

// NewDistributedManager 创建分布式MCP管理器
func NewDistributedManager(mcpClient *service.HTTPServiceClient) *DistributedManager {
	return &DistributedManager{
		mcpClient: mcpClient,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// CallTool 调用MCP工具
func (dm *DistributedManager) CallTool(ctx context.Context, serverName, method string, params map[string]any) (any, error) {
	// 构建请求
	requestBody := map[string]interface{}{
		"server": serverName,
		"method": method,
		"params": params,
	}

	return dm.makeHTTPRequest(ctx, "POST", "/api/mcp/call", requestBody)
}

// CallQNGWorkflow 调用QNG工作流
func (dm *DistributedManager) CallQNGWorkflow(ctx context.Context, message string) (string, error) {
	requestBody := map[string]interface{}{
		"message": message,
	}

	result, err := dm.makeHTTPRequest(ctx, "POST", "/api/mcp/qng/workflow", requestBody)
	if err != nil {
		return "", err
	}

	// 解析响应
	if resultMap, ok := result.(map[string]interface{}); ok {
		if workflowID, exists := resultMap["workflow_id"]; exists {
			if id, ok := workflowID.(string); ok {
				return id, nil
			}
		}
	}

	return "", fmt.Errorf("invalid workflow response format")
}

// GetQNGWorkflowStatus 获取QNG工作流状态
func (dm *DistributedManager) GetQNGWorkflowStatus(ctx context.Context, workflowID string) (*WorkflowStatus, error) {
	endpoint := fmt.Sprintf("/api/mcp/qng/workflow/%s/status", workflowID)

	result, err := dm.makeHTTPRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	// 将结果转换为WorkflowStatus
	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal workflow status: %w", err)
	}

	var status WorkflowStatus
	if err := json.Unmarshal(jsonData, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workflow status: %w", err)
	}

	return &status, nil
}

// SubmitWorkflowSignature 提交工作流签名
func (dm *DistributedManager) SubmitWorkflowSignature(ctx context.Context, workflowID, signature string) (any, error) {
	endpoint := fmt.Sprintf("/api/mcp/qng/workflow/%s/signature", workflowID)

	requestBody := map[string]interface{}{
		"signature": signature,
	}

	return dm.makeHTTPRequest(ctx, "POST", endpoint, requestBody)
}

// GetAllCapabilities 获取所有能力
func (dm *DistributedManager) GetAllCapabilities() map[string][]Capability {
	ctx := context.Background()
	result, err := dm.makeHTTPRequest(ctx, "GET", "/api/mcp/capabilities", nil)
	if err != nil {
		return make(map[string][]Capability)
	}

	// 解析能力
	if resultMap, ok := result.(map[string]interface{}); ok {
		if capabilities, exists := resultMap["capabilities"]; exists {
			// 转换为正确的格式
			capMap := make(map[string][]Capability)

			if capData, ok := capabilities.(map[string]interface{}); ok {
				for serverName, serverCaps := range capData {
					if capsArray, ok := serverCaps.([]interface{}); ok {
						var caps []Capability
						for _, cap := range capsArray {
							if capMap, ok := cap.(map[string]interface{}); ok {
								capability := Capability{
									Name:        getStringFromMap(capMap, "name"),
									Description: getStringFromMap(capMap, "description"),
								}

								// 解析参数
								if params, exists := capMap["parameters"]; exists {
									if paramsArray, ok := params.([]interface{}); ok {
										for _, param := range paramsArray {
											if paramMap, ok := param.(map[string]interface{}); ok {
												parameter := Parameter{
													Name:        getStringFromMap(paramMap, "name"),
													Type:        getStringFromMap(paramMap, "type"),
													Description: getStringFromMap(paramMap, "description"),
													Required:    getBoolFromMap(paramMap, "required"),
												}
												capability.Parameters = append(capability.Parameters, parameter)
											}
										}
									}
								}

								caps = append(caps, capability)
							}
						}
						capMap[serverName] = caps
					}
				}
			}

			return capMap
		}
	}

	return make(map[string][]Capability)
}

// makeHTTPRequest 发送HTTP请求到MCP服务
func (dm *DistributedManager) makeHTTPRequest(ctx context.Context, method, endpoint string, body interface{}) (interface{}, error) {
	// 获取MCP服务信息
	registry := service.GetRegistry()
	mcpService, err := registry.GetService("mcp")
	if err != nil {
		return nil, fmt.Errorf("MCP service not found: %w", err)
	}

	// 构建URL
	url := fmt.Sprintf("http://%s:%d%s", mcpService.Address, mcpService.Port, endpoint)

	// 准备请求体
	var requestBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		requestBody = bytes.NewBuffer(jsonData)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, method, url, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// 发送请求
	resp, err := dm.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	// 解析JSON响应
	var result interface{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

// 辅助函数
func getStringFromMap(m map[string]interface{}, key string) string {
	if value, exists := m[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func getBoolFromMap(m map[string]interface{}, key string) bool {
	if value, exists := m[key]; exists {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return false
}

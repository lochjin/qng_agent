package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Client RPC客户端
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// TransactionReceipt 交易收据
type TransactionReceipt struct {
	TransactionHash string `json:"transactionHash"`
	BlockNumber     string `json:"blockNumber"`
	Status          string `json:"status"`
	Success         bool   `json:"success"`
}

// RPCRequest RPC请求结构
type RPCRequest struct {
	JsonRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// RPCResponse RPC响应结构
type RPCResponse struct {
	JsonRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result"`
	Error   *RPCError   `json:"error"`
}

// RPCError RPC错误结构
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewClient 创建新的RPC客户端
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetTransactionReceipt 获取交易收据
func (c *Client) GetTransactionReceipt(ctx context.Context, txHash string) (*TransactionReceipt, error) {
	log.Printf("🔍 查询交易收据: %s", txHash)
	
	// 构建RPC请求
	request := RPCRequest{
		JsonRPC: "2.0",
		Method:  "eth_getTransactionReceipt",
		Params:  []interface{}{txHash},
		ID:      1,
	}
	
	// 发送请求
	response, err := c.sendRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("发送RPC请求失败: %w", err)
	}
	
	// 解析响应
	if response.Error != nil {
		return nil, fmt.Errorf("RPC错误: %s", response.Error.Message)
	}
	
	if response.Result == nil {
		// 交易还未被打包
		return nil, nil
	}
	
	// 解析交易收据
	receiptBytes, err := json.Marshal(response.Result)
	if err != nil {
		return nil, fmt.Errorf("解析交易收据失败: %w", err)
	}
	
	var receipt TransactionReceipt
	if err := json.Unmarshal(receiptBytes, &receipt); err != nil {
		return nil, fmt.Errorf("反序列化交易收据失败: %w", err)
	}
	
	// 检查交易状态
	receipt.Success = receipt.Status == "0x1"
	
	log.Printf("✅ 交易收据查询成功: 状态=%s, 区块=%s", receipt.Status, receipt.BlockNumber)
	return &receipt, nil
}

// GetBlockNumber 获取当前区块号
func (c *Client) GetBlockNumber(ctx context.Context) (int64, error) {
	request := RPCRequest{
		JsonRPC: "2.0",
		Method:  "eth_blockNumber",
		Params:  []interface{}{},
		ID:      1,
	}
	
	response, err := c.sendRequest(ctx, request)
	if err != nil {
		return 0, fmt.Errorf("获取区块号失败: %w", err)
	}
	
	if response.Error != nil {
		return 0, fmt.Errorf("RPC错误: %s", response.Error.Message)
	}
	
	// 解析十六进制区块号
	blockNumHex, ok := response.Result.(string)
	if !ok {
		return 0, fmt.Errorf("无效的区块号格式")
	}
	
	var blockNum int64
	_, err = fmt.Sscanf(blockNumHex, "0x%x", &blockNum)
	if err != nil {
		return 0, fmt.Errorf("解析区块号失败: %w", err)
	}
	
	return blockNum, nil
}

// sendRequest 发送RPC请求
func (c *Client) sendRequest(ctx context.Context, request RPCRequest) (*RPCResponse, error) {
	// 序列化请求
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}
	
	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(requestBytes))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	
	// 发送请求
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer httpResp.Body.Close()
	
	// 解析响应
	var response RPCResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	
	return &response, nil
}

// WaitForTransactionConfirmation 等待交易确认
func (c *Client) WaitForTransactionConfirmation(ctx context.Context, txHash string, requiredConfirmations int, pollingInterval time.Duration) (*TransactionReceipt, error) {
	log.Printf("⏳ 开始等待交易确认: %s (需要 %d 个确认)", txHash, requiredConfirmations)
	
	ticker := time.NewTicker(pollingInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			// 查询交易收据
			receipt, err := c.GetTransactionReceipt(ctx, txHash)
			if err != nil {
				log.Printf("⚠️ 查询交易收据失败: %v", err)
				continue
			}
			
			if receipt == nil {
				log.Printf("⏳ 交易尚未被打包，继续等待...")
				continue
			}
			
			if !receipt.Success {
				return receipt, fmt.Errorf("交易执行失败")
			}
			
			// 获取当前区块号
			currentBlock, err := c.GetBlockNumber(ctx)
			if err != nil {
				log.Printf("⚠️ 获取当前区块号失败: %v", err)
				continue
			}
			
			// 解析交易所在区块号
			var txBlock int64
			_, err = fmt.Sscanf(receipt.BlockNumber, "0x%x", &txBlock)
			if err != nil {
				log.Printf("⚠️ 解析交易区块号失败: %v", err)
				continue
			}
			
			confirmations := currentBlock - txBlock + 1
			log.Printf("🔍 交易确认数: %d/%d (当前区块: %d, 交易区块: %d)", 
				confirmations, requiredConfirmations, currentBlock, txBlock)
			
			if confirmations >= int64(requiredConfirmations) {
				log.Printf("✅ 交易确认完成: %s", txHash)
				return receipt, nil
			}
			
			log.Printf("⏳ 需要更多确认，继续等待...")
		}
	}
} 
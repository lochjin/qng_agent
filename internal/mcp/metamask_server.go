package mcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"qng_agent/internal/config"
	"time"
)

type MetaMaskServer struct {
	config config.MetaMaskConfig
	// 模拟钱包连接状态
	connected bool
	accounts  []string
	network   string
}

func NewMetaMaskServer(config config.MetaMaskConfig) *MetaMaskServer {
	return &MetaMaskServer{
		config: config,
		network: config.Network,
	}
}

func (s *MetaMaskServer) Start() error {
	log.Printf("🚀 MetaMask服务器启动")
	return nil
}

func (s *MetaMaskServer) Stop() error {
	log.Printf("🛑 MetaMask服务器停止")
	return nil
}

func (s *MetaMaskServer) Call(ctx context.Context, method string, params map[string]any) (any, error) {
	log.Printf("🔄 MetaMask服务器调用")
	log.Printf("🛠️  方法: %s", method)
	log.Printf("📋 参数: %+v", params)
	
	switch method {
	case "connect_wallet":
		return s.connectWallet(ctx, params)
	case "get_accounts":
		return s.getAccounts(ctx, params)
	case "sign_transaction":
		return s.signTransaction(ctx, params)
	case "get_balance":
		return s.getBalance(ctx, params)
	case "get_network":
		return s.getNetwork(ctx, params)
	default:
		log.Printf("❌ 未知方法: %s", method)
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}

func (s *MetaMaskServer) connectWallet(ctx context.Context, params map[string]any) (any, error) {
	log.Printf("🔗 连接MetaMask钱包")
	
	// 模拟连接过程
	time.Sleep(1 * time.Second)
	
	// 生成模拟账户
	accounts := []string{
		"0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6",
		"0x1234567890123456789012345678901234567890",
	}
	
	s.connected = true
	s.accounts = accounts
	
	log.Printf("✅ 钱包连接成功")
	log.Printf("📋 账户: %v", accounts)
	
	return map[string]any{
		"connected": true,
		"accounts":  accounts,
		"network":   s.network,
		"chain_id":  "1", // Ethereum mainnet
	}, nil
}

func (s *MetaMaskServer) getAccounts(ctx context.Context, params map[string]any) (any, error) {
	log.Printf("📋 获取账户列表")
	
	if !s.connected {
		log.Printf("❌ 钱包未连接")
		return nil, fmt.Errorf("wallet not connected")
	}
	
	log.Printf("✅ 返回账户列表: %v", s.accounts)
	return s.accounts, nil
}

func (s *MetaMaskServer) signTransaction(ctx context.Context, params map[string]any) (any, error) {
	log.Printf("✍️  签名交易")
	
	if !s.connected {
		log.Printf("❌ 钱包未连接")
		return nil, fmt.Errorf("wallet not connected")
	}
	
	// 获取交易数据
	txData, ok := params["transaction"].(map[string]any)
	if !ok {
		log.Printf("❌ 缺少交易数据")
		return nil, fmt.Errorf("transaction data required")
	}
	
	log.Printf("📋 交易数据: %+v", txData)
	
	// 模拟签名过程
	time.Sleep(2 * time.Second)
	
	// 生成模拟签名
	signature := s.generateSignature()
	
	log.Printf("✅ 交易签名成功")
	log.Printf("🔐 签名: %s", signature)
	
	return map[string]any{
		"signature": signature,
		"tx_hash":   "0x" + signature[:40],
		"status":    "signed",
	}, nil
}

func (s *MetaMaskServer) getBalance(ctx context.Context, params map[string]any) (any, error) {
	log.Printf("💰 获取余额")
	
	if !s.connected {
		log.Printf("❌ 钱包未连接")
		return nil, fmt.Errorf("wallet not connected")
	}
	
	account, ok := params["account"].(string)
	if !ok {
		log.Printf("❌ 缺少账户地址")
		return nil, fmt.Errorf("account address required")
	}
	
	log.Printf("📋 查询账户: %s", account)
	
	// 模拟余额查询
	balances := map[string]string{
		"ETH":  "2.5",
		"USDT": "1000.0",
		"BTC":  "0.1",
	}
	
	log.Printf("✅ 余额查询成功: %+v", balances)
	
	return balances, nil
}

func (s *MetaMaskServer) getNetwork(ctx context.Context, params map[string]any) (any, error) {
	log.Printf("🌐 获取网络信息")
	
	if !s.connected {
		log.Printf("❌ 钱包未连接")
		return nil, fmt.Errorf("wallet not connected")
	}
	
	networkInfo := map[string]any{
		"network":  s.network,
		"chain_id": "1",
		"name":     "Ethereum Mainnet",
		"rpc_url":  "https://mainnet.infura.io/v3/your-project-id",
	}
	
	log.Printf("✅ 网络信息: %+v", networkInfo)
	
	return networkInfo, nil
}

func (s *MetaMaskServer) generateSignature() string {
	// 生成随机签名
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (s *MetaMaskServer) GetCapabilities() []Capability {
	return []Capability{
		{
			Name:        "connect_wallet",
			Description: "连接MetaMask钱包",
			Parameters: []Parameter{
				{
					Name:        "request_permissions",
					Type:        "boolean",
					Description: "是否请求权限",
					Required:    false,
				},
			},
		},
		{
			Name:        "get_accounts",
			Description: "获取钱包账户列表",
			Parameters:  []Parameter{},
		},
		{
			Name:        "sign_transaction",
			Description: "签名交易",
			Parameters: []Parameter{
				{
					Name:        "transaction",
					Type:        "object",
					Description: "交易数据",
					Required:    true,
				},
			},
		},
		{
			Name:        "get_balance",
			Description: "获取账户余额",
			Parameters: []Parameter{
				{
					Name:        "account",
					Type:        "string",
					Description: "账户地址",
					Required:    true,
				},
			},
		},
		{
			Name:        "get_network",
			Description: "获取网络信息",
			Parameters:  []Parameter{},
		},
	}
}

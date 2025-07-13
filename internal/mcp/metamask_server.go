package mcp

import (
	"context"
	"fmt"
	"qng_agent/internal/config"
)

type MetaMaskServer struct {
	config   config.MetaMaskConfig
	sessions map[string]*WalletSession
	running  bool
}

type WalletSession struct {
	Address   string `json:"address"`
	ChainID   string `json:"chain_id"`
	Connected bool   `json:"connected"`
}

type ConnectRequest struct {
	QRCode bool `json:"qr_code"`
}

type SignRequest struct {
	Address string `json:"address"`
	Message string `json:"message"`
	Type    string `json:"type"` // "personal_sign", "sign_typed_data", "send_transaction"
}

func NewMetaMaskServer(config config.MetaMaskConfig) *MetaMaskServer {
	return &MetaMaskServer{
		config:   config,
		sessions: make(map[string]*WalletSession),
	}
}

func (s *MetaMaskServer) Start() error {
	s.running = true
	return nil
}

func (s *MetaMaskServer) Stop() error {
	s.running = false
	return nil
}

func (s *MetaMaskServer) Call(ctx context.Context, method string, params map[string]any) (any, error) {
	switch method {
	case "connect_wallet":
		return s.connectWallet(ctx, params)
	case "disconnect_wallet":
		return s.disconnectWallet(ctx, params)
	case "get_accounts":
		return s.getAccounts(ctx, params)
	case "sign_message":
		return s.signMessage(ctx, params)
	case "send_transaction":
		return s.sendTransaction(ctx, params)
	case "get_balance":
		return s.getBalance(ctx, params)
	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}

func (s *MetaMaskServer) connectWallet(ctx context.Context, params map[string]any) (any, error) {
	// 生成连接二维码或返回连接信息
	response := map[string]any{
		"qr_code":   "https://metamask.app.link/connect",
		"deep_link": "metamask://connect",
		"message":   "请使用MetaMask扫描二维码或点击链接连接钱包",
		"status":    "pending",
	}

	return response, nil
}

func (s *MetaMaskServer) disconnectWallet(ctx context.Context, params map[string]any) (any, error) {
	sessionID, ok := params["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("session_id parameter is required")
	}

	delete(s.sessions, sessionID)

	return map[string]any{
		"status": "disconnected",
	}, nil
}

func (s *MetaMaskServer) getAccounts(ctx context.Context, params map[string]any) (any, error) {
	// 模拟返回账户信息
	accounts := []string{
		"0x1234567890123456789012345678901234567890",
	}

	return map[string]any{
		"accounts": accounts,
		"network":  s.config.Network,
	}, nil
}

func (s *MetaMaskServer) signMessage(ctx context.Context, params map[string]any) (any, error) {
	message, ok := params["message"].(string)
	if !ok {
		return nil, fmt.Errorf("message parameter is required")
	}

	address, ok := params["address"].(string)
	if !ok {
		return nil, fmt.Errorf("address parameter is required")
	}

	// 返回签名请求信息
	return map[string]any{
		"type":    "signature_request",
		"message": message,
		"address": address,
		"method":  "personal_sign",
		"status":  "pending",
	}, nil
}

func (s *MetaMaskServer) sendTransaction(ctx context.Context, params map[string]any) (any, error) {
	to, ok := params["to"].(string)
	if !ok {
		return nil, fmt.Errorf("to parameter is required")
	}

	value, ok := params["value"].(string)
	if !ok {
		return nil, fmt.Errorf("value parameter is required")
	}

	// 返回交易请求信息
	return map[string]any{
		"type":     "transaction_request",
		"to":       to,
		"value":    value,
		"gas":      "21000",
		"gasPrice": "20000000000",
		"status":   "pending",
	}, nil
}

func (s *MetaMaskServer) getBalance(ctx context.Context, params map[string]any) (any, error) {
	address, ok := params["address"].(string)
	if !ok {
		return nil, fmt.Errorf("address parameter is required")
	}

	// 模拟返回余额信息
	return map[string]any{
		"address": address,
		"balance": "1.5", // ETH
		"tokens": map[string]string{
			"USDT": "1000.0",
			"BTC":  "0.1",
		},
	}, nil
}

func (s *MetaMaskServer) GetCapabilities() []Capability {
	return []Capability{
		{
			Name:        "connect_wallet",
			Description: "连接MetaMask钱包",
			Parameters: []Parameter{
				{
					Name:        "qr_code",
					Type:        "boolean",
					Description: "是否生成二维码",
					Required:    false,
				},
			},
		},
		{
			Name:        "get_accounts",
			Description: "获取钱包账户",
			Parameters:  []Parameter{},
		},
		{
			Name:        "sign_message",
			Description: "签名消息",
			Parameters: []Parameter{
				{
					Name:        "message",
					Type:        "string",
					Description: "要签名的消息",
					Required:    true,
				},
				{
					Name:        "address",
					Type:        "string",
					Description: "签名地址",
					Required:    true,
				},
			},
		},
		{
			Name:        "send_transaction",
			Description: "发送交易",
			Parameters: []Parameter{
				{
					Name:        "to",
					Type:        "string",
					Description: "接收地址",
					Required:    true,
				},
				{
					Name:        "value",
					Type:        "string",
					Description: "转账金额",
					Required:    true,
				},
			},
		},
		{
			Name:        "get_balance",
			Description: "获取余额",
			Parameters: []Parameter{
				{
					Name:        "address",
					Type:        "string",
					Description: "查询地址",
					Required:    true,
				},
			},
		},
	}
}

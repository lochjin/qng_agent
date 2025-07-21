package contracts

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"path/filepath"
	"strconv"
	"strings"
	"regexp"
)

// ContractManager 合约管理器
type ContractManager struct {
	config     *ContractConfig
	artifacts  map[string]*ContractArtifact
}

// ContractConfig 合约配置结构
type ContractConfig struct {
	Version   string                        `json:"version"`
	Network   NetworkConfig                 `json:"network"`
	Tokens    map[string]TokenConfig        `json:"tokens"`
	Contracts map[string]ContractInfo       `json:"contracts"`
	Workflows map[string]WorkflowConfig     `json:"workflows"`
}

// NetworkConfig 网络配置
type NetworkConfig struct {
	ChainID int    `json:"chainId"`
	Name    string `json:"name"`
	RPCURL  string `json:"rpcUrl"`
}

// TokenConfig 代币配置
type TokenConfig struct {
	Name            string `json:"name"`
	Symbol          string `json:"symbol"`
	Decimals        int    `json:"decimals"`
	IsNative        bool   `json:"isNative"`
	ContractAddress string `json:"contractAddress,omitempty"`
	ContractName    string `json:"contractName,omitempty"`
	Description     string `json:"description"`
}

// ContractInfo 合约信息
type ContractInfo struct {
	Name         string                    `json:"name"`
	Address      string                    `json:"address"`
	ArtifactPath string                    `json:"artifactPath"`
	Type         string                    `json:"type"`
	Description  string                    `json:"description"`
	Functions    map[string]FunctionInfo   `json:"functions"`
	SupportedPairs []SwapPair              `json:"supportedPairs,omitempty"`
}

// FunctionInfo 函数信息
type FunctionInfo struct {
	Signature    string          `json:"signature"`
	Description  string          `json:"description"`
	Payable      bool            `json:"payable,omitempty"`
	Parameters   []ParameterInfo `json:"parameters"`
	ExchangeRate string          `json:"exchangeRate,omitempty"`
}

// ParameterInfo 参数信息
type ParameterInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// SwapPair 交换对信息
type SwapPair struct {
	From        string  `json:"from"`
	To          string  `json:"to"`
	Method      string  `json:"method"`
	Rate        float64 `json:"rate"`
	Description string  `json:"description"`
}

// WorkflowConfig 工作流配置
type WorkflowConfig struct {
	Description     string   `json:"description"`
	SupportedPairs  []string `json:"supportedPairs,omitempty"`
	SupportedTokens []string `json:"supportedTokens,omitempty"`
	Contract        string   `json:"contract"`
	Patterns        []string `json:"patterns"`
}

// ContractArtifact 合约编译产物
type ContractArtifact struct {
	ContractName string      `json:"contractName"`
	ABI          interface{} `json:"abi"`
	Bytecode     string      `json:"bytecode"`
}

// TransactionData 交易数据
type TransactionData struct {
	To       string `json:"to"`
	Value    string `json:"value"`
	Data     string `json:"data"`
	GasLimit string `json:"gasLimit"`
	GasPrice string `json:"gasPrice"`
}

// SwapRequest 兑换请求
type SwapRequest struct {
	FromToken string
	ToToken   string
	Amount    string
	UserAddress string
}

// StakeRequest 质押请求
type StakeRequest struct {
	Token       string
	Amount      string
	Action      string // "stake", "unstake", "claimRewards"
	UserAddress string
}

// NewContractManager 创建合约管理器
func NewContractManager(configPath string) (*ContractManager, error) {
	log.Printf("🔧 初始化合约管理器")
	log.Printf("📋 配置文件路径: %s", configPath)
	
	manager := &ContractManager{
		artifacts: make(map[string]*ContractArtifact),
	}
	
	// 加载配置文件
	if err := manager.LoadConfig(configPath); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	
	// 加载合约 ABI
	if err := manager.LoadArtifacts(); err != nil {
		return nil, fmt.Errorf("failed to load artifacts: %w", err)
	}
	
	log.Printf("✅ 合约管理器初始化完成")
	return manager, nil
}

// LoadConfig 加载合约配置
func (cm *ContractManager) LoadConfig(configPath string) error {
	log.Printf("📋 加载合约配置: %s", configPath)
	
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	
	cm.config = &ContractConfig{}
	if err := json.Unmarshal(data, cm.config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	
	log.Printf("✅ 配置加载成功")
	log.Printf("📋 网络: %s (Chain ID: %d)", cm.config.Network.Name, cm.config.Network.ChainID)
	log.Printf("📋 代币数量: %d", len(cm.config.Tokens))
	log.Printf("📋 合约数量: %d", len(cm.config.Contracts))
	
	return nil
}

// LoadArtifacts 加载合约 ABI
func (cm *ContractManager) LoadArtifacts() error {
	log.Printf("📋 加载合约 ABI")
	
	for name, contract := range cm.config.Contracts {
		log.Printf("📋 加载合约 %s 的 ABI", name)
		
		artifactPath := contract.ArtifactPath
		if !filepath.IsAbs(artifactPath) {
			artifactPath = filepath.Join(".", artifactPath)
		}
		
		data, err := ioutil.ReadFile(artifactPath)
		if err != nil {
			log.Printf("⚠️  无法读取 %s 的 ABI 文件: %v", name, err)
			continue
		}
		
		artifact := &ContractArtifact{}
		if err := json.Unmarshal(data, artifact); err != nil {
			log.Printf("⚠️  无法解析 %s 的 ABI: %v", name, err)
			continue
		}
		
		cm.artifacts[name] = artifact
		log.Printf("✅ %s ABI 加载成功", name)
	}
	
	return nil
}

// ParseSwapRequest 解析兑换请求
func (cm *ContractManager) ParseSwapRequest(message string) (*SwapRequest, error) {
	log.Printf("🔄 解析兑换请求: %s", message)
	
	// 定义匹配模式
	patterns := []string{
		`兑换\s*(\d+(?:\.\d+)?)\s*(\w+)(?:\s*为?\s*(\w+))?`,
		`将\s*(\d+(?:\.\d+)?)\s*(\w+)\s*换成\s*(\w+)`,
		`swap\s+(\d+(?:\.\d+)?)\s+(\w+)\s+to\s+(\w+)`,
		`exchange\s+(\d+(?:\.\d+)?)\s+(\w+)\s+for\s+(\w+)`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(message)
		
		if len(matches) >= 3 {
			amount := matches[1]
			fromToken := strings.ToUpper(matches[2])
			toToken := ""
			
			if len(matches) >= 4 && matches[3] != "" {
				toToken = strings.ToUpper(matches[3])
			} else {
				// 如果没有指定目标代币，根据源代币推断
				if fromToken == "MEER" {
					toToken = "MTK"
				} else if fromToken == "MTK" {
					toToken = "MEER"
				}
			}
			
			if toToken == "" {
				continue
			}
			
			log.Printf("✅ 解析成功: %s %s -> %s", amount, fromToken, toToken)
			
			return &SwapRequest{
				FromToken: fromToken,
				ToToken:   toToken,
				Amount:    amount,
			}, nil
		}
	}
	
	return nil, fmt.Errorf("unable to parse swap request from message")
}

// BuildSwapTransaction 构建兑换交易
func (cm *ContractManager) BuildSwapTransaction(req *SwapRequest) (*TransactionData, error) {
	log.Printf("🔄 构建兑换交易")
	log.Printf("📋 从 %s 兑换 %s 到 %s", req.FromToken, req.Amount, req.ToToken)
	
	// 查找支持的交换对
	swapContract := cm.config.Contracts["SimpleSwap"]
	if swapContract.Address == "" {
		return nil, fmt.Errorf("SimpleSwap contract not found")
	}
	
	var swapPair *SwapPair
	for _, pair := range swapContract.SupportedPairs {
		if pair.From == req.FromToken && pair.To == req.ToToken {
			swapPair = &pair
			break
		}
	}
	
	if swapPair == nil {
		return nil, fmt.Errorf("unsupported swap pair: %s -> %s", req.FromToken, req.ToToken)
	}
	
	log.Printf("✅ 找到交换对: %s", swapPair.Description)
	
	// 解析金额
	amount, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %s", req.Amount)
	}
	
	// 构建交易数据
	txData := &TransactionData{
		To:       swapContract.Address,
		GasLimit: "0x186A0",  // 100000 gas
		GasPrice: "0x3B9ACA00", // 1 gwei
	}
	
	if swapPair.Method == "buyToken" {
		// MEER -> MTK：需要发送 ETH
		weiAmount := new(big.Int)
		weiAmount, _ = weiAmount.SetString(fmt.Sprintf("%.0f", amount*1e18), 10)
		txData.Value = "0x" + weiAmount.Text(16)
		txData.Data = "0xa4821719" // buyToken() 函数签名 (ethers.js计算)
		
		log.Printf("📋 MEER -> MTK 交易")
		log.Printf("📋 发送金额: %s MEER", req.Amount)
		log.Printf("📋 预期获得: %.0f MTK", amount*swapPair.Rate)
		
	} else if swapPair.Method == "sellToken" {
		// MTK -> MEER：调用 sellToken 函数
		tokenAmount := new(big.Int)
		tokenAmount, _ = tokenAmount.SetString(fmt.Sprintf("%.0f", amount*1e18), 10)
		
		// sellToken(uint256) 函数调用数据
		// 函数签名: 0x2397e4d7 (ethers.js计算)
		// 参数: 代币数量 (uint256)
		funcSig := "2397e4d7"
		amountHex := fmt.Sprintf("%064s", tokenAmount.Text(16))
		txData.Data = "0x" + funcSig + amountHex
		txData.Value = "0x0"
		
		log.Printf("📋 MTK -> MEER 交易")
		log.Printf("📋 卖出金额: %s MTK", req.Amount)
		log.Printf("📋 预期获得: %.6f MEER", amount*swapPair.Rate)
	}
	
	log.Printf("✅ 交易数据构建完成")
	return txData, nil
}

// ParseStakeRequest 解析质押请求
func (cm *ContractManager) ParseStakeRequest(message string) (*StakeRequest, error) {
	log.Printf("🔄 解析质押请求: %s", message)
	
	// 定义匹配模式
	patterns := []map[string]string{
		{
			"pattern": `质押\s*(\d+(?:\.\d+)?)\s*(\w+)`,
			"action":  "stake",
		},
		{
			"pattern": `将\s*(\d+(?:\.\d+)?)\s*(\w+)\s*质押`,
			"action":  "stake",
		},
		{
			"pattern": `stake\s+(\d+(?:\.\d+)?)\s+(\w+)`,
			"action":  "stake",
		},
		{
			"pattern": `取消质押\s*(\d+(?:\.\d+)?)\s*(\w+)`,
			"action":  "unstake",
		},
		{
			"pattern": `解质押\s*(\d+(?:\.\d+)?)\s*(\w+)`,
			"action":  "unstake",
		},
		{
			"pattern": `unstake\s+(\d+(?:\.\d+)?)\s+(\w+)`,
			"action":  "unstake",
		},
		{
			"pattern": `领取奖励|领取收益|claim\s+rewards|提取奖励|收取奖励`,
			"action":  "claimRewards",
		},
	}
	
	for _, patternInfo := range patterns {
		re := regexp.MustCompile(patternInfo["pattern"])
		matches := re.FindStringSubmatch(message)
		
		if len(matches) > 0 {
			action := patternInfo["action"]
			
			if action == "claimRewards" {
				// 领取奖励不需要金额
				log.Printf("✅ 解析成功: 领取奖励")
				return &StakeRequest{
					Token:  "MTK",
					Amount: "0",
					Action: action,
				}, nil
			} else if len(matches) >= 3 {
				amount := matches[1]
				token := strings.ToUpper(matches[2])
				
				log.Printf("✅ 解析成功: %s %s %s", action, amount, token)
				
				return &StakeRequest{
					Token:  token,
					Amount: amount,
					Action: action,
				}, nil
			}
		}
	}
	
	return nil, fmt.Errorf("unable to parse stake request from message")
}

// BuildStakeTransaction 构建质押交易
func (cm *ContractManager) BuildStakeTransaction(req *StakeRequest) (*TransactionData, error) {
	log.Printf("🔄 构建质押交易")
	log.Printf("📋 操作: %s, 代币: %s, 数量: %s", req.Action, req.Token, req.Amount)
	
	// 查找质押合约
	stakingContract := cm.config.Contracts["MTKStaking"]
	if stakingContract.Address == "" {
		return nil, fmt.Errorf("MTKStaking contract not found")
	}
	
	// 构建交易数据
	txData := &TransactionData{
		To:       stakingContract.Address,
		Value:    "0x0", // 质押不需要发送原生代币
		GasLimit: "0x30D40",  // 200000 gas (足够的余量)
		GasPrice: "0x3B9ACA00", // 1 gwei
	}
	
	switch req.Action {
	case "stake":
		// 质押操作
		amount, err := strconv.ParseFloat(req.Amount, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid amount: %s", req.Amount)
		}
		
		// 将金额转换为 wei
		weiAmount := new(big.Int)
		weiAmount, _ = weiAmount.SetString(fmt.Sprintf("%.0f", amount*1e18), 10)
		
		// stake(uint256) 函数调用数据
		// 函数签名: 0xa694fc3a
		funcSig := "a694fc3a"
		amountHex := fmt.Sprintf("%064s", weiAmount.Text(16))
		txData.Data = "0x" + funcSig + amountHex
		
		log.Printf("📋 质押交易: %s %s", req.Amount, req.Token)
		
	case "unstake":
		// 取消质押操作
		amount, err := strconv.ParseFloat(req.Amount, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid amount: %s", req.Amount)
		}
		
		// 将金额转换为 wei
		weiAmount := new(big.Int)
		weiAmount, _ = weiAmount.SetString(fmt.Sprintf("%.0f", amount*1e18), 10)
		
		// unstake(uint256) 函数调用数据
		// 函数签名: 0x2e17de78
		funcSig := "2e17de78"
		amountHex := fmt.Sprintf("%064s", weiAmount.Text(16))
		txData.Data = "0x" + funcSig + amountHex
		
		log.Printf("📋 取消质押交易: %s %s", req.Amount, req.Token)
		
	case "claimRewards":
		// 领取奖励操作
		// claimRewards() 函数调用数据
		// 函数签名: 0xef5cfb8c
		txData.Data = "0xef5cfb8c"
		
		log.Printf("📋 领取奖励交易")
		
	default:
		return nil, fmt.Errorf("unsupported stake action: %s", req.Action)
	}
	
	log.Printf("✅ 质押交易数据构建完成")
	return txData, nil
}

// BuildApproveTransaction 构建ERC20授权交易
func (cm *ContractManager) BuildApproveTransaction(req *StakeRequest) (*TransactionData, error) {
	log.Printf("🔄 构建ERC20授权交易")
	log.Printf("📋 代币: %s, 数量: %s", req.Token, req.Amount)
	
	// 获取MTK代币合约地址
	mtkToken := cm.config.Tokens["MTK"]
	if mtkToken.ContractAddress == "" {
		return nil, fmt.Errorf("MTK token contract address not found")
	}
	
	// 获取质押合约地址
	stakingContract := cm.config.Contracts["MTKStaking"]
	if stakingContract.Address == "" {
		return nil, fmt.Errorf("MTKStaking contract address not found")
	}
	
	// 解析授权金额
	amount, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %s", req.Amount)
	}
	
	// 将金额转换为 wei
	weiAmount := new(big.Int)
	weiAmount, _ = weiAmount.SetString(fmt.Sprintf("%.0f", amount*1e18), 10)
	
	// 构建交易数据
	txData := &TransactionData{
		To:       mtkToken.ContractAddress, // 发送给MTK代币合约
		Value:    "0x0",
		GasLimit: "0x1FBBF",  // 130000 gas (approve通常需要较少gas)
		GasPrice: "0x3B9ACA00", // 1 gwei
	}
	
	// approve(address spender, uint256 amount) 函数调用数据
	// 函数签名: 0x095ea7b3
	funcSig := "095ea7b3"
	
	// 第一个参数：spender地址（质押合约地址，去掉0x并填充到64位）
	spenderAddr := strings.TrimPrefix(stakingContract.Address, "0x")
	spenderAddrPadded := fmt.Sprintf("%064s", spenderAddr)
	
	// 第二个参数：授权金额（填充到64位十六进制）
	amountHex := fmt.Sprintf("%064s", weiAmount.Text(16))
	
	// 组合完整的调用数据
	txData.Data = "0x" + funcSig + spenderAddrPadded + amountHex
	
	log.Printf("📋 授权交易: %s %s 给质押合约 %s", req.Amount, req.Token, stakingContract.Address)
	log.Printf("📋 交易数据: %s", txData.Data)
	
	log.Printf("✅ 授权交易数据构建完成")
	return txData, nil
}

// GetContractInfo 获取合约信息
func (cm *ContractManager) GetContractInfo(name string) *ContractInfo {
	if contract, exists := cm.config.Contracts[name]; exists {
		return &contract
	}
	return nil
}

// GetTokenInfo 获取代币信息
func (cm *ContractManager) GetTokenInfo(symbol string) *TokenConfig {
	if token, exists := cm.config.Tokens[symbol]; exists {
		return &token
	}
	return nil
}

// GetSupportedTokens 获取支持的代币列表
func (cm *ContractManager) GetSupportedTokens() []string {
	tokens := make([]string, 0, len(cm.config.Tokens))
	for symbol := range cm.config.Tokens {
		tokens = append(tokens, symbol)
	}
	return tokens
}

// GetSupportedPairs 获取支持的交换对
func (cm *ContractManager) GetSupportedPairs() []string {
	pairs := make([]string, 0)
	for _, contract := range cm.config.Contracts {
		for _, pair := range contract.SupportedPairs {
			pairStr := fmt.Sprintf("%s-%s", pair.From, pair.To)
			pairs = append(pairs, pairStr)
		}
	}
	return pairs
}

// GetWorkflowDescription 获取工作流描述（供 LLM 使用）
func (cm *ContractManager) GetWorkflowDescription() string {
	description := fmt.Sprintf(`
合约系统信息:
- 网络: %s (Chain ID: %d)
- RPC: %s

支持的代币:
`, cm.config.Network.Name, cm.config.Network.ChainID, cm.config.Network.RPCURL)

	for symbol, token := range cm.config.Tokens {
		if token.IsNative {
			description += fmt.Sprintf("- %s (%s): 原生代币\n", symbol, token.Name)
		} else {
			description += fmt.Sprintf("- %s (%s): ERC20 代币，合约地址 %s\n", symbol, token.Name, token.ContractAddress)
		}
	}

	description += "\n支持的交换对:\n"
	for _, contract := range cm.config.Contracts {
		for _, pair := range contract.SupportedPairs {
			description += fmt.Sprintf("- %s -> %s (汇率: %.0f, 方法: %s)\n", 
				pair.From, pair.To, pair.Rate, pair.Method)
		}
	}

	description += "\n支持的操作模式:\n"
	for _, workflow := range cm.config.Workflows {
		description += fmt.Sprintf("- %s: %s\n", workflow.Description, strings.Join(workflow.Patterns, ", "))
	}

	return description
} 
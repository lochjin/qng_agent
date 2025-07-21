import React, { useState, useEffect, useRef } from 'react';
import './App.css';

function App() {
  const [messages, setMessages] = useState([]);
  const [inputMessage, setInputMessage] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [walletConnected, setWalletConnected] = useState(false);
  const [walletAddress, setWalletAddress] = useState('');
  const [currentSession, setCurrentSession] = useState(null);
  const [signatureRequest, setSignatureRequest] = useState(null);
  const [networkInfo, setNetworkInfo] = useState(null);
  const [walletError, setWalletError] = useState(null);
  const [balances, setBalances] = useState({
    meer: '0',
    mtk: '0'
  });
  const [isLoadingBalances, setIsLoadingBalances] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [config, setConfig] = useState(null);
  const [isLoadingConfig, setIsLoadingConfig] = useState(false);
  const messagesEndRef = useRef(null);
  
  // 生成唯一ID的函数
  const generateUniqueId = () => {
    return Date.now() + Math.random().toString(36).substr(2, 9);
  };

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  // MetaMask事件监听
  useEffect(() => {
    if (typeof window.ethereum !== 'undefined') {
      // 监听账户变化
      const handleAccountsChanged = (accounts) => {
        console.log('🔄 账户变化:', accounts);
        if (accounts.length === 0) {
          // 用户断开了连接
          setWalletConnected(false);
          setWalletAddress('');
          setNetworkInfo(null);
          setMessages(prev => [...prev, {
            id: Date.now(),
            type: 'system',
            content: '🔌 钱包已断开连接',
            timestamp: new Date()
          }]);
        } else {
          // 账户切换
          setWalletAddress(accounts[0]);
          // 更新余额
          updateTokenBalances(accounts[0]);
          setMessages(prev => [...prev, {
            id: Date.now(),
            type: 'system',
            content: `🔄 钱包账户已切换: ${accounts[0].slice(0, 6)}...${accounts[0].slice(-4)}`,
            timestamp: new Date()
          }]);
        }
      };

      // 监听链ID变化
      const handleChainChanged = (chainId) => {
        console.log('🔄 网络变化:', chainId);
        const networkName = getNetworkName(chainId);
        setNetworkInfo({ chainId, name: networkName });
        setMessages(prev => [...prev, {
          id: Date.now(),
          type: 'system',
          content: `🌐 网络已切换到: ${networkName}`,
          timestamp: new Date()
        }]);
      };

      // 监听连接状态
      const handleConnect = (connectInfo) => {
        console.log('🔗 钱包连接:', connectInfo);
        setWalletConnected(true);
        setWalletError(null);
      };

      const handleDisconnect = (error) => {
        console.log('🔌 钱包断开:', error);
        setWalletConnected(false);
        setWalletAddress('');
        setNetworkInfo(null);
        setWalletError(error?.message || '钱包连接已断开');
      };

      // 添加事件监听器
      window.ethereum.on('accountsChanged', handleAccountsChanged);
      window.ethereum.on('chainChanged', handleChainChanged);
      window.ethereum.on('connect', handleConnect);
      window.ethereum.on('disconnect', handleDisconnect);

      // 清理函数
      return () => {
        window.ethereum.removeListener('accountsChanged', handleAccountsChanged);
        window.ethereum.removeListener('chainChanged', handleChainChanged);
        window.ethereum.removeListener('connect', handleConnect);
        window.ethereum.removeListener('disconnect', handleDisconnect);
      };
    }
  }, []);

  // 获取网络名称
  const getNetworkName = (chainId) => {
    const networks = {
      '0x1': 'Ethereum Mainnet',
      '0x3': 'Ropsten Testnet',
      '0x4': 'Rinkeby Testnet',
      '0x5': 'Goerli Testnet',
      '0x2a': 'Kovan Testnet',
      '0x89': 'Polygon Mainnet',
      '0x13881': 'Polygon Mumbai Testnet',
      '0xa': 'Optimism',
      '0xa4b1': 'Arbitrum One',
      '0xa4ec': 'Arbitrum Nova',
      '0x38': 'BSC Mainnet',
      '0x61': 'BSC Testnet',
      '0xfa': 'Fantom Opera',
      '0xfa2': 'Fantom Testnet'
    };
    return networks[chainId] || `未知网络 (${chainId})`;
  };

  // 检查MetaMask是否可用
  const checkMetaMaskAvailability = () => {
    if (typeof window.ethereum === 'undefined') {
      throw new Error('MetaMask未安装，请先安装MetaMask扩展');
    }
    
    if (!window.ethereum.isMetaMask) {
      throw new Error('检测到非MetaMask钱包，请使用MetaMask');
    }
    
    return true;
  };

  // 格式化余额
  const formatBalance = (balance, decimals = 18) => {
    try {
      // 处理空值或无效值
      if (!balance || balance === '0x' || balance === '0x0') {
        return '0.0000';
      }
      
      const wei = BigInt(balance);
      const ether = Number(wei) / Math.pow(10, decimals);
      return ether.toFixed(4);
    } catch (error) {
      console.warn('⚠️ 余额格式化失败:', balance, error);
      return '0.0000';
    }
  };

  // 格式化地址
  const formatAddress = (address) => {
    if (!address || typeof address !== 'string') return '';
    if (address.length < 10) return address;
    return `${address.slice(0, 6)}...${address.slice(-4)}`;
  };

  // 真实API调用
  const API_BASE_URL = 'http://localhost:9090';
  const MCP_BASE_URL = 'http://localhost:9091';

  // 配置相关API调用
  const fetchConfig = async () => {
    console.log('🔗 获取配置:', `${API_BASE_URL}/api/config`);
    
    const response = await fetch(`${API_BASE_URL}/api/config`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      }
    });

    if (!response.ok) {
      throw new Error(`获取配置失败: ${response.status} ${response.statusText}`);
    }

    const data = await response.json();
    console.log('📡 配置响应:', data);
    return data;
  };

  const updateConfig = async (newConfig) => {
    console.log('🔗 更新配置:', `${API_BASE_URL}/api/config`);
    
    const response = await fetch(`${API_BASE_URL}/api/config`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(newConfig)
    });

    if (!response.ok) {
      throw new Error(`更新配置失败: ${response.status} ${response.statusText}`);
    }

    const data = await response.json();
    console.log('📡 配置更新响应:', data);
    return data;
  };

  const callAgentAPI = async (message) => {
    console.log('🔗 调用智能体API:', `${API_BASE_URL}/api/agent/process`);
    
    const response = await fetch(`${API_BASE_URL}/api/agent/process`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ message })
    });

    if (!response.ok) {
      throw new Error(`API调用失败: ${response.status} ${response.statusText}`);
    }

    const data = await response.json();
    console.log('📡 智能体API响应:', data);
    return data;
  };

  const pollWorkflowStatus = async (sessionId) => {
    console.log('🔄 轮询工作流状态:', sessionId);
    
    const response = await fetch(`${API_BASE_URL}/api/agent/poll/${sessionId}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      }
    });

    if (!response.ok) {
      throw new Error(`轮询失败: ${response.status} ${response.statusText}`);
    }

    const data = await response.json();
    console.log('📊 工作流状态:', data);
    return data;
  };

  const submitSignature = async (sessionId, signature) => {
    console.log('✍️ 提交签名:', sessionId);
    
    const response = await fetch(`${API_BASE_URL}/api/agent/signature`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ 
        session_id: sessionId, 
        signature: signature 
      })
    });

    if (!response.ok) {
      throw new Error(`签名提交失败: ${response.status} ${response.statusText}`);
    }

    const data = await response.json();
    console.log('✅ 签名提交响应:', data);
    return data;
  };

  const connectWallet = async () => {
    console.log('🔗 连接MetaMask钱包...');
    setWalletError(null);
    
    try {
      // 检查MetaMask是否可用
      checkMetaMaskAvailability();

      // 检查是否已经连接
      const accounts = await window.ethereum.request({ 
        method: 'eth_accounts' 
      });

      if (accounts.length > 0) {
        // 已经连接，直接获取信息
        console.log('✅ 钱包已连接，获取账户信息...');
        // 修复：返回 updateWalletInfo 的结果
        return await updateWalletInfo(accounts[0]);
      }

      // 请求连接钱包
      console.log('🔐 请求用户授权连接钱包...');
      const newAccounts = await window.ethereum.request({ 
        method: 'eth_requestAccounts' 
      });

      if (newAccounts.length === 0) {
        throw new Error('用户拒绝了钱包连接请求');
      }

      // 修复：返回 updateWalletInfo 的结果
      return await updateWalletInfo(newAccounts[0]);
      
    } catch (error) {
      console.error('❌ 钱包连接失败:', error);
      setWalletError(error.message);
      throw error;
    }
  };

  const updateWalletInfo = async (address) => {
    try {
      // 获取网络信息
      const chainId = await window.ethereum.request({ 
        method: 'eth_chainId' 
      });
      
      // 获取MEER余额（原生代币）
      const balance = await window.ethereum.request({
        method: 'eth_getBalance',
        params: [address, 'latest']
      });

      const networkName = getNetworkName(chainId);
      const formattedBalance = formatBalance(balance);

      setWalletConnected(true);
      setWalletAddress(address);
      setNetworkInfo({
        chainId,
        name: networkName,
        balance: formattedBalance
      });

      // 获取代币余额
      await updateTokenBalances(address);
      
      console.log('✅ 钱包信息更新成功:', {
        address,
        chainId,
        networkName,
        balance: formattedBalance
      });
      
      return {
        connected: true,
        address,
        network: networkName,
        chain_id: chainId,
        balance: formattedBalance
      };
    } catch (error) {
      console.error('❌ 更新钱包信息失败:', error);
      throw error;
    }
  };

  // 获取代币余额
  const updateTokenBalances = async (address) => {
    setIsLoadingBalances(true);
    try {
      console.log('💰 获取代币余额...');
      
      // MTK 合约地址（从部署信息获取）
      const MTK_CONTRACT_ADDRESS = '0x1859Bd4e1d2Ba470b1E6D9C8d14dF785e533E3A0';
      
      // 获取MEER余额（原生代币）
      const meerBalance = await window.ethereum.request({
        method: 'eth_getBalance',
        params: [address, 'latest']
      });

      // 获取MTK余额（ERC20代币）
      // balanceOf(address) 函数调用数据
      const balanceOfSelector = '0x70a08231'; // balanceOf(address) 函数签名
      const paddedAddress = address.slice(2).padStart(64, '0'); // 去掉0x并填充到64位
      const callData = balanceOfSelector + paddedAddress;

      const mtkBalance = await window.ethereum.request({
        method: 'eth_call',
        params: [{
          to: MTK_CONTRACT_ADDRESS,
          data: callData
        }, 'latest']
      });

      // 格式化余额
      const formattedMeerBalance = formatBalance(meerBalance);
      const formattedMtkBalance = formatBalance(mtkBalance || '0x0');

      setBalances({
        meer: formattedMeerBalance,
        mtk: formattedMtkBalance
      });

      console.log('✅ 余额获取成功:', {
        meer: formattedMeerBalance,
        mtk: formattedMtkBalance
      });

    } catch (error) {
      console.error('❌ 获取余额失败:', error);
      setBalances({ meer: '0', mtk: '0' });
    } finally {
      setIsLoadingBalances(false);
    }
  };

  // 手动刷新余额
  const refreshBalances = async () => {
    if (walletAddress) {
      await updateTokenBalances(walletAddress);
    }
  };

  // 查询指定代币余额
  const queryBalance = async (tokenSymbol) => {
    if (!walletConnected || !walletAddress) {
      return {
        success: false,
        message: '请先连接钱包'
      };
    }

    try {
      setIsLoadingBalances(true);
      console.log(`💰 查询 ${tokenSymbol} 余额...`);

      let balance = '0';
      const upperSymbol = tokenSymbol.toUpperCase();

      if (upperSymbol === 'ETH' || upperSymbol === 'MEER') {
        // 查询原生代币余额
        const nativeBalance = await window.ethereum.request({
          method: 'eth_getBalance',
          params: [walletAddress, 'latest']
        });
        balance = formatBalance(nativeBalance);
        
        // 更新状态
        setBalances(prev => ({
          ...prev,
          meer: balance
        }));

      } else if (upperSymbol === 'MTK') {
        // 查询MTK代币余额
        const MTK_CONTRACT_ADDRESS = '0x1859Bd4e1d2Ba470b1E6D9C8d14dF785e533E3A0';
        const balanceOfSelector = '0x70a08231';
        const paddedAddress = walletAddress.slice(2).padStart(64, '0');
        const callData = balanceOfSelector + paddedAddress;

        const mtkBalance = await window.ethereum.request({
          method: 'eth_call',
          params: [{
            to: MTK_CONTRACT_ADDRESS,
            data: callData
          }, 'latest']
        });

        balance = formatBalance(mtkBalance || '0x0');
        
        // 更新状态
        setBalances(prev => ({
          ...prev,
          mtk: balance
        }));
      } else {
        return {
          success: false,
          message: `暂不支持查询 ${tokenSymbol} 代币余额`
        };
      }

      console.log(`✅ ${upperSymbol} 余额查询成功: ${balance}`);
      
      return {
        success: true,
        symbol: upperSymbol,
        balance: balance,
        address: walletAddress,
        message: `${upperSymbol} 余额: ${balance} ${upperSymbol}`
      };

    } catch (error) {
      console.error(`❌ 查询 ${tokenSymbol} 余额失败:`, error);
      return {
        success: false,
        message: `查询 ${tokenSymbol} 余额失败: ${error.message}`
      };
    } finally {
      setIsLoadingBalances(false);
    }
  };

  const switchNetwork = async (targetChainId) => {
    try {
      console.log(`🔄 切换到网络: ${targetChainId}`);
      
      await window.ethereum.request({
        method: 'wallet_switchEthereumChain',
        params: [{ chainId: targetChainId }],
      });
      
      console.log('✅ 网络切换成功');
    } catch (switchError) {
      // 如果网络不存在，尝试添加网络
      if (switchError.code === 4902) {
        console.log('➕ 网络不存在，尝试添加网络...');
        await addNetwork(targetChainId);
      } else {
        throw switchError;
      }
    }
  };

  const addNetwork = async (chainId) => {
    const networkConfigs = {
      '0x1': {
        chainId: '0x1',
        chainName: 'Ethereum Mainnet',
        nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
        rpcUrls: ['https://mainnet.infura.io/v3/'],
        blockExplorerUrls: ['https://etherscan.io']
      },
      '0x89': {
        chainId: '0x89',
        chainName: 'Polygon Mainnet',
        nativeCurrency: { name: 'MATIC', symbol: 'MATIC', decimals: 18 },
        rpcUrls: ['https://polygon-rpc.com'],
        blockExplorerUrls: ['https://polygonscan.com']
      },
      '0x38': {
        chainId: '0x38',
        chainName: 'BSC Mainnet',
        nativeCurrency: { name: 'BNB', symbol: 'BNB', decimals: 18 },
        rpcUrls: ['https://bsc-dataseed.binance.org'],
        blockExplorerUrls: ['https://bscscan.com']
      }
    };

    const config = networkConfigs[chainId];
    if (!config) {
      throw new Error(`不支持的网络: ${chainId}`);
    }

    await window.ethereum.request({
      method: 'wallet_addEthereumChain',
      params: [config],
    });
  };

  const disconnectWallet = () => {
    setWalletConnected(false);
    setWalletAddress('');
    setNetworkInfo(null);
    setWalletError(null);
    setMessages(prev => [...prev, {
      id: Date.now(),
      type: 'system',
      content: '🔌 钱包已断开连接',
      timestamp: new Date()
    }]);
  };

  const handleSendMessage = async () => {
    if (!inputMessage.trim()) return;

    const userMessage = inputMessage;
    setInputMessage('');
    setIsLoading(true);

    // 添加用户消息
    setMessages(prev => [...prev, {
      id: Date.now(),
      type: 'user',
      content: userMessage,
      timestamp: new Date()
    }]);

    try {
      // 检查是否是余额查询请求
      const balanceQueryPattern = /查询|余额|balance/i;
      const tokenPattern = /(ETH|MEER|MTK|eth|meer|mtk)/i;
      
      if (balanceQueryPattern.test(userMessage) && tokenPattern.test(userMessage)) {
        console.log('🔍 检测到余额查询请求');
        
        // 提取代币符号
        const tokenMatch = userMessage.match(tokenPattern);
        const tokenSymbol = tokenMatch ? tokenMatch[1] : 'ETH';
        
        // 添加处理消息
        setMessages(prev => [...prev, {
          id: Date.now() + 1,
          type: 'system', 
          content: `🔍 正在查询 ${tokenSymbol.toUpperCase()} 余额...`,
          timestamp: new Date()
        }]);

        // 查询余额
        const result = await queryBalance(tokenSymbol);
        
        if (result.success) {
          setMessages(prev => [...prev, {
            id: Date.now() + 2,
            type: 'assistant',
            content: `💰 ${result.message}\n📋 钱包地址: ${formatAddress(result.address)}`,
            timestamp: new Date()
          }]);
        } else {
          setMessages(prev => [...prev, {
            id: Date.now() + 2,
            type: 'system',
            content: `❌ ${result.message}`,
            timestamp: new Date()
          }]);
        }
        
        setIsLoading(false);
        return; // 直接返回，不继续执行后续流程
      }
      // 1. 调用智能体API
      const execution = await callAgentAPI(userMessage);
      setCurrentSession(execution);

      // 添加系统消息
      setMessages(prev => [...prev, {
        id: Date.now() + 1,
        type: 'system',
        content: '🔄 正在分析您的请求...',
        timestamp: new Date()
      }]);

      // 2. 轮询工作流状态
      let status;
      let pollCount = 0;
      const maxPolls = 30; // 最多轮询30次
      
      // 使用 workflow_id 而不是 session_id
      const workflowId = execution.workflow_id || execution.session_id;
      if (!workflowId) {
        throw new Error('未收到有效的工作流ID');
      }
      
      while (pollCount < maxPolls) {
        status = await pollWorkflowStatus(workflowId);
        pollCount++;
        
        console.log(`📊 第${pollCount}次轮询状态:`, status);
        
        if (status.status === 'completed' || status.status === 'failed' || status.need_signature) {
          break;
        }
        
        // 等待2秒后继续轮询
        await new Promise(resolve => setTimeout(resolve, 2000));
      }
      
      if (status.need_signature) {
        console.log('📝 签名请求数据:', status.signature_request);
        setSignatureRequest(status.signature_request);
        setMessages(prev => [...prev, {
          id: Date.now() + 2,
          type: 'system',
          content: '✍️ 需要您签名授权交易，请点击下方按钮进行签名',
          timestamp: new Date(),
          requiresSignature: true
        }]);
      } else if (status.status === 'completed') {
        setMessages(prev => [...prev, {
          id: Date.now() + 2,
          type: 'system',
          content: '✅ 工作流执行完成！',
          timestamp: new Date()
        }]);
      } else if (status.status === 'failed') {
        setMessages(prev => [...prev, {
          id: Date.now() + 2,
          type: 'error',
          content: `❌ 工作流执行失败: ${status.error || '未知错误'}`,
          timestamp: new Date()
        }]);
      }

    } catch (error) {
      console.error('❌ 处理失败:', error);
      setMessages(prev => [...prev, {
        id: Date.now() + 3,
        type: 'error',
        content: `❌ 处理失败: ${error.message}`,
        timestamp: new Date()
      }]);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSignature = async () => {
    if (!signatureRequest) return;

    setIsLoading(true);

    try {
      console.log('✍️ 开始签名流程...');
      console.log('📋 完整签名请求:', signatureRequest);
      
      // 使用MetaMask进行签名
      if (typeof window.ethereum === 'undefined') {
        throw new Error('MetaMask未安装');
      }

      // 检查签名请求数据完整性
      if (!signatureRequest) {
        throw new Error('签名请求数据为空');
      }

      // 获取当前连接的账户地址
      const accounts = await window.ethereum.request({ method: 'eth_accounts' });
      if (!accounts || accounts.length === 0) {
        throw new Error('请先连接 MetaMask 钱包');
      }
      const fromAddress = accounts[0];
      console.log('📋 发送方地址:', fromAddress);

      // 构建交易数据
      const transactionData = {
        from: fromAddress, // 添加发送方地址
        to: signatureRequest.to_address || signatureRequest.ToAddress,
        value: signatureRequest.value || signatureRequest.Value || '0x0',
        data: signatureRequest.data || signatureRequest.Data || '0x',
        gas: signatureRequest.gas_limit || signatureRequest.GasLimit || '0x186A0', // 100000 gas
        gasPrice: signatureRequest.gas_price || signatureRequest.GasPrice || '0x3B9ACA00' // 1 gwei
      };

      console.log('📝 交易数据:', transactionData);
      
      // 验证必需字段
      if (!transactionData.to) {
        throw new Error('缺少交易目标地址 (to)');
      }
      if (!transactionData.from) {
        throw new Error('缺少发送方地址 (from)');
      }

      // 检查MetaMask状态
      console.log('🔍 检查MetaMask状态...');
      const isUnlocked = await window.ethereum._metamask.isUnlocked();
      console.log('🔐 MetaMask解锁状态:', isUnlocked);
      
      if (!isUnlocked) {
        console.log('🔒 MetaMask被锁定，请解锁后重试');
        throw new Error('MetaMask被锁定，请解锁后重试');
      }

      // 检查当前网络
      const currentChainId = await window.ethereum.request({ method: 'eth_chainId' });
      console.log('🌐 当前网络Chain ID:', currentChainId);
      
      // 检查网络是否正确（这里应该根据你的实际网络配置）
      const expectedChainId = '0x1FC6'; // 8134 in hex，你的自定义网络
      const normalizedCurrentChainId = currentChainId.toLowerCase();
      const normalizedExpectedChainId = expectedChainId.toLowerCase();
      
      if (normalizedCurrentChainId !== normalizedExpectedChainId) {
        console.log(`⚠️ 网络不匹配! 当前: ${currentChainId}, 期望: ${expectedChainId}`);
        setMessages(prev => [...prev, {
          id: generateUniqueId(),
          type: 'error',
          content: `⚠️ 请切换到正确的网络 (Chain ID: ${expectedChainId})，当前网络: ${currentChainId}`,
          timestamp: new Date()
        }]);
        throw new Error(`网络不匹配，请切换到 Chain ID: ${expectedChainId}`);
      } else {
        console.log('✅ 网络匹配成功:', currentChainId);
      }

      // 请求用户授权（确保MetaMask获得焦点）
      console.log('🚀 发起MetaMask签名请求...');
      console.log('📋 请求参数:', JSON.stringify(transactionData, null, 2));
      
      // 尝试不同的方法来确保弹窗显示
      let signature;
      try {
        // 方法1: 使用 eth_sendTransaction
        signature = await window.ethereum.request({
          method: 'eth_sendTransaction',
          params: [transactionData]
        });
      } catch (sendError) {
        console.log('❌ eth_sendTransaction 失败:', sendError);
        
        // 方法2: 尝试使用 personal_sign 作为备选
        if (sendError.code === -32603 || sendError.code === 4001) {
          console.log('🔄 尝试alternative方法...');
          throw sendError; // 直接抛出原始错误
        } else {
          throw sendError;
        }
      }

      console.log('✅ 交易签名成功:', signature);
      
      // 提交签名到后端
      const sessionId = currentSession.workflow_id || currentSession.session_id;
      const result = await submitSignature(sessionId, signature);
      
      setMessages(prev => [...prev, {
        id: Date.now(),
        type: 'system',
        content: '✅ 签名已提交，交易正在处理中...',
        timestamp: new Date()
      }]);

      // 轮询交易状态和后续签名请求
      let pollCount = 0;
      const maxPolls = 30; // 增加轮询次数
      let workflowCompleted = false;
      
      while (pollCount < maxPolls && !workflowCompleted) {
        const status = await pollWorkflowStatus(sessionId);
        pollCount++;
        
        console.log(`📊 第${pollCount}次轮询状态:`, status);
        
        if (status.status === 'completed') {
          setMessages(prev => [...prev, {
            id: Date.now() + 1,
            type: 'system',
            content: `🎉 工作流完成！最后交易哈希: ${signature}`,
            timestamp: new Date()
          }]);
          workflowCompleted = true;
          break;
        } else if (status.status === 'failed') {
          setMessages(prev => [...prev, {
            id: Date.now() + 1,
            type: 'error',
            content: `❌ 工作流失败: ${status.error || '未知错误'}`,
            timestamp: new Date()
          }]);
          workflowCompleted = true;
          break;
        } else if (status.need_signature) {
          // 检测到新的签名请求
          console.log('🔔 检测到新的签名请求:', status.signature_request);
          setMessages(prev => [...prev, {
            id: Date.now() + 1,
            type: 'system',
            content: `✅ 第一步完成！现在需要签名第二步操作...`,
            timestamp: new Date()
          }]);
          
          // 更新签名请求状态，触发新的签名流程
          setSignatureRequest(status.signature_request);
          return; // 返回，等待用户处理新的签名请求
        }
        
        // 等待3秒后继续轮询
        await new Promise(resolve => setTimeout(resolve, 3000));
      }

      // 只有在工作流完全完成时才清空状态
      if (workflowCompleted) {
        setSignatureRequest(null);
        setCurrentSession(null);
      }

    } catch (error) {
      console.error('❌ 签名失败:', error);
      setMessages(prev => [...prev, {
        id: Date.now(),
        type: 'error',
        content: `❌ 签名失败: ${error.message}`,
        timestamp: new Date()
      }]);
    } finally {
      setIsLoading(false);
    }
  };

  const handleConnectWallet = async () => {
    try {
      const walletInfo = await connectWallet();
      setMessages(prev => [...prev, {
        id: Date.now(),
        type: 'system',
        content: `🔗 钱包连接成功！地址: ${formatAddress(walletInfo.address)}`,
        timestamp: new Date()
      }]);
    } catch (error) {
      setMessages(prev => [...prev, {
        id: Date.now(),
        type: 'error',
        content: `❌ 钱包连接失败: ${error.message}`,
        timestamp: new Date()
      }]);
    }
  };

  const handleSwitchNetwork = async (chainId) => {
    try {
      await switchNetwork(chainId);
      setMessages(prev => [...prev, {
        id: Date.now(),
        type: 'system',
        content: `🌐 网络切换成功！`,
        timestamp: new Date()
      }]);
    } catch (error) {
      setMessages(prev => [...prev, {
        id: Date.now(),
        type: 'error',
        content: `❌ 网络切换失败: ${error.message}`,
        timestamp: new Date()
      }]);
    }
  };

  const handleKeyPress = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSendMessage();
    }
  };

  // 配置相关处理函数
  const handleOpenSettings = async () => {
    setShowSettings(true);
    setIsLoadingConfig(true);
    
    try {
      const configData = await fetchConfig();
      setConfig(configData);
    } catch (error) {
      console.error('❌ 获取配置失败:', error);
      setMessages(prev => [...prev, {
        id: Date.now(),
        type: 'error',
        content: `❌ 获取配置失败: ${error.message}`,
        timestamp: new Date()
      }]);
    } finally {
      setIsLoadingConfig(false);
    }
  };

  const handleCloseSettings = () => {
    setShowSettings(false);
    setConfig(null);
  };

  const handleSaveConfig = async (newConfig) => {
    setIsLoadingConfig(true);
    
    try {
      await updateConfig(newConfig);
      setConfig(newConfig);
      setMessages(prev => [...prev, {
        id: Date.now(),
        type: 'system',
        content: '✅ 配置保存成功！',
        timestamp: new Date()
      }]);
    } catch (error) {
      console.error('❌ 保存配置失败:', error);
      setMessages(prev => [...prev, {
        id: Date.now(),
        type: 'error',
        content: `❌ 保存配置失败: ${error.message}`,
        timestamp: new Date()
      }]);
    } finally {
      setIsLoadingConfig(false);
    }
  };

  return (
    <div className="app">
      {/* Debug Panel */}
      <div className="debug-panel">
        <h3>🔍 调试面板</h3>
        <div className="debug-info">
          <div>API状态: <span className="status-connected">已连接</span></div>
          <div>后端地址: {API_BASE_URL}</div>
          <div>MCP地址: {MCP_BASE_URL}</div>
          <div>钱包状态: {walletConnected ? '已连接' : '未连接'}</div>
          {currentSession && (
            <div>会话ID: {currentSession.session_id}</div>
          )}
        </div>
      </div>

      <div className="header">
        <div className="header-left">
          <h1>🤖 QNG 智能体</h1>
        </div>
        <div className="header-right">
          <button 
            className="settings-btn"
            onClick={handleOpenSettings}
            title="设置"
          >
            ⚙️ 设置
          </button>
        </div>
        <div className="wallet-info">
          {walletConnected ? (
            <div className="wallet-details">
              <div className="wallet-address">
                🔗 {walletAddress ? formatAddress(walletAddress) : '连接中...'}
              </div>
              {networkInfo && (
                <div className="network-info">
                  <span className="network-name">{networkInfo.name}</span>
                </div>
              )}
              
              {/* 代币余额显示 */}
              <div className="wallet-balances">
                <div className="balance-row">
                  <span className="token-name">💎 MEER:</span>
                  <span className="balance-amount">
                    {isLoadingBalances ? '加载中...' : `${balances.meer} MEER`}
                  </span>
                  <button 
                    className="query-balance-btn"
                    onClick={() => queryBalance('MEER')}
                    disabled={isLoadingBalances}
                    title="查询MEER余额"
                  >
                    🔍
                  </button>
                </div>
                <div className="balance-row">
                  <span className="token-name">🪙 MTK:</span>
                  <span className="balance-amount">
                    {isLoadingBalances ? '加载中...' : `${balances.mtk} MTK`}
                  </span>
                  <button 
                    className="query-balance-btn"
                    onClick={() => queryBalance('MTK')}
                    disabled={isLoadingBalances}
                    title="查询MTK余额"
                  >
                    🔍
                  </button>
                </div>
                <div className="balance-actions">
                  <button 
                    className="refresh-balance-btn"
                    onClick={refreshBalances}
                    disabled={isLoadingBalances}
                    title="刷新所有余额"
                  >
                    {isLoadingBalances ? '🔄' : '🔄 刷新'}
                  </button>
                  <button 
                    className="query-balance-btn"
                    onClick={() => queryBalance('ETH')}
                    disabled={isLoadingBalances}
                    title="查询ETH余额"
                  >
                    💎 ETH
                  </button>
                </div>
              </div>

              <div className="wallet-actions">
                <button 
                  className="switch-network-btn"
                  onClick={() => handleSwitchNetwork('0x1')}
                  title="切换到以太坊主网"
                >
                  🌐 ETH
                </button>
                <button 
                  className="switch-network-btn"
                  onClick={() => handleSwitchNetwork('0x89')}
                  title="切换到Polygon"
                >
                  🌐 POLYGON
                </button>
                <button 
                  className="disconnect-btn"
                  onClick={disconnectWallet}
                  title="断开连接"
                >
                  🔌
                </button>
              </div>
            </div>
          ) : (
            <div className="connect-section">
              <button 
                className="connect-btn"
                onClick={handleConnectWallet}
                disabled={isLoading}
              >
                🔗 连接钱包
              </button>
              {walletError && (
                <div className="wallet-error">
                  ❌ {walletError}
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      <div className="chat-container">
        {!walletConnected && (
          <div className="wallet-connect-prompt">
            <div className="wallet-status disconnected">
              🔌 钱包未连接
            </div>
            <p>请先连接MetaMask钱包以使用智能体功能</p>
            <button 
              className="big-connect-btn"
              onClick={handleConnectWallet}
              disabled={isLoading}
            >
              🔗 连接 MetaMask 钱包
            </button>
            <div className="wallet-instructions">
              <h4>📋 连接步骤：</h4>
              <ol>
                <li>确保已安装MetaMask浏览器扩展</li>
                <li>点击上方"连接钱包"按钮</li>
                <li>在MetaMask弹窗中授权连接</li>
                <li>选择要使用的账户</li>
              </ol>
            </div>
          </div>
        )}
        
        <div className="messages">
          {messages.map((message) => (
            <div key={message.id} className={`message ${message.type}`}>
              <div className="message-content">
                {message.content}
              </div>
              <div className="message-time">
                {message.timestamp.toLocaleTimeString()}
              </div>
            </div>
          ))}
          {isLoading && (
            <div className="message system">
              <div className="message-content">
                <div className="loading">
                  <span>⏳</span>
                  <span>处理中</span>
                  <span>...</span>
                </div>
              </div>
            </div>
          )}
          <div ref={messagesEndRef} />
        </div>

        {signatureRequest && (
          <div className="signature-request">
            <h3>✍️ 交易签名请求</h3>
            <div className="signature-details">
              <p><strong>操作:</strong> {signatureRequest.action}</p>
              <p><strong>从:</strong> {signatureRequest.from_token}</p>
              <p><strong>到:</strong> {signatureRequest.to_token}</p>
              <p><strong>数量:</strong> {signatureRequest.amount}</p>
              <p><strong>Gas费:</strong> {signatureRequest.gas_fee}</p>
              <p><strong>滑点:</strong> {signatureRequest.slippage}</p>
              <p><strong>合约地址:</strong> {signatureRequest.to_address}</p>
              <p><strong>交易值:</strong> {signatureRequest.value}</p>
            </div>
            
            <div className="signature-actions">
              <button 
                className="signature-btn primary"
                onClick={handleSignature}
                disabled={isLoading}
              >
                {isLoading ? '⏳ 等待签名...' : '🔐 确认签名'}
              </button>
              
              <button 
                className="signature-btn secondary"
                onClick={async () => {
                  console.log('🔄 手动触发MetaMask...');
                  try {
                    // 先检查MetaMask状态
                    const accounts = await window.ethereum.request({ method: 'eth_accounts' });
                    console.log('👤 当前账户:', accounts);
                    
                    // 手动请求权限
                    await window.ethereum.request({ method: 'eth_requestAccounts' });
                    console.log('✅ 权限已获取，请点击确认签名按钮');
                    
                    setMessages(prev => [...prev, {
                      id: generateUniqueId(),
                      type: 'system',
                      content: '🔄 已重新获取MetaMask权限，请点击"确认签名"按钮',
                      timestamp: new Date()
                    }]);
                  } catch (error) {
                    console.error('❌ 获取权限失败:', error);
                    setMessages(prev => [...prev, {
                      id: generateUniqueId(),
                      type: 'error',
                      content: `❌ 获取MetaMask权限失败: ${error.message}`,
                      timestamp: new Date()
                    }]);
                  }
                }}
                disabled={isLoading}
              >
                🔄 重新唤醒MetaMask
              </button>
            </div>
            
            <div className="signature-tips">
              <h4>💡 签名提示：</h4>
              <ul>
                <li>确保MetaMask已解锁</li>
                <li>检查浏览器是否阻止了弹窗</li>
                <li>如果没有弹窗，请点击"重新唤醒MetaMask"</li>
                <li>确认当前网络正确</li>
              </ul>
            </div>
          </div>
        )}

        <div className="input-container">
          <textarea
            value={inputMessage}
            onChange={(e) => setInputMessage(e.target.value)}
            onKeyPress={handleKeyPress}
            placeholder="输入您的请求，例如：我需要将1000USDT兑换成BTC"
            disabled={isLoading}
          />
          <button 
            onClick={handleSendMessage}
            disabled={isLoading || !inputMessage.trim()}
            className="send-btn"
          >
            发送
          </button>
        </div>
      </div>

      {/* 设置弹窗 */}
      {showSettings && (
        <SettingsModal 
          config={config}
          isLoading={isLoadingConfig}
          onClose={handleCloseSettings}
          onSave={handleSaveConfig}
        />
      )}
    </div>
  );
}

// 设置弹窗组件
function SettingsModal({ config, isLoading, onClose, onSave }) {
  const [formData, setFormData] = useState({
    llm: {
      provider: 'openai',
      openai: {
        api_key: '',
        model: 'gpt-4',
        base_url: 'https://api.openai.com/v1',
        timeout: 30,
        max_tokens: 2000
      },
      gemini: {
        api_key: '',
        model: 'gemini-1.5-flash',
        timeout: 30
      },
      anthropic: {
        api_key: '',
        model: 'claude-3-5-sonnet-20241022',
        timeout: 30
      }
    },
    mcp: {
      host: 'localhost',
      port: 8081,
      timeout: 30,
      qng: {
        enabled: true,
        host: 'localhost',
        port: 8082,
        timeout: 30
      },
      metamask: {
        enabled: true,
        host: 'localhost',
        port: 8083,
        timeout: 30
      }
    }
  });

  // 初始化表单数据
  useEffect(() => {
    if (config) {
      // 深度合并配置数据
      setFormData(prevData => {
        const newData = JSON.parse(JSON.stringify(prevData)); // 深拷贝
        
        // 合并LLM配置
        if (config.LLM || config.llm) {
          const llmConfig = config.LLM || config.llm;
          newData.llm.provider = llmConfig.Provider || llmConfig.provider || newData.llm.provider;
          
          if (llmConfig.OpenAI || llmConfig.openai) {
            const openaiConfig = llmConfig.OpenAI || llmConfig.openai;
            newData.llm.openai = {
              ...newData.llm.openai,
              api_key: openaiConfig.APIKey || openaiConfig.api_key || newData.llm.openai.api_key,
              model: openaiConfig.Model || openaiConfig.model || newData.llm.openai.model,
              base_url: openaiConfig.BaseURL || openaiConfig.base_url || newData.llm.openai.base_url,
              timeout: openaiConfig.Timeout || openaiConfig.timeout || newData.llm.openai.timeout,
              max_tokens: openaiConfig.MaxTokens || openaiConfig.max_tokens || newData.llm.openai.max_tokens
            };
          }
          
          if (llmConfig.Gemini || llmConfig.gemini) {
            const geminiConfig = llmConfig.Gemini || llmConfig.gemini;
            newData.llm.gemini = {
              ...newData.llm.gemini,
              api_key: geminiConfig.APIKey || geminiConfig.api_key || newData.llm.gemini.api_key,
              model: geminiConfig.Model || geminiConfig.model || newData.llm.gemini.model,
              timeout: geminiConfig.Timeout || geminiConfig.timeout || newData.llm.gemini.timeout
            };
          }
          
          if (llmConfig.Anthropic || llmConfig.anthropic) {
            const anthropicConfig = llmConfig.Anthropic || llmConfig.anthropic;
            newData.llm.anthropic = {
              ...newData.llm.anthropic,
              api_key: anthropicConfig.APIKey || anthropicConfig.api_key || newData.llm.anthropic.api_key,
              model: anthropicConfig.Model || anthropicConfig.model || newData.llm.anthropic.model,
              timeout: anthropicConfig.Timeout || anthropicConfig.timeout || newData.llm.anthropic.timeout
            };
          }
        }
        
        // 合并MCP配置
        if (config.MCP || config.mcp) {
          const mcpConfig = config.MCP || config.mcp;
          newData.mcp = {
            ...newData.mcp,
            host: mcpConfig.Host || mcpConfig.host || newData.mcp.host,
            port: mcpConfig.Port || mcpConfig.port || newData.mcp.port,
            timeout: mcpConfig.Timeout || mcpConfig.timeout || newData.mcp.timeout
          };
          
          if (mcpConfig.QNG || mcpConfig.qng) {
            const qngConfig = mcpConfig.QNG || mcpConfig.qng;
            newData.mcp.qng = {
              ...newData.mcp.qng,
              enabled: qngConfig.Enabled !== undefined ? qngConfig.Enabled : (qngConfig.enabled !== undefined ? qngConfig.enabled : newData.mcp.qng.enabled),
              host: qngConfig.Host || qngConfig.host || newData.mcp.qng.host,
              port: qngConfig.Port || qngConfig.port || newData.mcp.qng.port,
              timeout: qngConfig.Timeout || qngConfig.timeout || newData.mcp.qng.timeout
            };
          }
          
          if (mcpConfig.MetaMask || mcpConfig.metamask) {
            const metamaskConfig = mcpConfig.MetaMask || mcpConfig.metamask;
            newData.mcp.metamask = {
              ...newData.mcp.metamask,
              enabled: metamaskConfig.Enabled !== undefined ? metamaskConfig.Enabled : (metamaskConfig.enabled !== undefined ? metamaskConfig.enabled : newData.mcp.metamask.enabled),
              host: metamaskConfig.Host || metamaskConfig.host || newData.mcp.metamask.host,
              port: metamaskConfig.Port || metamaskConfig.port || newData.mcp.metamask.port,
              timeout: metamaskConfig.Timeout || metamaskConfig.timeout || newData.mcp.metamask.timeout
            };
          }
        }
        
        return newData;
      });
    }
  }, [config]);

  const handleInputChange = (path, value) => {
    setFormData(prevData => {
      const newData = { ...prevData };
      const keys = path.split('.');
      let current = newData;
      
      for (let i = 0; i < keys.length - 1; i++) {
        if (!current[keys[i]]) {
          current[keys[i]] = {};
        }
        current = current[keys[i]];
      }
      
      current[keys[keys.length - 1]] = value;
      return newData;
    });
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    console.log('🔧 提交配置数据:', JSON.stringify(formData, null, 2));
    
    // 转换为后端期望的格式（大写首字母）
    const backendFormat = {
      LLM: {
        Provider: formData.llm.provider,
        OpenAI: {
          APIKey: formData.llm.openai.api_key,
          Model: formData.llm.openai.model,
          BaseURL: formData.llm.openai.base_url,
          Timeout: formData.llm.openai.timeout,
          MaxTokens: formData.llm.openai.max_tokens
        },
        Gemini: {
          APIKey: formData.llm.gemini.api_key,
          Model: formData.llm.gemini.model,
          Timeout: formData.llm.gemini.timeout
        },
        Anthropic: {
          APIKey: formData.llm.anthropic.api_key,
          Model: formData.llm.anthropic.model,
          Timeout: formData.llm.anthropic.timeout
        }
      },
      MCP: {
        Host: formData.mcp.host,
        Port: formData.mcp.port,
        Timeout: formData.mcp.timeout,
        QNG: {
          Enabled: formData.mcp.qng.enabled,
          Host: formData.mcp.qng.host,
          Port: formData.mcp.qng.port,
          Timeout: formData.mcp.qng.timeout
        },
        MetaMask: {
          Enabled: formData.mcp.metamask.enabled,
          Host: formData.mcp.metamask.host,
          Port: formData.mcp.metamask.port,
          Timeout: formData.mcp.metamask.timeout
        }
      }
    };
    
    console.log('📤 发送到后端的数据:', JSON.stringify(backendFormat, null, 2));
    onSave(backendFormat);
  };

  if (isLoading && !config) {
    return (
      <div className="modal-overlay">
        <div className="modal-content settings-modal">
          <div className="loading">
            <span>⏳</span>
            <span>加载配置中</span>
            <span>...</span>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="modal-overlay">
      <div className="modal-content settings-modal">
        <div className="modal-header">
          <h2>⚙️ 系统设置</h2>
          <button className="modal-close" onClick={onClose}>×</button>
        </div>
        
        <form onSubmit={handleSubmit} className="settings-form">
          <div className="settings-section">
            <h3>🤖 LLM 配置</h3>
            
            {/* LLM Provider 选择 */}
            <div className="form-group">
              <label>提供商:</label>
              <select 
                value={formData.llm?.provider || 'openai'}
                onChange={(e) => handleInputChange('llm.provider', e.target.value)}
              >
                <option value="openai">OpenAI</option>
                <option value="gemini">Google Gemini</option>
                <option value="anthropic">Anthropic Claude</option>
              </select>
            </div>

            {/* OpenAI 配置 */}
            {formData.llm?.provider === 'openai' && (
              <div className="provider-config">
                <h4>🔮 OpenAI 配置</h4>
                <div className="form-group">
                  <label>API Key:</label>
                  <input
                    type="password"
                    value={formData.llm?.openai?.api_key || ''}
                    onChange={(e) => handleInputChange('llm.openai.api_key', e.target.value)}
                    placeholder="sk-..."
                  />
                </div>
                <div className="form-group">
                  <label>模型:</label>
                  <select 
                    value={formData.llm?.openai?.model || 'gpt-4'}
                    onChange={(e) => handleInputChange('llm.openai.model', e.target.value)}
                  >
                    <option value="gpt-4">GPT-4</option>
                    <option value="gpt-4-turbo">GPT-4 Turbo</option>
                    <option value="gpt-3.5-turbo">GPT-3.5 Turbo</option>
                  </select>
                </div>
                <div className="form-group">
                  <label>Base URL:</label>
                  <input
                    type="text"
                    value={formData.llm?.openai?.base_url || ''}
                    onChange={(e) => handleInputChange('llm.openai.base_url', e.target.value)}
                    placeholder="https://api.openai.com/v1"
                  />
                </div>
                <div className="form-group">
                  <label>超时时间 (秒):</label>
                  <input
                    type="number"
                    value={formData.llm?.openai?.timeout || 30}
                    onChange={(e) => handleInputChange('llm.openai.timeout', parseInt(e.target.value))}
                    min="10"
                    max="300"
                  />
                </div>
                <div className="form-group">
                  <label>最大Token数:</label>
                  <input
                    type="number"
                    value={formData.llm?.openai?.max_tokens || 2000}
                    onChange={(e) => handleInputChange('llm.openai.max_tokens', parseInt(e.target.value))}
                    min="100"
                    max="8000"
                  />
                </div>
              </div>
            )}

            {/* Gemini 配置 */}
            {formData.llm?.provider === 'gemini' && (
              <div className="provider-config">
                <h4>🌟 Google Gemini 配置</h4>
                <div className="form-group">
                  <label>API Key:</label>
                  <input
                    type="password"
                    value={formData.llm?.gemini?.api_key || ''}
                    onChange={(e) => handleInputChange('llm.gemini.api_key', e.target.value)}
                    placeholder="AIza..."
                  />
                </div>
                <div className="form-group">
                  <label>模型:</label>
                  <select 
                    value={formData.llm?.gemini?.model || 'gemini-1.5-flash'}
                    onChange={(e) => handleInputChange('llm.gemini.model', e.target.value)}
                  >
                    <option value="gemini-1.5-flash">Gemini 1.5 Flash</option>
                    <option value="gemini-1.5-pro">Gemini 1.5 Pro</option>
                    <option value="gemini-pro">Gemini Pro</option>
                  </select>
                </div>
                <div className="form-group">
                  <label>超时时间 (秒):</label>
                  <input
                    type="number"
                    value={formData.llm?.gemini?.timeout || 30}
                    onChange={(e) => handleInputChange('llm.gemini.timeout', parseInt(e.target.value))}
                    min="10"
                    max="300"
                  />
                </div>
              </div>
            )}

            {/* Anthropic 配置 */}
            {formData.llm?.provider === 'anthropic' && (
              <div className="provider-config">
                <h4>🧠 Anthropic Claude 配置</h4>
                <div className="form-group">
                  <label>API Key:</label>
                  <input
                    type="password"
                    value={formData.llm?.anthropic?.api_key || ''}
                    onChange={(e) => handleInputChange('llm.anthropic.api_key', e.target.value)}
                    placeholder="sk-ant-..."
                  />
                </div>
                <div className="form-group">
                  <label>模型:</label>
                  <select 
                    value={formData.llm?.anthropic?.model || 'claude-3-5-sonnet-20241022'}
                    onChange={(e) => handleInputChange('llm.anthropic.model', e.target.value)}
                  >
                    <option value="claude-3-5-sonnet-20241022">Claude 3.5 Sonnet</option>
                    <option value="claude-3-opus-20240229">Claude 3 Opus</option>
                    <option value="claude-3-haiku-20240307">Claude 3 Haiku</option>
                  </select>
                </div>
                <div className="form-group">
                  <label>超时时间 (秒):</label>
                  <input
                    type="number"
                    value={formData.llm?.anthropic?.timeout || 30}
                    onChange={(e) => handleInputChange('llm.anthropic.timeout', parseInt(e.target.value))}
                    min="10"
                    max="300"
                  />
                </div>
              </div>
            )}
          </div>

          <div className="settings-section">
            <h3>🔗 MCP Server 配置</h3>
            
            <div className="form-group">
              <label>主机地址:</label>
              <input
                type="text"
                value={formData.mcp?.host || 'localhost'}
                onChange={(e) => handleInputChange('mcp.host', e.target.value)}
                placeholder="localhost"
              />
            </div>
            
            <div className="form-group">
              <label>端口:</label>
              <input
                type="number"
                value={formData.mcp?.port || 8081}
                onChange={(e) => handleInputChange('mcp.port', parseInt(e.target.value))}
                min="1000"
                max="65535"
              />
            </div>
            
            <div className="form-group">
              <label>超时时间 (秒):</label>
              <input
                type="number"
                value={formData.mcp?.timeout || 30}
                onChange={(e) => handleInputChange('mcp.timeout', parseInt(e.target.value))}
                min="10"
                max="300"
              />
            </div>

            {/* QNG MCP配置 */}
            <div className="sub-section">
              <h4>⛏️ QNG MCP Server</h4>
              <div className="form-group">
                <label>
                  <input
                    type="checkbox"
                    checked={formData.mcp?.qng?.enabled || false}
                    onChange={(e) => handleInputChange('mcp.qng.enabled', e.target.checked)}
                  />
                  启用 QNG MCP Server
                </label>
              </div>
              <div className="form-group">
                <label>主机地址:</label>
                <input
                  type="text"
                  value={formData.mcp?.qng?.host || 'localhost'}
                  onChange={(e) => handleInputChange('mcp.qng.host', e.target.value)}
                  placeholder="localhost"
                />
              </div>
              <div className="form-group">
                <label>端口:</label>
                <input
                  type="number"
                  value={formData.mcp?.qng?.port || 8082}
                  onChange={(e) => handleInputChange('mcp.qng.port', parseInt(e.target.value))}
                  min="1000"
                  max="65535"
                />
              </div>
            </div>

            {/* MetaMask MCP配置 */}
            <div className="sub-section">
              <h4>🦊 MetaMask MCP Server</h4>
              <div className="form-group">
                <label>
                  <input
                    type="checkbox"
                    checked={formData.mcp?.metamask?.enabled || false}
                    onChange={(e) => handleInputChange('mcp.metamask.enabled', e.target.checked)}
                  />
                  启用 MetaMask MCP Server
                </label>
              </div>
              <div className="form-group">
                <label>主机地址:</label>
                <input
                  type="text"
                  value={formData.mcp?.metamask?.host || 'localhost'}
                  onChange={(e) => handleInputChange('mcp.metamask.host', e.target.value)}
                  placeholder="localhost"
                />
              </div>
              <div className="form-group">
                <label>端口:</label>
                <input
                  type="number"
                  value={formData.mcp?.metamask?.port || 8083}
                  onChange={(e) => handleInputChange('mcp.metamask.port', parseInt(e.target.value))}
                  min="1000"
                  max="65535"
                />
              </div>
            </div>
          </div>

          <div className="settings-actions">
            <button type="button" className="btn-secondary" onClick={onClose}>
              取消
            </button>
            <button type="submit" className="btn-primary" disabled={isLoading}>
              {isLoading ? '⏳ 保存中...' : '💾 保存配置'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

export default App; 
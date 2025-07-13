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
  const messagesEndRef = useRef(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  // 真实API调用
  const API_BASE_URL = 'http://localhost:9090';
  const MCP_BASE_URL = 'http://localhost:9091';

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
    
    // 检查MetaMask是否可用
    if (typeof window.ethereum === 'undefined') {
      throw new Error('MetaMask未安装，请先安装MetaMask扩展');
    }

    try {
      // 请求连接钱包
      const accounts = await window.ethereum.request({ 
        method: 'eth_requestAccounts' 
      });
      
      // 获取网络信息
      const chainId = await window.ethereum.request({ 
        method: 'eth_chainId' 
      });
      
      // 获取余额
      const balance = await window.ethereum.request({
        method: 'eth_getBalance',
        params: [accounts[0], 'latest']
      });

      setWalletConnected(true);
      setWalletAddress(accounts[0]);
      
      console.log('✅ 钱包连接成功:', {
        address: accounts[0],
        chainId: chainId,
        balance: balance
      });
      
      return {
        connected: true,
        accounts: accounts,
        network: 'Ethereum Mainnet',
        chain_id: chainId,
        balance: balance
      };
    } catch (error) {
      console.error('❌ 钱包连接失败:', error);
      throw error;
    }
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
      
      while (pollCount < maxPolls) {
        status = await pollWorkflowStatus(execution.session_id);
        pollCount++;
        
        console.log(`📊 第${pollCount}次轮询状态:`, status);
        
        if (status.status === 'completed' || status.status === 'failed') {
          break;
        }
        
        // 等待2秒后继续轮询
        await new Promise(resolve => setTimeout(resolve, 2000));
      }
      
      if (status.need_signature) {
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
      
      // 使用MetaMask进行签名
      if (typeof window.ethereum === 'undefined') {
        throw new Error('MetaMask未安装');
      }

      // 构建交易数据
      const transactionData = {
        to: signatureRequest.to_address,
        value: signatureRequest.value || '0x0',
        data: signatureRequest.data || '0x',
        gas: signatureRequest.gas_limit || '0x186A0', // 100000 gas
        gasPrice: signatureRequest.gas_price || '0x3B9ACA00' // 1 gwei
      };

      console.log('📝 交易数据:', transactionData);

      // 请求用户签名
      const signature = await window.ethereum.request({
        method: 'eth_sendTransaction',
        params: [transactionData]
      });

      console.log('✅ 交易签名成功:', signature);
      
      // 提交签名到后端
      const result = await submitSignature(currentSession.session_id, signature);
      
      setMessages(prev => [...prev, {
        id: Date.now(),
        type: 'system',
        content: '✅ 签名已提交，交易正在处理中...',
        timestamp: new Date()
      }]);

      // 轮询交易状态
      let pollCount = 0;
      const maxPolls = 20;
      
      while (pollCount < maxPolls) {
        const status = await pollWorkflowStatus(currentSession.session_id);
        pollCount++;
        
        console.log(`📊 第${pollCount}次轮询交易状态:`, status);
        
        if (status.status === 'completed') {
          setMessages(prev => [...prev, {
            id: Date.now() + 1,
            type: 'system',
            content: `🎉 交易完成！交易哈希: ${signature}`,
            timestamp: new Date()
          }]);
          break;
        } else if (status.status === 'failed') {
          setMessages(prev => [...prev, {
            id: Date.now() + 1,
            type: 'error',
            content: `❌ 交易失败: ${status.error || '未知错误'}`,
            timestamp: new Date()
          }]);
          break;
        }
        
        // 等待3秒后继续轮询
        await new Promise(resolve => setTimeout(resolve, 3000));
      }

      setSignatureRequest(null);
      setCurrentSession(null);

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
      await connectWallet();
      setMessages(prev => [...prev, {
        id: Date.now(),
        type: 'system',
        content: `🔗 钱包连接成功！地址: ${walletAddress}`,
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

  const handleKeyPress = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSendMessage();
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
        <h1>🤖 QNG 智能体</h1>
        <div className="wallet-info">
          {walletConnected ? (
            <span className="connected">
              🔗 {walletAddress.slice(0, 6)}...{walletAddress.slice(-4)}
            </span>
          ) : (
            <button 
              className="connect-btn"
              onClick={handleConnectWallet}
              disabled={isLoading}
            >
              🔗 连接钱包
            </button>
          )}
        </div>
      </div>

      <div className="chat-container">
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
            </div>
            <button 
              className="signature-btn"
              onClick={handleSignature}
              disabled={isLoading}
            >
              🔐 确认签名
            </button>
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
    </div>
  );
}

export default App; 
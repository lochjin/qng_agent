# QNG Agent - 智能区块链工作流系统

一个基于LangGraph和MCP协议的智能区块链工作流系统，支持代币兑换、质押等操作，具有用户签名验证和MetaMask集成功能。

## 🚀 功能特性

### 核心功能
- **智能工作流执行**: 基于LangGraph的任务分解和执行
- **LLM集成**: 支持OpenAI、Gemini、Anthropic等多种LLM提供商
- **MCP协议**: 模块化的MCP服务器架构
- **Long Polling**: 实时工作流状态更新
- **用户签名**: MetaMask钱包集成和交易签名
- **现代化UI**: React前端界面

### 工作流支持
- **代币兑换**: USDT ↔ BTC 等代币兑换
- **代币质押**: 将代币质押到各种DeFi协议
- **余额查询**: 查询钱包余额和代币信息
- **交易历史**: 查看交易记录和状态

## 🏗️ 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   React前端     │    │   智能体API     │    │   MCP服务器     │
│   (端口3000)    │◄──►│   (端口8080)    │◄──►│   (端口8081)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                       │
                                ▼                       ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │   QNG Chain     │    │  MetaMask服务   │
                       │  (LangGraph)    │    │   (端口8083)    │
                       └─────────────────┘    └─────────────────┘
```

### 服务组件

1. **智能体 (Agent)**
   - 分析用户请求
   - 调用LLM进行任务分解
   - 管理工作流执行
   - 处理用户签名

2. **QNG MCP服务器**
   - 执行QNG工作流
   - Long Polling状态更新
   - 会话管理
   - 签名验证

3. **MetaMask MCP服务器**
   - 钱包连接
   - 交易签名
   - 余额查询
   - 网络信息

4. **QNG Chain (LangGraph)**
   - 任务分解节点
   - 交易执行节点
   - 签名验证节点
   - 结果聚合节点

## 📋 系统要求

### 必需软件
- **Go 1.21+**: 后端开发
- **Node.js 18+**: 前端开发
- **npm 9+**: 包管理

### 可选软件
- **Git**: 版本控制
- **Docker**: 容器化部署

## 🛠️ 安装和运行

### 1. 克隆项目
```bash
git clone https://github.com/your-org/qng-agent.git
cd qng-agent
```

### 2. 配置环境变量
```bash
# 复制配置文件
cp config/config.yaml.example config/config.yaml

# 设置环境变量
export OPENAI_API_KEY="your-openai-api-key"
export GEMINI_API_KEY="your-gemini-api-key"
export ANTHROPIC_API_KEY="your-anthropic-api-key"
```

### 3. 启动系统
```bash
# 给脚本执行权限
chmod +x start.sh stop.sh

# 启动所有服务
./start.sh

# 或者分步启动
./start.sh build    # 构建项目
./start.sh start    # 启动服务
```

### 4. 访问系统
- **前端界面**: http://localhost:3000
- **智能体API**: http://localhost:8080
- **MCP服务器**: http://localhost:8081

### 5. 停止系统
```bash
./stop.sh
```

## 🔧 配置说明

### LLM配置
```yaml
llm:
  provider: "openai"  # openai, gemini, anthropic
  openai:
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4"
    timeout: 30
```

### MCP配置
```yaml
mcp:
  mode: "distributed"  # local, distributed
  qng:
    enabled: true
    host: "localhost"
    port: 8082
  metamask:
    enabled: true
    network: "Ethereum Mainnet"
```

## 📖 使用指南

### 基本使用流程

1. **连接钱包**
   - 点击"连接钱包"按钮
   - 授权MetaMask连接

2. **发送请求**
   - 在输入框中输入您的需求
   - 例如："我需要将1000USDT兑换成BTC"

3. **等待处理**
   - 系统会分析您的请求
   - 自动分解为具体任务
   - 显示处理进度

4. **签名授权**
   - 如果需要签名，会弹出签名请求
   - 在MetaMask中确认交易
   - 等待交易完成

5. **查看结果**
   - 系统会显示执行结果
   - 包含交易哈希和状态信息

### 支持的命令示例

```
✅ 代币兑换
"我需要将1000USDT兑换成BTC"
"帮我用500USDT换ETH"

✅ 代币质押
"将我的BTC质押到Compound"
"帮我质押0.1BTC到Aave"

✅ 余额查询
"查看我的钱包余额"
"我的USDT余额是多少"

✅ 复合操作
"将1000USDT兑换成BTC，然后质押到Compound"
```

## 🔍 API文档

### 智能体API

#### 处理消息
```http
POST /api/agent/process
Content-Type: application/json

{
  "message": "我需要将1000USDT兑换成BTC"
}
```

#### 轮询状态
```http
GET /api/agent/poll/{session_id}
```

#### 提交签名
```http
POST /api/agent/signature
Content-Type: application/json

{
  "session_id": "session_123",
  "signature": "0x..."
}
```

### MCP API

#### 执行工作流
```http
POST /api/mcp/qng/execute_workflow
Content-Type: application/json

{
  "message": "用户消息"
}
```

#### 轮询会话
```http
GET /api/mcp/qng/poll_session?session_id={session_id}&timeout=30
```

## 🧪 开发指南

### 项目结构
```
qng-agent/
├── cmd/                    # 命令行工具
│   ├── agent/             # 智能体主程序
│   └── mcp/               # MCP服务器
├── internal/               # 内部包
│   ├── agent/             # 智能体逻辑
│   ├── mcp/               # MCP协议实现
│   ├── qng/               # QNG链实现
│   ├── llm/               # LLM客户端
│   └── config/            # 配置管理
├── frontend/               # React前端
│   ├── src/
│   └── public/
├── config/                 # 配置文件
├── logs/                   # 日志文件
└── scripts/                # 脚本文件
```

### 开发模式
```bash
# 启动开发模式
./start.sh

# 查看日志
tail -f logs/agent.log
tail -f logs/mcp.log
tail -f logs/frontend.log

# 重启服务
./start.sh restart
```

### 添加新的工作流节点

1. **创建节点**
```go
// internal/qng/nodes/my_node.go
type MyNode struct{}

func (n *MyNode) Execute(ctx context.Context, input NodeInput) (*NodeOutput, error) {
    // 实现节点逻辑
    return &NodeOutput{
        Data:      result,
        NextNodes: []string{"next_node"},
        Completed: false,
    }, nil
}
```

2. **注册节点**
```go
// internal/qng/langgraph.go
func (lg *LangGraph) registerNodes() {
    lg.nodes["my_node"] = NewMyNode()
}
```

3. **更新图结构**
```go
func (lg *LangGraph) buildGraph() {
    lg.edges["my_node"] = []string{"next_node"}
}
```

## 🐛 故障排除

### 常见问题

1. **服务启动失败**
   ```bash
   # 检查端口占用
   lsof -i :8080
   lsof -i :8081
   lsof -i :3000
   
   # 强制停止进程
   ./stop.sh --force
   ```

2. **LLM调用失败**
   - 检查API密钥配置
   - 确认网络连接
   - 查看日志文件

3. **钱包连接失败**
   - 确保MetaMask已安装
   - 检查网络配置
   - 确认权限设置

4. **工作流执行超时**
   - 增加超时时间配置
   - 检查网络延迟
   - 查看详细日志

### 日志查看
```bash
# 实时查看日志
tail -f logs/agent.log
tail -f logs/mcp.log
tail -f logs/frontend.log

# 查看错误日志
grep "ERROR" logs/*.log
grep "WARN" logs/*.log
```

## 🤝 贡献指南

### 开发流程

1. **Fork项目**
2. **创建特性分支**
   ```bash
   git checkout -b feature/amazing-feature
   ```
3. **提交更改**
   ```bash
   git commit -m 'Add amazing feature'
   ```
4. **推送到分支**
   ```bash
   git push origin feature/amazing-feature
   ```
5. **创建Pull Request**

### 代码规范

- 使用Go标准格式化工具
- 遵循React最佳实践
- 添加适当的测试
- 更新相关文档

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🙏 致谢

- [LangGraph](https://github.com/langchain-ai/langgraph) - 工作流引擎
- [MCP Protocol](https://modelcontextprotocol.io/) - 模型上下文协议
- [React](https://reactjs.org/) - 前端框架
- [MetaMask](https://metamask.io/) - 钱包集成

## 📞 联系方式

- **项目主页**: https://github.com/your-org/qng-agent
- **问题反馈**: https://github.com/your-org/qng-agent/issues
- **邮箱**: your-email@example.com

---

**注意**: 这是一个演示项目，请在生产环境中谨慎使用，并确保遵循相关法律法规。 
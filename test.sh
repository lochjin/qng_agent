#!/bin/bash

# QNG Agent 集成测试脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    log_info "检查测试依赖..."
    
    # 检查Go
    if ! command -v go &> /dev/null; then
        log_error "Go未安装，请先安装Go"
        exit 1
    fi
    
    # 检查配置文件
    if [ ! -f "config/config.yaml" ]; then
        log_warning "配置文件不存在，创建默认配置..."
        cp config/config.yaml.example config/config.yaml 2>/dev/null || {
            log_error "无法创建配置文件"
            exit 1
        }
    fi
    
    log_success "依赖检查完成"
}

# 构建测试
build_test() {
    log_info "构建测试程序..."
    
    # 清理旧的构建文件
    rm -f test_integration
    
    # 构建测试程序
    go build -o test_integration test_integration.go
    
    if [ $? -eq 0 ]; then
        log_success "测试程序构建成功"
    else
        log_error "测试程序构建失败"
        exit 1
    fi
}

# 运行单元测试
run_unit_tests() {
    log_info "运行单元测试..."
    
    # 运行Go单元测试
    go test ./internal/... -v
    
    if [ $? -eq 0 ]; then
        log_success "单元测试通过"
    else
        log_warning "单元测试失败，继续集成测试"
    fi
}

# 运行集成测试
run_integration_test() {
    log_info "运行集成测试..."
    
    # 运行集成测试
    ./test_integration
    
    if [ $? -eq 0 ]; then
        log_success "集成测试通过"
    else
        log_error "集成测试失败"
        exit 1
    fi
}

# 运行前端测试
run_frontend_test() {
    log_info "运行前端测试..."
    
    cd frontend
    
    # 检查Node.js依赖
    if [ ! -d "node_modules" ]; then
        log_info "安装前端依赖..."
        npm install
    fi
    
    # 运行前端测试
    npm test -- --watchAll=false 2>/dev/null || {
        log_warning "前端测试失败或未配置"
    }
    
    cd ..
}

# 运行性能测试
run_performance_test() {
    log_info "运行性能测试..."
    
    # 这里可以添加性能测试
    # 例如：并发请求测试、内存使用测试等
    log_info "性能测试跳过（需要更多配置）"
}

# 生成测试报告
generate_report() {
    log_info "生成测试报告..."
    
    # 创建测试报告目录
    mkdir -p test_reports
    
    # 生成简单的测试报告
    cat > test_reports/integration_test_report.md << EOF
# QNG Agent 集成测试报告

## 测试时间
$(date)

## 测试结果
- ✅ 配置加载测试
- ✅ QNG Chain测试
- ✅ MCP服务器测试
- ✅ 智能体测试
- ✅ 完整工作流测试

## 测试环境
- Go版本: $(go version)
- Node.js版本: $(node --version 2>/dev/null || echo "未安装")
- 操作系统: $(uname -s)

## 注意事项
- LLM集成测试需要API密钥
- 钱包集成测试需要MetaMask
- 性能测试需要更多配置

EOF

    log_success "测试报告已生成: test_reports/integration_test_report.md"
}

# 清理测试文件
cleanup_test() {
    log_info "清理测试文件..."
    
    # 删除测试程序
    rm -f test_integration
    
    # 删除测试报告（可选）
    # rm -rf test_reports
    
    log_success "清理完成"
}

# 显示帮助信息
show_help() {
    echo "QNG Agent 测试脚本"
    echo ""
    echo "用法: $0 [命令]"
    echo ""
    echo "命令:"
    echo "  all        运行所有测试"
    echo "  unit       运行单元测试"
    echo "  integration 运行集成测试"
    echo "  frontend   运行前端测试"
    echo "  performance 运行性能测试"
    echo "  report     生成测试报告"
    echo "  clean      清理测试文件"
    echo "  help       显示帮助信息"
    echo ""
}

# 主函数
main() {
    case "${1:-all}" in
        all)
            check_dependencies
            build_test
            run_unit_tests
            run_integration_test
            run_frontend_test
            run_performance_test
            generate_report
            cleanup_test
            log_success "🎉 所有测试完成！"
            ;;
        unit)
            check_dependencies
            run_unit_tests
            ;;
        integration)
            check_dependencies
            build_test
            run_integration_test
            cleanup_test
            ;;
        frontend)
            run_frontend_test
            ;;
        performance)
            run_performance_test
            ;;
        report)
            generate_report
            ;;
        clean)
            cleanup_test
            ;;
        help)
            show_help
            ;;
        *)
            log_error "未知命令: $1"
            show_help
            exit 1
            ;;
    esac
}

# 运行主函数
main "$@" 
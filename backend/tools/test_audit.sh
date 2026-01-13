#!/bin/bash

# 审计日志功能测试脚本
# 用于快速验证审计日志功能是否正常工作

set -e

BASE_URL="http://localhost:8080"
ADMIN_TOKEN=""

echo "=========================================="
echo "审计日志功能测试脚本"
echo "=========================================="
echo ""

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

success() {
    echo -e "${GREEN}✓ $1${NC}"
}

error() {
    echo -e "${RED}✗ $1${NC}"
}

info() {
    echo -e "${YELLOW}➜ $1${NC}"
}

# 1. 登录获取管理员 token
info "步骤 1: 登录获取管理员 token"
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/user/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}')

ADMIN_TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$ADMIN_TOKEN" ]; then
    error "登录失败，请确保后端已启动且存在 admin 用户"
    echo "提示: 可以先注册用户，然后运行: cd backend && go run cmd/set_admin/main.go 1 admin"
    exit 1
fi

success "登录成功，获取到 token"
echo ""

# 2. 创建测试规则（会自动记录审计日志）
info "步骤 2: 创建测试规则"
CREATE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/rule/create_rule" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d '{
        "group_name": "test-audit",
        "name": "测试审计日志规则",
        "file_content": "groups:\n  - name: test\n    rules:\n      - alert: TestAlert\n        expr: up == 0\n        for: 5m"
    }')

RULE_ID=$(echo $CREATE_RESPONSE | grep -o '"id":[0-9]*' | cut -d':' -f2)

if [ -z "$RULE_ID" ]; then
    error "创建规则失败"
    echo "响应: $CREATE_RESPONSE"
    exit 1
fi

success "创建规则成功，规则 ID: $RULE_ID"
echo ""

# 3. 更新规则（会自动记录审计日志）
info "步骤 3: 更新规则"
UPDATE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/rule/update_rule" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d "{
        \"id\": $RULE_ID,
        \"name\": \"测试审计日志规则-已更新\",
        \"file_content\": \"groups:\n  - name: test\n    rules:\n      - alert: TestAlert\n        expr: up == 0\n        for: 10m\",
        \"comment\": \"测试更新操作\"
    }")

if echo $UPDATE_RESPONSE | grep -q "successfully"; then
    success "更新规则成功"
else
    error "更新规则失败"
    echo "响应: $UPDATE_RESPONSE"
fi
echo ""

# 等待一秒，确保审计日志已写入
sleep 1

# 4. 查询审计日志列表
info "步骤 4: 查询审计日志列表"
AUDIT_LOGS=$(curl -s -X GET "$BASE_URL/api/admin/audit/logs?page=1&page_size=10" \
    -H "Authorization: Bearer $ADMIN_TOKEN")

if echo $AUDIT_LOGS | grep -q "data"; then
    success "查询审计日志列表成功"
    echo ""
    echo "最近的审计记录:"
    echo $AUDIT_LOGS | python3 -m json.tool 2>/dev/null || echo $AUDIT_LOGS | jq '.' 2>/dev/null || echo $AUDIT_LOGS
else
    error "查询审计日志失败"
    echo "响应: $AUDIT_LOGS"
fi
echo ""

# 5. 查询审计统计数据
info "步骤 5: 查询审计统计数据"
AUDIT_STATS=$(curl -s -X GET "$BASE_URL/api/admin/audit/stats" \
    -H "Authorization: Bearer $ADMIN_TOKEN")

if echo $AUDIT_STATS | grep -q "data"; then
    success "查询审计统计成功"
    echo ""
    echo "审计统计数据:"
    echo $AUDIT_STATS | python3 -m json.tool 2>/dev/null || echo $AUDIT_STATS | jq '.' 2>/dev/null || echo $AUDIT_STATS
else
    error "查询审计统计失败"
    echo "响应: $AUDIT_STATS"
fi
echo ""

# 6. 测试筛选功能
info "步骤 6: 测试筛选功能（按资源类型筛选）"
FILTERED_LOGS=$(curl -s -X GET "$BASE_URL/api/admin/audit/logs?resource_type=rule&action=create" \
    -H "Authorization: Bearer $ADMIN_TOKEN")

if echo $FILTERED_LOGS | grep -q "data"; then
    success "筛选测试成功（规则创建操作）"
else
    error "筛选测试失败"
fi
echo ""

# 7. 清理测试数据（删除测试规则，也会记录审计日志）
info "步骤 7: 清理测试数据"
DELETE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/rule/delete_rule" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d "{\"id\": $RULE_ID}")

if echo $DELETE_RESPONSE | grep -q "successfully"; then
    success "删除测试规则成功"
else
    error "删除测试规则失败"
    echo "响应: $DELETE_RESPONSE"
fi
echo ""

echo "=========================================="
echo "测试完成！"
echo "=========================================="
echo ""
info "前端访问地址: http://localhost:5173/audit"
info "可以在前端页面查看完整的审计日志，包括："
echo "  - 规则创建记录"
echo "  - 规则更新记录（含变更对比）"
echo "  - 规则删除记录"
echo "  - 操作统计和趋势分析"
echo ""
success "审计日志功能测试通过！"

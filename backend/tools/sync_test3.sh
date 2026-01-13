#!/bin/bash
# 测试并发合并情况
set -euo pipefail

BASE_URL="http://localhost:8080"
USER="admin"
PASS="admin"       # 按实际修改

echo "1) 登录拿 token"
TOKEN=$(curl -s -X POST "$BASE_URL/api/user/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$USER\",\"password\":\"$PASS\"}" | jq -r '.data.access_token')
[[ -z "$TOKEN" || "$TOKEN" == "null" ]] && { echo "登录失败"; exit 1; }

echo "2) 创建一条干净的测试规则"
CREATE_RES=$(curl -s -X POST "$BASE_URL/api/rule/create_rule" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "group_name": "test-merge",
    "name": "并发可合并测试规则",
    "file_content": "groups:\n  - name: merge\n    rules:\n      - alert: Base\n        expr: up==0\n        for: 1m"
  }')

RULE_ID=$(echo "$CREATE_RES" | jq -r '.id')
if [[ -z "$RULE_ID" || "$RULE_ID" == "null" ]]; then
  echo "创建规则失败: $CREATE_RES"
  exit 1
fi
echo "规则ID: $RULE_ID (初始版本=1)"

BASE_VERSION=1

echo "3) 构造两个基于同一 base_version 的并发更新："
echo "   - A：只改 name"
echo "   - B：只改 file_content"

payloadA=$(cat <<JSON
{
  "id": $RULE_ID,
  "name": "并发更新-A-改名字",
  "file_content": "groups:\\n  - name: merge\\n    rules:\\n      - alert: Base\\n        expr: up==0\\n        for: 1m",
  "base_version": $BASE_VERSION,
  "comment": "A 修改 name"
}
JSON
)

payloadB=$(cat <<JSON
{
  "id": $RULE_ID,
  "name": "并发可合并测试规则",
  "file_content": "groups:\\n  - name: merge\\n    rules:\\n      - alert: Base-Changed-By-B\\n        expr: up==0\\n        for: 2m",
  "base_version": $BASE_VERSION,
  "comment": "B 修改 file_content"
}
JSON
)

curlA() {
  echo ">>> A 开始提交"
  curl -s -w "\nHTTP:%{http_code}\n" -X POST "$BASE_URL/api/rule/update_rule" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "$payloadA"
}

curlB() {
  echo ">>> B 开始提交"
  curl -s -w "\nHTTP:%{http_code}\n" -X POST "$BASE_URL/api/rule/update_rule" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "$payloadB"
}

echo "4) 并发发出 A / B 请求（同一个 base_version=$BASE_VERSION）"
curlA &
curlB &
wait

echo "5) 查看最终规则内容和版本"
curl -s -X GET "$BASE_URL/api/rule/list" \
  -H "Authorization: Bearer $TOKEN" \
  | jq ".data[] | select(.id==$RULE_ID)"

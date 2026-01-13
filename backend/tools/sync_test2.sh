#!/bin/bash
# 当前版本和提交时版本不同的情况。
set -euo pipefail

BASE_URL="http://localhost:8080"
USER="admin"
PASS="admin"   # 按你的实际密码填
RULE_ID=5      # 或者用创建新规则的方式先拿 ID

echo "1) 登录拿 token"
TOKEN=$(curl -s -X POST "$BASE_URL/api/user/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$USER\",\"password\":\"$PASS\"}" | jq -r '.data.access_token')
[[ -z "$TOKEN" || "$TOKEN" == "null" ]] && { echo "登录失败"; exit 1; }

echo "2) 获取当前版本"
CUR_VER=$(curl -s -X GET "$BASE_URL/api/rule/list" \
  -H "Authorization: Bearer $TOKEN" | jq -r ".data[] | select(.id==$RULE_ID) | .version")
echo "当前版本: $CUR_VER"

payloadA=$(cat <<JSON
{
  "id": $RULE_ID,
  "name": "更新-A",
  "file_content": "groups:\\n  - name: test\\n    rules:\\n      - alert: A\\n        expr: up==0",
  "base_version": $CUR_VER,
  "comment": "A"
}
JSON
)

payloadB=$(cat <<JSON
{
  "id": $RULE_ID,
  "name": "更新-B",
  "file_content": "groups:\\n  - name: test\\n    rules:\\n      - alert: B\\n        expr: up==0",
  "base_version": $CUR_VER,
  "comment": "B"
}
JSON
)

echo "3) 第一次更新（应成功）"
curl -s -w "\nHTTP:%{http_code}\n" -X POST "$BASE_URL/api/rule/update_rule" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "$payloadA"

echo "4) 用旧 base_version 再更新（应 409 冲突）"
curl -s -w "\nHTTP:%{http_code}\n" -X POST "$BASE_URL/api/rule/update_rule" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "$payloadB"

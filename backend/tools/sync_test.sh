#!/bin/bash
# 同时竞争情况
set -euo pipefail

BASE_URL="http://localhost:8080"
USER="admin"
PASS="admin"

echo "1) 登录拿 token"
TOKEN=$(curl -s -X POST "$BASE_URL/api/user/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$USER\",\"password\":\"$PASS\"}" | jq -r '.data.access_token')

if [[ -z "$TOKEN" || "$TOKEN" == "null" ]]; then
  echo "登录失败，检查后端或账户" && exit 1
fi

echo "2) 创建或复用规则（示例：新建）"
RULE_ID=$(curl -s -X POST "$BASE_URL/api/rule/create_rule" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "group_name": "test-lock",
    "name": "并发测试规则",
    "file_content": "groups:\n  - name: testsync\n    rules:\n      - alert: Foo\n        expr: up==0\n        for: 1m"
  }' | jq -r '.id')

echo "规则ID: $RULE_ID (初始版本=1)"

BASE_VERSION=1

echo "3) 并发两次更新，基线版本都用 $BASE_VERSION"
payloadA=$(cat <<'JSON'
{
  "id": RULE_ID,
  "name": "并发更新-A",
  "file_content": "groups:\n  - name: test\n    rules:\n      - alert: FooA\n        expr: up==0\n        for: 1m",
  "base_version": BASE_VERSION,
  "comment": "A"
}
JSON
)

payloadB=$(cat <<'JSON'
{
  "id": RULE_ID,
  "name": "并发更新-B",
  "file_content": "groups:\n  - name: test\n    rules:\n      - alert: FooB\n        expr: up==0\n        for: 2m",
  "base_version": BASE_VERSION,
  "comment": "B"
}
JSON
)

payloadA=${payloadA/RULE_ID/$RULE_ID}
payloadA=${payloadA/BASE_VERSION/$BASE_VERSION}
payloadB=${payloadB/RULE_ID/$RULE_ID}
payloadB=${payloadB/BASE_VERSION/$BASE_VERSION}

curlA() {
  curl -s -w "\nHTTP:%{http_code}\n" -X POST "$BASE_URL/api/rule/update_rule" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "$payloadA"
}

curlB() {
  curl -s -w "\nHTTP:%{http_code}\n" -X POST "$BASE_URL/api/rule/update_rule" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "$payloadB"
}

echo "同时发出请求..."
curlA &
curlB &
wait

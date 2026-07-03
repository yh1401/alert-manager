#!/usr/bin/env python3
"""
Mock Prometheus Server
模拟 Prometheus HTTP API，用于本地开发测试。
提供 /api/v1/query, /api/v1/alerts, /api/v1/rules, /api/v1/targets 等接口。
"""
import json
import time
import random
import os
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse, PlainTextResponse
import uvicorn

app = FastAPI(title="Mock Prometheus")

# 模拟数据存储
start_time = time.time()

def uptime():
    return int(time.time() - start_time)

@app.get("/-/healthy")
@app.get("/-/ready")
async def healthy():
    return PlainTextResponse("Prometheus Server is up and running.")

@app.get("/api/v1/query")
async def query(request: Request):
    query_expr = request.query_params.get("query", "")
    now = time.time()

    # 根据 query 返回不同的模拟数据
    if query_expr == "up":
        data = [
            {
                "metric": {"__name__": "up", "instance": "localhost:9090", "job": "prometheus"},
                "value": [now, "1"]
            },
            {
                "metric": {"__name__": "up", "instance": "localhost:9100", "job": "node_exporter"},
                "value": [now, "1"]
            },
            {
                "metric": {"__name__": "up", "instance": "localhost:3000", "job": "grafana"},
                "value": [now, "1"]
            }
        ]
    elif "node_cpu" in query_expr:
        data = [
            {
                "metric": {"__name__": "node_cpu_seconds_total", "instance": "localhost:9100", "mode": "idle"},
                "value": [now, str(random.uniform(1000, 5000))]
            }
        ]
    elif "node_memory" in query_expr:
        data = [
            {
                "metric": {"__name__": "node_memory_MemAvailable_bytes", "instance": "localhost:9100"},
                "value": [now, str(random.randint(2*1024*1024*1024, 4*1024*1024*1024))]
            }
        ]
    else:
        data = [
            {
                "metric": {"__name__": query_expr, "instance": "localhost:9090"},
                "value": [now, str(random.uniform(0, 100))]
            }
        ]

    return JSONResponse({
        "status": "success",
        "data": {
            "resultType": "vector",
            "result": data
        }
    })

@app.get("/api/v1/query_range")
async def query_range(request: Request):
    query_expr = request.query_params.get("query", "")
    start = float(request.query_params.get("start", time.time() - 3600))
    end = float(request.query_params.get("end", time.time()))
    step = request.query_params.get("step", "15s")

    # 生成时间序列数据
    values = []
    current = start
    while current <= end:
        val = random.uniform(0, 100)
        values.append([current, str(val)])
        current += 15

    data = [{
        "metric": {"__name__": query_expr, "instance": "localhost:9090"},
        "values": values
    }]

    return JSONResponse({
        "status": "success",
        "data": {
            "resultType": "matrix",
            "result": data
        }
    })

@app.get("/api/v1/alerts")
async def alerts():
    now = time.time()
    iso_time = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(now - 300))

    mock_alerts = {
        "alerts": [
            {
                "labels": {
                    "alertname": "HighCPUUsage",
                    "severity": "warning",
                    "instance": "localhost:9100",
                    "job": "node_exporter"
                },
                "annotations": {
                    "summary": "CPU 使用率超过 80%",
                    "description": "实例 localhost:9100 CPU 使用率为 85.3%"
                },
                "state": "firing",
                "activeAt": iso_time,
                "value": "85.3"
            },
            {
                "labels": {
                    "alertname": "HighMemoryUsage",
                    "severity": "critical",
                    "instance": "localhost:9100",
                    "job": "node_exporter"
                },
                "annotations": {
                    "summary": "内存使用率超过 90%",
                    "description": "实例 localhost:9100 内存使用率为 92.1%"
                },
                "state": "firing",
                "activeAt": iso_time,
                "value": "92.1"
            },
            {
                "labels": {
                    "alertname": "DiskSpaceLow",
                    "severity": "warning",
                    "instance": "localhost:9100",
                    "job": "node_exporter"
                },
                "annotations": {
                    "summary": "磁盘空间不足",
                    "description": "分区 / 剩余空间 15%"
                },
                "state": "pending",
                "activeAt": iso_time,
                "value": "85"
            }
        ]
    }

    return JSONResponse({"status": "success", "data": mock_alerts})

@app.get("/api/v1/rules")
async def rules():
    now = time.time()
    iso_time = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(now))

    mock_rules = {
        "groups": [
            {
                "name": "cpu_alerts",
                "file": "/tmp/alert-agent-rules/cpu_alerts.yaml",
                "rules": [
                    {
                        "name": "HighCPUUsage",
                        "query": "avg(100 - (avg by (instance) (irate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100)) > 80",
                        "health": "ok",
                        "type": "alerting",
                        "labels": {"severity": "warning"},
                        "annotations": {"summary": "CPU 使用率超过 80%"},
                        "alerts": [],
                        "state": "ok",
                        "evaluationTime": 0.002,
                        "lastEvaluation": iso_time
                    }
                ],
                "interval": 60,
                "evaluationTime": 0.003,
                "lastEvaluation": iso_time
            },
            {
                "name": "memory_alerts",
                "file": "/tmp/alert-agent-rules/memory_alerts.yaml",
                "rules": [
                    {
                        "name": "HighMemoryUsage",
                        "query": "(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100 > 85",
                        "health": "ok",
                        "type": "alerting",
                        "labels": {"severity": "critical"},
                        "annotations": {"summary": "内存使用率超过 85%"},
                        "alerts": [],
                        "state": "ok",
                        "evaluationTime": 0.001,
                        "lastEvaluation": iso_time
                    }
                ],
                "interval": 60,
                "evaluationTime": 0.002,
                "lastEvaluation": iso_time
            },
            {
                "name": "disk_alerts",
                "file": "/tmp/alert-agent-rules/disk_alerts.yaml",
                "rules": [
                    {
                        "name": "HighDiskUsage",
                        "query": "(1 - (node_filesystem_avail_bytes{mountpoint=\"/\"} / node_filesystem_size_bytes{mountpoint=\"/\"})) * 100 > 90",
                        "health": "err",
                        "lastError": "unexpected end of JSON input",
                        "type": "alerting",
                        "labels": {"severity": "critical"},
                        "annotations": {"summary": "磁盘使用率超过 90%"},
                        "alerts": [],
                        "state": "err",
                        "evaluationTime": 0.005,
                        "lastEvaluation": iso_time
                    }
                ],
                "interval": 60,
                "evaluationTime": 0.006,
                "lastEvaluation": iso_time
            }
        ]
    }

    return JSONResponse({"status": "success", "data": mock_rules})

@app.get("/api/v1/targets")
async def targets():
    now = time.time()
    iso_time = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(now))

    mock_targets = {
        "activeTargets": [
            {
                "labels": {"job": "prometheus"},
                "targets": [
                    {
                        "discoveredLabels": {"__address__": "localhost:9090"},
                        "labels": {"instance": "localhost:9090"},
                        "scrapePool": "prometheus",
                        "scrapeUrl": "http://localhost:9090/metrics",
                        "globalUrl": "http://localhost:9090/metrics",
                        "lastError": "",
                        "lastScrape": iso_time,
                        "lastScrapeDuration": 0.002,
                        "health": "up"
                    }
                ]
            },
            {
                "labels": {"job": "node_exporter"},
                "targets": [
                    {
                        "discoveredLabels": {"__address__": "localhost:9100"},
                        "labels": {"instance": "localhost:9100"},
                        "scrapePool": "node_exporter",
                        "scrapeUrl": "http://localhost:9100/metrics",
                        "globalUrl": "http://localhost:9100/metrics",
                        "lastError": "",
                        "lastScrape": iso_time,
                        "lastScrapeDuration": 0.001,
                        "health": "up"
                    }
                ]
            },
            {
                "labels": {"job": "grafana"},
                "targets": [
                    {
                        "discoveredLabels": {"__address__": "localhost:3000"},
                        "labels": {"instance": "localhost:3000"},
                        "scrapePool": "grafana",
                        "scrapeUrl": "http://localhost:3000/metrics",
                        "globalUrl": "http://localhost:3000/metrics",
                        "lastError": "connection refused",
                        "lastScrape": iso_time,
                        "lastScrapeDuration": 0,
                        "health": "down"
                    }
                ]
            }
        ],
        "droppedTargets": []
    }

    return JSONResponse({"status": "success", "data": mock_targets})

@app.get("/api/v1/status/config")
async def status_config():
    return JSONResponse({
        "status": "success",
        "data": {"yaml": "global:\n  scrape_interval: 15s\n\nscrape_configs:\n  - job_name: prometheus\n  - job_name: node_exporter"}
    })

@app.get("/")
async def root():
    return JSONResponse({
        "status": "running",
        "service": "Mock Prometheus",
        "uptime_seconds": uptime()
    })

if __name__ == "__main__":
    print("[Mock Prometheus] 启动在端口 9090")
    uvicorn.run(app, host="0.0.0.0", port=9090)

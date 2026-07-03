#!/usr/bin/env python3
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
import uvicorn
import os

app = FastAPI()

reload_count = 0
rules_dir = "/tmp/mock-vmalert-rules"

@app.post("/api/v1/rules/reload")
async def reload(request: Request):
    global reload_count
    reload_count += 1
    print(f"[INFO] Reload request #{reload_count} received from {request.client.host}")
    return JSONResponse({"status": "success", "reload_count": reload_count})

@app.get("/api/v1/rules")
async def list_rules():
    rules = []
    if os.path.exists(rules_dir):
        for f in os.listdir(rules_dir):
            if f.endswith(".yaml") or f.endswith(".yml"):
                rules.append({"file": f})
    return JSONResponse({"data": rules, "reload_count": reload_count})

@app.get("/")
async def health():
    return JSONResponse({"status": "running", "reload_count": reload_count})

if __name__ == "__main__":
    os.makedirs(rules_dir, exist_ok=True)
    uvicorn.run(app, host="0.0.0.0", port=8880)

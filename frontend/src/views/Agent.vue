<template>
    <div class="agent-page">
        <el-card class="card">
            <h2>部署 Agent</h2>
            <p>
                此页面提供通过后端接口下载 agent
                二进制，并包含快速部署与配置示例。
            </p>

            <el-divider />

            <section class="download">
                <h3>下载</h3>
                <p>点击下方按钮通过后端接口下载 agent（linux程序）</p>
                <el-button
                    type="primary"
                    @click="downloadAgent"
                    :loading="loading"
                >
                    下载
                </el-button>
                <el-button v-if="signedHash" type="text" @click="copyHash"
                    >复制 SHA256</el-button
                >
            </section>

            <el-divider />

            <section class="install">
                <h3>快速安装（Linux）</h3>
                <ol>
                    <li>
                        把下载的文件上传或直接下载到目标机器，例如：<code
                            >/tmp/agent</code
                        >
                    </li>
                    <li>赋予可执行权限并移动到可执行路径并运行：</li>
                </ol>
                <el-card class="snippet">
                    <pre><code>chmod +x /tmp/agent
sudo mv /tmp/agent /usr/local/bin/alert-agent
/usr/local/bin/alert-agent --help

# 常用参数（可省略则使用默认值）
# -backend          后端地址，默认 http://localhost:8080
# -vmalert_url      vmalert 地址，默认 http://localhost:8880
# -poll_interval    配置拉取间隔，默认 10s
# -heartbeat_interval 心跳间隔，默认 30s
# -rule_path        规则文件或目录，可多次指定，默认 ./rules.yaml
# -rules_file       （兼容旧参数）首个规则文件路径，默认 ./rules.yaml
# -identity_file    节点身份文件，默认 ./agent-state.yaml
# -log_file         日志文件，默认 ./agent.log
# -log_max_age      日志保留天数，默认 7

# 示例：指定分组、后台地址、心跳/拉取间隔
./agent \
  -backend http://localhost:8080 \
  -rule_path /etc/vmalert/rules.yaml \
  -rule_path /etc/vmalert/extra.d \
  -heartbeat_interval 30s \
  -poll_interval 10s
</code></pre>
                </el-card>

                <h4>使用 systemd 管理（推荐）</h4>
                <p>
                    创建 systemd 服务文件，例如
                    <code>/etc/systemd/system/alert-agent.service</code>：
                </p>
                <el-card class="snippet">
                    <pre><code>[Unit]
Description=Alert Manager Agent
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/alert-agent --config /etc/alert-agent/config.yaml
Restart=always
User=root

[Install]
WantedBy=multi-user.target
</code></pre>
                </el-card>

                <p>启用并启动：</p>
                <el-card class="snippet">
                    <pre><code>sudo systemctl daemon-reload
sudo systemctl enable alert-agent
sudo systemctl start alert-agent
sudo journalctl -u alert-agent -f
</code></pre>
                </el-card>
            </section>

            <el-divider />

            <section class="config">
                <h3>配置说明</h3>
                <p>agent 支持通过配置文件或命令行参数设置，例如：</p>
                <el-card class="snippet">
                    <pre><code># /etc/alert-agent/config.yaml (示例)
server:
  url: "http://YOUR_ALERT_MANAGER_HOST:PORT"
agent:
  id: "node-01"            # 可选，不设置时 agent 可能会自生成 id
  data_dir: "/var/lib/alert-agent"
logging:
  level: "info"
</code></pre>
                </el-card>

                <p>常见参数：</p>
                <ul>
                    <li>
                        <code>--config &lt;path&gt;</code>：指定配置文件路径
                    </li>
                    <li>
                        <code>--server.url &lt;url&gt;</code
                        >：直接通过命令行覆盖服务端地址
                    </li>
                </ul>
            </section>

            <el-divider />

            <section class="register">
                <h3>注册与首次连接</h3>
                <p>
                    当 agent
                    启动并且能访问后端管理服务时，它会自动注册。查看后端“节点管理”页面可以看到新注册的节点。
                </p>
                <p>
                    如果你的后端启用了鉴权，需要在 agent
                    配置或启动参数中添加相应的 token 信息。
                </p>
            </section>

            <el-divider />

            <section class="advanced">
                <h3>进阶：在容器 / Kubernetes 中运行</h3>
                <p>
                    可以将 agent 以侧车或 DaemonSet 方式运行在 Kubernetes
                    中。示例（DaemonSet 片段）：
                </p>
                <el-card class="snippet">
                    <pre><code>apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: alert-agent
spec:
  selector:
    matchLabels:
      app: alert-agent
  template:
    metadata:
      labels:
        app: alert-agent
    spec:
      containers:
      - name: alert-agent
        image: your-repo/alert-agent:latest
        args: ["--config", "/etc/alert-agent/config.yaml"]
        volumeMounts:
        - name: config
          mountPath: /etc/alert-agent
      volumes:
      - name: config
        configMap:
          name: alert-agent-config
</code></pre>
                </el-card>
            </section>

            <el-divider />

            <section class="notes">
                <h3>其他说明 / 故障排查</h3>
                <ul>
                    <li>
                        确保 agent 能访问后端管理服务的网络（防火墙、代理等）。
                    </li>
                    <li>
                        查看 agent 日志（systemd：<code
                            >journalctl -u alert-agent -f</code
                        >）。
                    </li>
                    <li>
                        若后端需要证书，请在配置中配置 TLS
                        相关选项并确保证书路径正确。
                    </li>
                    <li>
                        如果你使用的是 Windows，请把二进制改名为
                        <code>alert-agent.exe</code
                        >，并以服务方式或任务计划启动。
                    </li>
                </ul>
            </section>
        </el-card>
    </div>
</template>

<script setup>
import { ref } from "vue";
import { ElMessage } from "element-plus";

const loading = ref(false);
const signedHash = ref("");

// Helper: parse filename from Content-Disposition header
function parseFilenameFromContentDisposition(cd) {
    if (!cd) return null;
    // filename*=UTF-8''... or filename="..."
    // try filename* first
    const fnStar = cd.match(/filename\\*=UTF-8''([^;\\n\\r\\s]+)/i);
    if (fnStar && fnStar[1]) {
        try {
            // decode percent-encoding
            return decodeURIComponent(fnStar[1]);
        } catch (e) {
            return fnStar[1];
        }
    }
    const fnQuoted = cd.match(/filename=\"?([^\";]+)\"?/i);
    if (fnQuoted && fnQuoted[1]) {
        return fnQuoted[1];
    }
    return null;
}

async function downloadAgent() {
    loading.value = true;
    signedHash.value = "";
    try {
        const resp = await fetch("/api/agent/download", {
            method: "GET",
            // credentials: 'include' // enable if backend requires cookies/auth
        });
        if (!resp.ok) {
            ElMessage.error("下载失败: " + resp.status + " " + resp.statusText);
            loading.value = false;
            return;
        }

        // try to read sha256 from header X-SHA256 (optional)
        const hashHeader =
            resp.headers.get("X-SHA256") || resp.headers.get("X-File-Sha256");
        if (hashHeader) {
            signedHash.value = hashHeader;
        }

        const blob = await resp.blob();

        // determine filename
        let filename = "agent";
        const cd = resp.headers.get("Content-Disposition");
        const parsed = parseFilenameFromContentDisposition(cd);
        if (parsed) {
            filename = parsed;
        } else {
            // if content-type suggests exe on windows, we may add .exe (not mandatory)
            const ct = resp.headers.get("Content-Type") || "";
            if (ct.includes("windows") || ct.includes("exe")) {
                filename = "agent.exe";
            }
        }

        const url = window.URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        a.remove();
        window.URL.revokeObjectURL(url);

        ElMessage.success("下载已完成：" + filename);
    } catch (err) {
        console.error(err);
        ElMessage.error("下载出错");
    } finally {
        loading.value = false;
    }
}

function copyHash() {
    if (!signedHash.value) {
        ElMessage.info("没有可用的哈希值");
        return;
    }
    navigator.clipboard
        ?.writeText(signedHash.value)
        .then(() => ElMessage.success("已复制 SHA256"))
        .catch(() => ElMessage.error("复制失败"));
}
</script>

<style scoped>
.agent-page {
    padding: 16px;
}
.card {
    padding: 20px;
}
.download {
    margin-bottom: 12px;
}
.snippet {
    background: #0f1724;
    color: #e6eef8;
    padding: 12px;
    border-radius: 6px;
    overflow: auto;
    font-family:
        ui-monospace, SFMono-Regular, Menlo, Monaco, "Roboto Mono",
        "Segoe UI Mono";
    margin: 12px 0;
}
.snippet pre {
    margin: 0;
    white-space: pre-wrap;
}
.hint {
    margin-top: 8px;
    color: #6b7280;
}
</style>

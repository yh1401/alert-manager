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
# -backend              后端地址，默认 http://localhost:8080
#                       可以只填写端口（例如 :8080）或完整 URL（http://host:port）
# -vmalert_url          vmalert 地址，默认 http://localhost:8880
#                       可以只填写端口（例如 :8880）或完整 URL（http://host:port）
# -reload_url           vmalert reload 路径，输入完整的api路径
# -poll_interval        配置拉取间隔，默认 10s
# -heartbeat_interval   心跳间隔，默认 30s
# -rule_path            规则文件或目录，可多次指定，默认 ./rules.yaml
# -rules_file           （兼容旧参数）首个规则文件路径，默认 ./rules.yaml
# -identity_file        节点身份文件，默认 ./agent-state.yaml
# -log_file             日志文件，默认 ./agent.log
# -log_max_age          日志保留天数，默认 7

# 重要提示：
# 1) 规则文件的“注册”仅在 agent 第一次向后端注册时生效（即首次注册时 agent 会把规则信息写入后端）。
#    如果你在首次注册后修改了规则路径并希望后端重新按该路径记录规则注册信息，
#    需要在后端删除该 agent 的注册记录并让 agent 重新注册，或在管理界面执行相应重置操作。
# 2) 如果你的 vmalert reload 接口路径为 /api/reload/ 或相似路径，请将 -reload_url 设置为实际api路径 。
#    backend 与 vmalert_url 参数可只填写 ip:port（如 127.0.0.1:8080 / 127.0.0.1:8880）以简化配置。
#
# 示例：指定分组、后台地址、心跳/拉取间隔（使用端口简写）
/usr/local/bin/alert-agent \
  -backend :8080 \
  -vmalert_url :8880 \
  -reload_url api \
  -rule_path /etc/vmalert/rules.yaml \
  -rule_path /etc/vmalert/extra.d \
  -heartbeat_interval 30s \
  -poll_interval 10s \
  -log_file /var/log/alert-agent.log</code></pre>
                </el-card>

                <h4>使用 nohup 直接运行（简易）</h4>
                <p>
                    如果不使用进程管理器，可以用 <code>nohup</code> 将 agent
                    放到后台运行，并把日志重定向到文件：
                </p>
                <el-card class="snippet">
                    <pre><code># 把 agent 以 nohup 后台方式运行（示例）
sudo mkdir -p /var/log/alert-agent /etc/alert-agent
sudo chown $(whoami) /var/log/alert-agent

nohup /usr/local/bin/alert-agent \
  -backend http://YOUR_BACKEND_HOST:8080 \
  -vmalert_url http://YOUR_VMALERT_HOST:8880 \
  -reload_url api \
  -rule_path /etc/alert-agent/rules.yaml \
  -log_file /var/log/alert-agent/agent.log \
  > /var/log/alert-agent/agent.log 2>&1 &</code></pre>
                </el-card>
            </section>

            <section class="register">
                <h3>注册与首次连接</h3>
                <p>
                    当 agent
                    启动并且能访问后端管理服务时，它会自动注册。查看后端“节点管理”页面可以看到新注册的节点。
                </p>
                <p>
                    规则文件的注册信息只有在 agent
                    第一次注册时写入后端（请参见上面的“重要提示”）。如果需要修改已注册的规则路径或重新注册，请在后端删除该节点的注册信息后重启
                    agent。
                </p>
                <p>
                    如果你的后端启用了鉴权，需要在 agent
                    配置或启动参数中添加相应的 token 信息。
                </p>
            </section>

            <el-divider />

            <section class="notes">
                <h3>其他说明 / 故障排查</h3>
                <ul>
                    <li>
                        确保 agent 能访问后端管理服务的网络（防火墙、代理等）。
                    </li>
                    <li>查看 agent 日志，在启动参数定义的文件地址</li>
                    <li>
                        若后端需要证书，请在配置中配置 TLS
                        相关选项并确保证书路径正确。
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
    const fnStar = cd.match(/filename\*=UTF-8''([^;\n\r\s]+)/i);
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

<template>
    <el-container class="layout-container">
        <el-aside width="200px" class="aside">
            <div class="logo">Alert Manager</div>
            <el-menu
                router
                :default-active="$route.path"
                background-color="#001529"
                text-color="#fff"
            >
                <el-menu-item index="/dashboard">
                    <el-icon><DataBoard /></el-icon>
                    <span>总览</span>
                </el-menu-item>
                <el-menu-item index="/rules">
                    <el-icon><List /></el-icon>
                    <span>告警规则</span>
                </el-menu-item>
                <el-menu-item index="/nodes">
                    <el-icon><Monitor /></el-icon>
                    <span>节点管理</span>
                </el-menu-item>
                <el-menu-item index="/permissions">
                    <el-icon><Setting /></el-icon>
                    <span>权限管理</span>
                </el-menu-item>
                <el-menu-item index="/audit">
                    <el-icon><Document /></el-icon>
                    <span>审计日志</span>
                </el-menu-item>
                <el-menu-item index="/agent">
                    <el-icon><Document /></el-icon>
                    <span>代理安装</span>
                </el-menu-item>
            </el-menu>
        </el-aside>
        <el-container>
            <el-header class="header">
                <div class="header-right">
                    <el-button type="text" @click="goBack">
                        <el-icon><ArrowLeft /></el-icon>
                        返回
                    </el-button>
                    <el-button type="text" @click="logout">退出登录</el-button>
                </div>
            </el-header>
            <el-main>
                <div class="content-wrapper">
                    <router-view />
                </div>
            </el-main>
        </el-container>
    </el-container>
</template>

<script setup>
import { useRouter } from "vue-router";
import {
    List,
    Monitor,
    Setting,
    Document,
    ArrowLeft,
    DataBoard,
} from "@element-plus/icons-vue";
const router = useRouter();

const logout = () => {
    localStorage.removeItem("token");
    localStorage.removeItem("username");
    // 清除所有定时器（防止后台请求继续发送）
    // 获取最大定时器ID并清除所有
    const maxId = window.setTimeout(() => {}, 0);
    for (let i = maxId; i >= 0; i--) {
        window.clearInterval(i);
        window.clearTimeout(i);
    }
    router.push("/login");
};

const goBack = () => {
    router.back();
};
</script>

<style scoped>
.layout-container {
    height: 100vh;
    display: flex;
    flex-direction: row;
}
.aside {
    background: linear-gradient(180deg, #1a1e2e 0%, #2c3142 100%);
    color: white;
    width: 220px;
    flex-shrink: 0;
    height: 100vh;
    overflow-y: auto;
    box-shadow: 2px 0 8px rgba(0, 0, 0, 0.1);
}
.logo {
    height: 64px;
    line-height: 64px;
    text-align: center;
    font-weight: 600;
    font-size: 20px;
    background: rgba(0, 0, 0, 0.2);
    letter-spacing: 2px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}
.header {
    background: linear-gradient(135deg, #ffffff 0%, #f5f7fa 100%);
    border-bottom: none;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
    display: flex;
    align-items: center;
    justify-content: flex-end;
    padding-right: 24px;
    height: 64px;
    flex-shrink: 0;
}

.header-right {
    display: flex;
    gap: 16px;
    align-items: center;
}

:deep(.el-container) {
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
}

:deep(.el-main) {
    padding: 24px;
    overflow: auto;
    flex: 1;
    background: linear-gradient(135deg, #e8f0fe 0%, #f5f7fa 100%);
}

:deep(.el-menu) {
    border-right: none;
}

:deep(.el-menu-item) {
    margin: 8px 12px;
    border-radius: 8px;
    transition: all 0.3s ease;
}

:deep(.el-menu-item:hover) {
    background: rgba(24, 144, 255, 0.15) !important;
    transform: translateX(4px);
}

:deep(.el-menu-item.is-active) {
    background: linear-gradient(90deg, #1890ff 0%, #096dd9 100%) !important;
    box-shadow: 0 4px 12px rgba(24, 144, 255, 0.3);
}

.content-wrapper {
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    padding: 0 20px;
}
</style>

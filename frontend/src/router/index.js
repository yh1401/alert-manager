import { createRouter, createWebHistory } from "vue-router";

const routes = [
    {
        path: "/login",
        name: "Login",
        component: () => import("../views/Login.vue"),
    },
    {
        path: "/",
        name: "Layout",
        component: () => import("../views/Layout.vue"),
        redirect: "/dashboard",
        children: [
            {
                path: "dashboard",
                name: "Dashboard",
                component: () => import("../views/Dashboard.vue"),
                meta: { title: "总览" },
            },
            {
                path: "rules",
                name: "Rules",
                component: () => import("../views/RuleList.vue"),
                meta: { title: "告警规则" },
            },
            {
                path: "rules/new",
                name: "RuleNew",
                component: () => import("../views/RuleEditor.vue"),
                meta: { title: "新建规则" },
            },
            {
                path: "rules/:id/edit",
                name: "RuleEdit",
                component: () => import("../views/RuleEditor.vue"),
                meta: { title: "编辑规则" },
            },
            {
                path: "nodes",
                name: "Nodes",
                component: () => import("../views/NodeList.vue"),
                meta: { title: "节点管理" },
            },
            {
                path: "nodes/:id",
                name: "NodeDetail",
                component: () => import("../views/NodeDetail.vue"),
                meta: { title: "节点详情" },
            },
            {
                path: "rules/:ruleId/versions",
                name: "RuleVersion",
                component: () => import("../views/RuleVersion.vue"),
                props: true,
                meta: { title: "规则版本管理" },
            },
            {
                path: "agent",
                name: "Agent",
                component: () => import("../views/Agent.vue"),
                meta: { title: "Agent 部署" },
            },
            {
                path: "permissions",
                name: "PermissionManagement",
                component: () => import("../views/PermissionManagement.vue"),
                meta: { title: "权限管理" },
            },
            {
                path: "prometheus",
                name: "PrometheusMonitor",
                component: () => import("../views/PrometheusMonitor.vue"),
                meta: { title: "Prometheus 监控" },
            },
            {
                path: "audit",
                name: "AuditLog",
                component: () => import("../views/AuditLog.vue"),
                meta: { title: "审计日志" },
            },
            // 后续可以在这里加 Nodes 等页面
        ],
    },
];

const router = createRouter({
    history: createWebHistory(),
    routes,
});

// 简单的路由守卫
router.beforeEach((to, from, next) => {
    const token = localStorage.getItem("token");
    if (to.path !== "/login" && !token) {
        // 重定向到登录页前，清除 localStorage
        localStorage.removeItem("token");
        localStorage.removeItem("username");
        next("/login");
    } else if (to.path === "/login" && token) {
        // 如果已登录且试图进入登录页，重定向到首页
        next("/");
    } else {
        next();
    }
});

export default router;

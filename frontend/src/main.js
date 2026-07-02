import { createApp } from "vue";
import ElementPlus from "element-plus";
import "element-plus/dist/index.css";
import * as ElementPlusIconsVue from "@element-plus/icons-vue";
import axios from "axios";
import App from "./App.vue";
import router from "./router";
import "./style.css";

const app = createApp(App);

// 配置 axios baseURL
axios.defaults.baseURL = "/";

// 请求拦截器：自动附加 Authorization
axios.interceptors.request.use(
    (config) => {
        const token = localStorage.getItem("token");
        if (token) {
            const auth = token.startsWith("Bearer ")
                ? token
                : `Bearer ${token}`;
            config.headers = config.headers || {};
            config.headers.Authorization = auth;
        }
        return config;
    },
    (error) => Promise.reject(error),
);

// 注册所有图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
    app.component(key, component);
}

app.use(ElementPlus);
app.use(router);
app.mount("#app");

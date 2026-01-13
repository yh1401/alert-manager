<template>
    <div class="login-container">
      <el-card class="login-card">
        <template #header >
          <h2 style="text-align: center">Alert Manager</h2>
        </template>
        <el-form :model="form" label-width="0">
          <el-form-item>
            <el-input v-model="form.username" placeholder="用户名" prefix-icon="User" />
          </el-form-item>
          <el-form-item>
            <el-input v-model="form.password" type="password" placeholder="密码" prefix-icon="Lock" show-password />
          </el-form-item>
          <el-link v-if="isLogin" type="primary" @click="isLogin = false">没有账号，去注册</el-link>
          <el-link v-else type="primary" @click="isLogin = true">返回登录</el-link>

          <el-button v-if="isLogin" type="primary" style="width: 100%" @click="handleLogin" :loading="loading">登录</el-button>
          <el-button v-else type="primary" style="width: 100%" @click="handleRegister" :loading="loading">注册</el-button>


        </el-form>
      </el-card>
    </div>
  </template>
  
  <script setup>
  import { ref } from 'vue';
  import axios from 'axios';
  import { useRouter } from 'vue-router';
  import { ElMessage } from 'element-plus';
  
  const router = useRouter()
  const isLogin = ref(true);
  const form = ref({ username: '', password: '' })
  const loading = ref(false)
  
  const handleLogin = async () => {
    if (!form.value.username || !form.value.password) return ElMessage.warning('请输入用户名和密码')
    
    loading.value = true
    try {
      // 调用后端登录接口
      const res = await axios.post('/api/user/login', form.value)
      if (res.data.code === 200) {
        localStorage.setItem('token', res.data.data.access_token)
        ElMessage.success('登录成功')
        router.push('/')
      } else {
        ElMessage.error(res.data.msg || '登录失败')
      }
    } catch (err) {
      ElMessage.error(err.response?.data?.error || '登录请求失败')
    } finally {
      loading.value = false
    }
  }

  const handleRegister = async () => {
  if (!form.value.username || !form.value.password) return ElMessage.warning('请输入用户名和密码')

    loading.value = true
    try {
      // 调用后端注册接口
      const res = await axios.post('/api/user/register', form.value)
      if (res.data.code === 200) {
        ElMessage.success('注册成功,请登录')
        isLogin.value = true
      } else {
        ElMessage.error(res.data.msg || '注册失败')
      }
    } catch (err) {
      ElMessage.error(err.response?.data?.error || '注册请求失败')
    } finally {
      loading.value = false
    }
  }

  </script>
  
  

  <style scoped>
  .login-container {
    height: 100vh;
    display: flex;
    justify-content: center;
    align-items: center;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    position: relative;
    overflow: hidden;
  }
  
  .login-container::before {
    content: '';
    position: absolute;
    width: 400px;
    height: 400px;
    background: radial-gradient(circle, rgba(255,255,255,0.1) 0%, transparent 70%);
    top: -200px;
    right: -100px;
    animation: float 6s ease-in-out infinite;
  }
  
  .login-container::after {
    content: '';
    position: absolute;
    width: 300px;
    height: 300px;
    background: radial-gradient(circle, rgba(255,255,255,0.08) 0%, transparent 70%);
    bottom: -150px;
    left: -80px;
    animation: float 8s ease-in-out infinite reverse;
  }
  
  @keyframes float {
    0%, 100% { transform: translateY(0); }
    50% { transform: translateY(20px); }
  }
  
  .login-card {
    width: 420px;
    border-radius: 16px;
    box-shadow: 0 8px 32px rgba(31, 38, 135, 0.37);
    backdrop-filter: blur(4px);
    border: 1px solid rgba(255, 255, 255, 0.18);
    animation: slideUp 0.6s ease-out;
    z-index: 1;
  }
  
  @keyframes slideUp {
    from {
      opacity: 0;
      transform: translateY(30px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }
  
  :deep(.el-card__header) {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    border-radius: 16px 16px 0 0;
  }
  
  :deep(.el-button--primary) {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    border: none;
    border-radius: 8px;
    transition: all 0.3s ease;
  }
  
  :deep(.el-button--primary:hover) {
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(102, 126, 234, 0.4);
  }
  
  :deep(.el-input) {
    margin-bottom: 8px;
  }
  
  :deep(.el-input__wrapper) {
    border-radius: 8px;
    transition: all 0.3s ease;
  }
  
  :deep(.el-input__wrapper:hover) {
    box-shadow: 0 2px 8px rgba(102, 126, 234, 0.2);
  }
  </style>

  
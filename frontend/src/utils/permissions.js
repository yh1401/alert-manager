import { ref } from 'vue'
import axios from 'axios'

const API_BASE = 'http://localhost:8080/api'

// 全局状态
const currentUser = ref(null)
const userPermissions = ref([])
const isAdmin = ref(false)

const getToken = () => localStorage.getItem('token')

// 加载当前用户权限
export async function loadUserPermissions() {
  try {
    const token = getToken()
    if (!token) return

    // 获取用户权限列表
    const res = await axios.get(`${API_BASE}/user/permissions`, {
      headers: { Authorization: `Bearer ${token}` }
    })
    userPermissions.value = res.data.data || []

    // 简单解析token获取用户信息（或从后端额外接口获取）
    // 这里我们通过获取用户列表来判断是否是admin（实际应该有专门接口）
    try {
      const userRes = await axios.get(`${API_BASE}/admin/users`, {
        headers: { Authorization: `Bearer ${token}` }
      })
      // 如果能成功调用admin接口，说明是管理员
      isAdmin.value = true
    } catch {
      isAdmin.value = false
    }
  } catch (err) {
    console.error('Failed to load permissions:', err)
  }
}

// 检查是否有指定资源的权限
export function hasPermission(resourceType, resourceId, action = 'read') {
  if (isAdmin.value) return true
  
  return userPermissions.value.some(p => {
    if (p.resource_type !== resourceType || p.resource_id !== resourceId) {
      return false
    }
    // read 可以由 read 或 write 满足
    if (action === 'read') {
      return p.action === 'read' || p.action === 'write'
    }
    // write 必须是 write
    return p.action === action
  })
}

// 检查是否是管理员
export function checkIsAdmin() {
  return isAdmin.value
}

// 获取用户的所有权限
export function getUserPermissions() {
  return userPermissions.value
}

export default {
  loadUserPermissions,
  hasPermission,
  checkIsAdmin,
  getUserPermissions
}

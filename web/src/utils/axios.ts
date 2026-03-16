import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const service = axios.create({
  baseURL: '/api',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json;charset=utf-8'
  }
})

// 请求拦截器
service.interceptors.request.use(
  (config) => {
    // 从本地存储取token
    const token = localStorage.getItem('user-token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    ElMessage.error('请求失败，请检查网络')
    return Promise.reject(error)
  }
)

// 响应拦截器
service.interceptors.response.use(
  (response) => {
    const res = response.data
    if (res.code !== 200) {
      ElMessage.error(res.msg || '操作失败')
      return Promise.reject(res)
    }
    return res
  },
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('user-token')
      ElMessageBox.confirm(
        '登录过期，请重新登录',
        '提示',
        {
          confirmButtonText: '重新登录',
          cancelButtonText: '取消',
          type: 'warning'
        }
      ).then(() => {
        window.location.href = '/login'
      })
    } else {
      ElMessage.error(error.message || '服务器错误')
    }
    return Promise.reject(error)
  }
)

export default service
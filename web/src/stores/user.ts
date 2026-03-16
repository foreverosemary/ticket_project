import { defineStore } from 'pinia'
import { ref } from 'vue'
import { login, register } from '@/api/user'
import { ElMessage } from 'element-plus'

export const useUserStore = defineStore(
  'user',
  () => {
    const token = ref(localStorage.getItem('user-token') || '')
    const username = ref(localStorage.getItem('user-name') || '')
    const roleCode = ref(localStorage.getItem('user-role') || '')

    // 登录
    const userLogin = async (params) => {
      try {
        const res = await login(params)
        token.value = res.token
        username.value = res.username
        roleCode.value = res.roleCode
        // 本地存储
        localStorage.setItem('user-token', res.token)
        localStorage.setItem('user-name', res.username)
        localStorage.setItem('user-role', res.roleCode)
        ElMessage.success('登录成功')
        return res
      } catch (error) {
        ElMessage.error('登录失败')
        return Promise.reject(error)
      }
    }

    // 注册
    const userRegister = async (params) => {
      try {
        const res = await register(params)
        ElMessage.success('注册成功，请登录')
        return res
      } catch (error) {
        ElMessage.error('注册失败')
        return Promise.reject(error)
      }
    }

    // 登出
    const logout = () => {
      token.value = ''
      username.value = ''
      roleCode.value = ''
      localStorage.clear()
    }

    return {
      token,
      username,
      roleCode,
      userLogin,
      userRegister,
      logout
    }
  },
  {
    persist: true
  }
)
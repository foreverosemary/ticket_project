<template>
  <div class="login-container">
    <el-card class="login-card" shadow="hover">
      <h2 class="login-title">G-Ticket 校园抢票系统</h2>
      <el-form ref="loginFormRef" :model="loginForm" :rules="loginRules" label-width="0px" class="login-form">
        <el-form-item prop="username">
          <el-input v-model="loginForm.username" placeholder="请输入用户名" prefix-icon="User" size="large" />
        </el-form-item>
        <el-form-item prop="password">
          <el-input v-model="loginForm.password" type="password" placeholder="请输入密码" prefix-icon="Lock" size="large" show-password />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" size="large" class="login-btn" @click="handleLogin" :loading="loading">
            登录
          </el-button>
        </el-form-item>
        <div class="login-register">
          还没有账号？<router-link to="/register">立即注册</router-link>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useUserStore } from '@/stores/user'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'

const loginFormRef = ref()
const loading = ref(false)
const router = useRouter()
const userStore = useUserStore()

const loginForm = reactive({
  username: '',
  password: ''
})

const loginRules = reactive({
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 2, max: 20, message: '用户名长度2-20位', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, max: 20, message: '密码长度6-20位', trigger: 'blur' }
  ]
})

const handleLogin = async () => {
  const valid = await loginFormRef.value.validate()
  if (!valid) return

  loading.value = true
  try {
    await userStore.userLogin(loginForm)
    router.push(userStore.roleCode === 'ADMIN' ? '/admin/activities' : '/activities')
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped lang="scss">
.login-container {
  width: 100vw;
  height: 100vh;
  background: #f5f7fa;
  display: flex;
  align-items: center;
  justify-content: center;

  .login-card {
    width: 400px;
    padding: 20px;

    .login-title {
      text-align: center;
      margin-bottom: 20px;
      color: #303133;
    }

    .login-form {
      .login-btn {
        width: 100%;
        margin-top: 10px;
      }
    }

    .login-register {
      text-align: center;
      margin-top: 15px;
      a {
        color: #409eff;
        text-decoration: none;
      }
    }
  }
}
</style>
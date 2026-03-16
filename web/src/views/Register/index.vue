<template>
  <div class="register-container">
    <el-card class="register-card" shadow="hover">
      <h2 class="register-title">用户注册</h2>
      <el-form ref="registerFormRef" :model="registerForm" :rules="registerRules" label-width="0px" class="register-form">
        <el-form-item prop="username">
          <el-input v-model="registerForm.username" placeholder="请输入用户名" prefix-icon="User" size="large" />
        </el-form-item>
        <el-form-item prop="password">
          <el-input v-model="registerForm.password" type="password" placeholder="请输入密码" prefix-icon="Lock" size="large" show-password />
        </el-form-item>
        <el-form-item prop="confirmPassword">
          <el-input v-model="registerForm.confirmPassword" type="password" placeholder="请确认密码" prefix-icon="Lock" size="large" show-password />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" size="large" class="register-btn" @click="handleRegister" :loading="loading">
            注册
          </el-button>
        </el-form-item>
        <div class="register-login">
          已有账号？<router-link to="/login">立即登录</router-link>
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

const registerFormRef = ref()
const loading = ref(false)
const router = useRouter()
const userStore = useUserStore()

const registerForm = reactive({
  username: '',
  password: '',
  confirmPassword: ''
})

const registerRules = reactive({
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 2, max: 20, message: '用户名长度2-20位', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, max: 20, message: '密码长度6-20位', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    {
      validator: (_, value, callback) => {
        if (value !== registerForm.password) {
          callback(new Error('两次密码不一致'))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
})

const handleRegister = async () => {
  const valid = await registerFormRef.value.validate()
  if (!valid) return

  loading.value = true
  try {
    await userStore.userRegister({
      username: registerForm.username,
      password: registerForm.password
    })
    router.push('/login')
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped lang="scss">
.register-container {
  width: 100vw;
  height: 100vh;
  background: #f5f7fa;
  display: flex;
  align-items: center;
  justify-content: center;

  .register-card {
    width: 400px;
    padding: 20px;

    .register-title {
      text-align: center;
      margin-bottom: 20px;
      color: #303133;
    }

    .register-form {
      .register-btn {
        width: 100%;
        margin-top: 10px;
      }
    }

    .register-login {
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
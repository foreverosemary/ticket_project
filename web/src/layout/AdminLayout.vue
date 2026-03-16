<template>
  <el-container class="admin-layout" style="height: 100vh;">
    <el-header class="layout-header">
      <div class="header-left">
        <h2>G-Ticket 管理员后台</h2>
      </div>
      <div class="header-right">
        <span>管理员：{{ userStore.username }}</span>
        <el-button type="text" @click="handleLogout">退出登录</el-button>
      </div>
    </el-header>
    <el-container>
      <el-aside width="200px" class="layout-aside">
        <el-menu default-active="/admin/activities" class="layout-menu" router>
          <el-menu-item index="/admin/activities">
            <template #icon><el-icon><Calendar /></el-icon></template>
            活动管理
          </el-menu-item>
          <el-menu-item index="/admin/orders">
            <template #icon><el-icon><ShoppingCart /></el-icon></template>
            订单管理
          </el-menu-item>
          <el-menu-item index="/admin/users">
            <template #icon><el-icon><User /></el-icon></template>
            用户管理
          </el-menu-item>
        </el-menu>
      </el-aside>
      <el-main class="layout-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { Calendar, ShoppingCart, User } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'

const userStore = useUserStore()
const router = useRouter()

const handleLogout = () => {
  userStore.logout()
  ElMessage.success('退出成功')
  router.push('/login')
}
</script>

<style scoped lang="scss">
.admin-layout {
  .layout-header {
    background: #1989fa;
    color: #fff;
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0 20px;
    .header-left h2 {
      margin: 0;
    }
    .header-right {
      span {
        margin-right: 20px;
      }
      el-button {
        color: #fff;
      }
    }
  }
  .layout-aside {
    background: #f0f2f5;
    .layout-menu {
      height: 100%;
      border-right: none;
    }
  }
  .layout-main {
    padding: 20px;
    background: #fff;
  }
}
</style>
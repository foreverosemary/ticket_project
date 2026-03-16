<template>
  <el-container class="layout-container" style="height: 100vh;">
    <el-header class="layout-header">
      <div class="header-left">
        <h2>G-Ticket 校园抢票系统</h2>
      </div>
      <div class="header-right">
        <span>欢迎您，{{ userStore.username }}</span>
        <el-button type="text" @click="handleLogout">退出登录</el-button>
      </div>
    </el-header>
    <el-container>
      <el-aside width="200px" class="layout-aside">
        <el-menu default-active="/user/orders" class="layout-menu" router>
          <el-menu-item index="/user/orders">
            <template #icon><el-icon><ShoppingCart /></el-icon></template>
            我的订单
          </el-menu-item>
          <el-menu-item index="/user/tickets">
            <template #icon><el-icon><Ticket /></el-icon></template>
            我的票券
          </el-menu-item>
          <el-menu-item index="/user/info">
            <template #icon><el-icon><UserFilled /></el-icon></template>
            个人信息
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
import { ShoppingCart, Ticket, UserFilled } from '@element-plus/icons-vue'
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
.layout-container {
  .layout-header {
    background: #409eff;
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
    background: #f5f7fa;
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
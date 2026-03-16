import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { ElMessage } from 'element-plus'

// 公开路由
const publicRoutes = [
  {
    path: '/',
    redirect: '/activities'
  },
  {
    path: '/login',
    component: () => import('@/views/Login/index.vue')
  },
  {
    path: '/register',
    component: () => import('@/views/Register/index.vue')
  },
  {
    path: '/activities',
    component: () => import('@/views/Activity/List.vue')
  },
  {
    path: '/activities/:id',
    component: () => import('@/views/Activity/Detail.vue')
  }
]

// 私有路由
const privateRoutes = [
  {
    path: '/user',
    redirect: '/user/orders',
    component: () => import('@/layout/index.vue'),
    children: [
      { path: 'orders', component: () => import('@/views/User/OrderList.vue') },
      { path: 'tickets', component: () => import('@/views/User/TicketList.vue') },
      { path: 'info', component: () => import('@/views/User/Info.vue') }
    ]
  },
  {
    path: '/admin',
    redirect: '/admin/activities',
    component: () => import('@/layout/AdminLayout.vue'),
    children: [
      { path: 'activities', component: () => import('@/views/Admin/Activity/List.vue') },
      { path: 'activities/create', component: () => import('@/views/Admin/Activity/Create.vue') },
      { path: 'activities/edit/:id', component: () => import('@/views/Admin/Activity/Edit.vue') },
      { path: 'orders', component: () => import('@/views/Admin/OrderList.vue') },
      { path: 'users', component: () => import('@/views/Admin/UserList.vue') }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes: [...publicRoutes, ...privateRoutes]
})

// 路由守卫
router.beforeEach((to, _, next) => {
  const userStore = useUserStore()
  // 无需登录的路由
  if (['/login', '/register'].includes(to.path)) {
    next()
    return
  }
  // 未登录
  if (!userStore.token) {
    ElMessage.warning('请先登录')
    next('/login')
    return
  }
  // 管理员权限
  if (to.path.startsWith('/admin') && userStore.roleCode !== 'ADMIN') {
    ElMessage.error('无管理员权限')
    next('/user/orders')
    return
  }
  next()
})

export default router
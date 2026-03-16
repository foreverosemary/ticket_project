import request from '@/utils/axios'

// 注册
export const register = (params) => {
  return request({
    url: '/v1/users',
    method: 'post',
    data: params
  })
}

// 登录
export const login = (params) => {
  return request({
    url: '/v1/users/login',
    method: 'post',
    data: params
  })
}

// 获取当前用户信息
export const getCurrentUser = () => {
  return request({
    url: '/v1/users/current',
    method: 'get'
  })
}
import request from '@/utils/axios'

// 创建订单
export const createOrder = (params) => {
  return request({
    url: '/v1/orders',
    method: 'post',
    data: params
  })
}

// 获取我的订单
export const getMyOrderList = (params) => {
  return request({
    url: '/v1/users/orders',
    method: 'get',
    params
  })
}

// 更新订单状态
export const updateOrder = (id, params) => {
  return request({
    url: `/v1/orders/${id}`,
    method: 'patch',
    data: params
  })
}
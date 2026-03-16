import request from '@/utils/axios'

// 获取活动列表
export const getActivityList = (params) => {
  return request({
    url: '/v1/activities',
    method: 'get',
    params
  })
}

// 获取活动详情
export const getActivityDetail = (id) => {
  return request({
    url: `/v1/activities/${id}`,
    method: 'get'
  })
}

// 创建活动
export const createActivity = (params) => {
  return request({
    url: '/v1/activities',
    method: 'post',
    data: params
  })
}
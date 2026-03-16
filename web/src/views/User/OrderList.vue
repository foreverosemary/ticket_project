<template>
  <div class="order-list-container">
    <el-page-header content="我的订单" class="page-header" />
    <el-form :model="searchForm" inline class="search-form" @submit.prevent="getMyOrderListApi">
      <el-form-item label="活动名称">
        <el-input v-model="searchForm.activityName" placeholder="请输入活动名称" clearable />
      </el-form-item>
      <el-form-item label="订单状态">
        <el-select v-model="searchForm.status" placeholder="请选择订单状态" clearable>
          <el-option label="待支付" value="0" />
          <el-option label="已支付" value="1" />
          <el-option label="已取消" value="2" />
          <el-option label="已过期" value="3" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="getMyOrderListApi">查询</el-button>
        <el-button @click="resetSearch">重置</el-button>
      </el-form-item>
    </el-form>

    <el-table :data="orderList" border stripe class="order-table" v-loading="loading">
      <el-table-column prop="orderId" label="订单ID" align="center" />
      <el-table-column prop="activityName" label="活动名称" align="center" />
      <el-table-column prop="status" label="订单状态" align="center" :formatter="formatOrderStatus" />
      <el-table-column prop="createdAt" label="创建时间" align="center" :formatter="formatTime" />
      <el-table-column prop="payTime" label="支付时间" align="center" :formatter="formatPayTime" />
      <el-table-column label="操作" align="center">
        <template #default="scope">
          <el-button type="primary" size="small" @click="toOrderDetail(scope.row.orderId)">
            查看详情
          </el-button>
          <el-button size="small" type="success" @click="handlePay(scope.row.orderId)" v-if="scope.row.status === 0">
            立即支付
          </el-button>
          <el-button size="small" type="danger" @click="handleCancel(scope.row.orderId)" v-if="scope.row.status === 0">
            取消订单
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
      class="pagination"
      v-model:current-page="searchForm.pageNum"
      v-model:page-size="searchForm.pageSize"
      :total="total"
      layout="total, sizes, prev, pager, next, jumper"
      @size-change="getMyOrderListApi"
      @current-change="getMyOrderListApi"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { getMyOrderList, updateOrder } from '@/api/order'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import dayjs from 'dayjs'

const router = useRouter()
const orderList = ref([])
const total = ref(0)
const loading = ref(false)

const searchForm = reactive({
  activityName: '',
  status: '',
  pageNum: 1,
  pageSize: 10
})

onMounted(() => {
  getMyOrderListApi()
})

const getMyOrderListApi = async () => {
  loading.value = true
  try {
    const params = {
      ...searchForm,
      status: searchForm.status ? Number(searchForm.status) : undefined
    }
    const res = await getMyOrderList(params)
    orderList.value = res.orders
    total.value = res.total
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}

const resetSearch = () => {
  searchForm.activityName = ''
  searchForm.status = ''
  searchForm.pageNum = 1
  getMyOrderListApi()
}

const formatOrderStatus = (row) => {
  switch (row.status) {
    case 0: return '待支付'
    case 1: return '已支付'
    case 2: return '已取消'
    case 3: return '已过期'
    default: return '未知'
  }
}

const formatTime = (row) => {
  return dayjs(row.createdAt).format('YYYY-MM-DD HH:mm')
}

const formatPayTime = (row) => {
  return row.payTime ? dayjs(row.payTime).format('YYYY-MM-DD HH:mm') : '未支付'
}

const toOrderDetail = (id) => {
  router.push(`/user/order/${id}`)
}

const handlePay = async (id) => {
  try {
    await updateOrder(id, { status: 1 })
    ElMessage.success('支付成功')
    getMyOrderListApi()
  } catch (error) {
    console.error(error)
  }
}

const handleCancel = async (id) => {
  await ElMessageBox.confirm(
    '确定要取消该订单吗？',
    '提示',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    }
  )
  try {
    await updateOrder(id, { status: 2 })
    ElMessage.success('取消成功')
    getMyOrderListApi()
  } catch (error) {
    console.error(error)
  }
}
</script>

<style scoped lang="scss">
.order-list-container {
  padding: 20px;

  .page-header {
    margin-bottom: 20px;
  }

  .search-form {
    margin-bottom: 20px;
  }

  .order-table {
    margin-bottom: 20px;
  }

  .pagination {
    text-align: right;
  }
}
</style>
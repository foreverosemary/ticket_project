<template>
  <div class="activity-list-container">
    <el-page-header content="校园活动列表" class="page-header" />
    <el-form :model="searchForm" inline class="search-form" @submit.prevent="getActivityListApi">
      <el-form-item label="活动名称">
        <el-input v-model="searchForm.name" placeholder="请输入活动名称" clearable />
      </el-form-item>
      <el-form-item label="活动状态">
        <el-select v-model="searchForm.status" placeholder="请选择活动状态" clearable>
          <el-option label="未开始" value="0" />
          <el-option label="进行中" value="1" />
          <el-option label="已结束" value="2" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="getActivityListApi">查询</el-button>
        <el-button @click="resetSearch">重置</el-button>
      </el-form-item>
    </el-form>

    <el-row :gutter="20" class="activity-card-list">
      <el-col :xs="24" :sm="12" :md="8" :lg="6" v-for="item in activityList" :key="item.activityId">
        <el-card shadow="hover" class="activity-card" @click="toActivityDetail(item.activityId)">
          <div class="activity-status" :class="getStatusClass(item.status)">
            {{ getStatusText(item.status) }}
          </div>
          <h3 class="activity-name">{{ item.name }}</h3>
          <div class="activity-info">
            <p>库存：{{ item.stock }}张</p>
            <p>时间：{{ formatTime(item.startTime) }} 至 {{ formatTime(item.endTime) }}</p>
          </div>
          <el-button
            type="primary"
            size="small"
            class="grab-btn"
            :disabled="item.status !== 1 || item.stock === 0"
            @click.stop="handleGrab(item.activityId)"
          >
            {{ item.stock === 0 ? '已售罄' : '立即抢票' }}
          </el-button>
        </el-card>
      </el-col>
    </el-row>

    <el-pagination
      class="pagination"
      v-model:current-page="searchForm.pageNum"
      v-model:page-size="searchForm.pageSize"
      :total="total"
      layout="total, sizes, prev, pager, next, jumper"
      @size-change="getActivityListApi"
      @current-change="getActivityListApi"
    />

    <!-- 抢票弹窗 -->
    <el-dialog title="抢票" v-model="grabDialogVisible" width="300px">
      <el-form :model="grabForm" label-width="80px">
        <el-form-item label="抢票数量">
          <el-input-number v-model="grabForm.need" :min="1" :max="10" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="grabDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmGrab">确认抢票</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { getActivityList } from '@/api/activity'
import { createOrder } from '@/api/order'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { ElMessage } from 'element-plus'
import dayjs from 'dayjs'

const router = useRouter()
const userStore = useUserStore()
const activityList = ref([])
const total = ref(0)
const grabDialogVisible = ref(false)
const currentActivityId = ref(0)
const grabForm = reactive({ need: 1 })

const searchForm = reactive({
  name: '',
  status: '',
  pageNum: 1,
  pageSize: 12
})

onMounted(() => {
  getActivityListApi()
})

const getActivityListApi = async () => {
  try {
    // 状态转数字
    const params = {
      ...searchForm,
      status: searchForm.status ? Number(searchForm.status) : undefined
    }
    const res = await getActivityList(params)
    activityList.value = res.activities
    total.value = res.total
  } catch (error) {
    console.error(error)
  }
}

const resetSearch = () => {
  searchForm.name = ''
  searchForm.status = ''
  searchForm.pageNum = 1
  getActivityListApi()
}

const getStatusClass = (status) => {
  switch (status) {
    case 0: return 'status-unstart'
    case 1: return 'status-ing'
    case 2: return 'status-end'
    default: return ''
  }
}

const getStatusText = (status) => {
  switch (status) {
    case 0: return '未开始'
    case 1: return '进行中'
    case 2: return '已结束'
    default: return '未知'
  }
}

const formatTime = (time) => {
  return dayjs(time).format('YYYY-MM-DD HH:mm')
}

const toActivityDetail = (id) => {
  router.push(`/activities/${id}`)
}

const handleGrab = (id) => {
  if (!userStore.token) {
    ElMessage.warning('请先登录')
    router.push('/login')
    return
  }
  const activity = activityList.value.find(item => item.activityId === id)
  if (!activity) return
  currentActivityId.value = id
  grabForm.need = 1
  grabDialogVisible.value = true
}

const confirmGrab = async () => {
  if (grabForm.need < 1 || grabForm.need > 10) {
    ElMessage.error('请选择1-10张票')
    return
  }
  try {
    await createOrder({
      activityId: currentActivityId.value,
      need: grabForm.need
    })
    ElMessage.success('抢票成功，请前往我的订单查看')
    grabDialogVisible.value = false
    getActivityListApi()
  } catch (error) {
    console.error(error)
  }
}
</script>

<style scoped lang="scss">
.activity-list-container {
  padding: 20px;

  .page-header {
    margin-bottom: 20px;
  }

  .search-form {
    margin-bottom: 20px;
  }

  .activity-card-list {
    margin-bottom: 20px;

    .activity-card {
      height: 280px;
      cursor: pointer;
      position: relative;
      padding: 15px;

      .activity-status {
        position: absolute;
        top: 15px;
        right: 15px;
        padding: 2px 8px;
        border-radius: 4px;
        font-size: 12px;
        &.status-unstart {
          background: #e6a23c;
          color: #fff;
        }
        &.status-ing {
          background: #67c23a;
          color: #fff;
        }
        &.status-end {
          background: #909399;
          color: #fff;
        }
      }

      .activity-name {
        font-size: 16px;
        font-weight: 600;
        margin: 10px 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .activity-info {
        p {
          font-size: 14px;
          color: #666;
          margin: 5px 0;
        }
      }

      .grab-btn {
        position: absolute;
        bottom: 15px;
        left: 15px;
        right: 15px;
        width: calc(100% - 30px);
      }
    }
  }

  .pagination {
    text-align: right;
  }
}
</style>
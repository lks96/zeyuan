<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ArrowRight, Blocks, Database, Package, RefreshCw, Store, Users } from '@lucide/vue'
import {
  fetchHealth,
  fetchTenantSummary,
  fetchToolPackages,
  type HealthPayload,
  type TenantSummary,
  type ToolPackage,
} from '@/services/api'

const toolPackages = ref<ToolPackage[]>([])
const health = ref<HealthPayload | null>(null)
const summary = ref<TenantSummary | null>(null)
const isLoading = ref(true)
const apiError = ref('')

const activeTools = computed(() => toolPackages.value.filter((tool) => tool.status === 'active'))
const planningTools = computed(() => toolPackages.value.filter((tool) => tool.status === 'planning'))
const pausedTools = computed(() => toolPackages.value.filter((tool) => tool.status === 'paused'))
const visibleToolPackages = computed(() => toolPackages.value.slice(0, 6))

const healthText = computed(() => {
  if (apiError.value) return '服务异常'
  if (health.value?.status === 'ok' && health.value.database === 'ok') return '系统正常'
  if (health.value?.status === 'ok') return '接口在线'
  return '检测中'
})

const healthDetail = computed(() => {
  if (apiError.value) return apiError.value
  if (!summary.value) return '正在读取当前账号、店铺和工具包数据'
  const scope = summary.value.adminCanViewAll ? '管理员可查看全部店铺' : '普通用户仅查看已绑定店铺'
  return `当前身份：${currentUserText.value}，${scope}`
})

const currentUserText = computed(() => {
  if (!summary.value) return '未加载'
  return `${summary.value.currentUser.displayName} · ${summary.value.currentUser.role === 'admin' ? '管理员' : '普通用户'}`
})

const databaseLabel = computed(() => {
  if (!health.value?.database) return '未知'
  return health.value.database === 'ok' ? '正常' : '异常'
})

onMounted(loadDashboard)

async function loadDashboard() {
  isLoading.value = true
  apiError.value = ''

  try {
    const [healthPayload, packagePayload, summaryPayload] = await Promise.all([
      fetchHealth(),
      fetchToolPackages(),
      fetchTenantSummary(),
    ])
    health.value = healthPayload
    toolPackages.value = packagePayload
    summary.value = summaryPayload
  } catch {
    apiError.value = '无法加载仪表盘数据，请检查后端服务或当前账号权限。'
  } finally {
    isLoading.value = false
  }
}

function toolStatusLabel(status: ToolPackage['status']) {
  if (status === 'active') return '可用'
  if (status === 'paused') return '维护中'
  return '规划中'
}

function toolStatusColor(status: ToolPackage['status']) {
  if (status === 'active') return 'success'
  if (status === 'paused') return 'warning'
  return 'secondary'
}
</script>

<template>
  <section class="dashboard-grid">
    <div class="status-panel">
      <div>
        <p class="section-label">系统状态</p>
        <h2>{{ healthText }}</h2>
        <p class="muted">{{ healthDetail }}</p>
      </div>
      <va-chip :color="apiError ? 'warning' : 'success'" square>
        {{ apiError ? '待处理' : '正常' }}
      </va-chip>
    </div>

    <div class="system-card page-panel">
      <div class="system-card-row">
        <span class="module-icon">
          <Database :size="21" />
        </span>
        <div>
          <p class="section-label">数据库</p>
          <strong>{{ databaseLabel }}</strong>
        </div>
      </div>
      <p class="muted">接口服务：{{ health?.service || '未连接' }}</p>
    </div>
  </section>

  <section class="metric-row dashboard-metrics">
    <va-card class="metric-card">
      <va-card-content>
        <span class="metric-label">
          <Store :size="16" />
          可见店铺
        </span>
        <strong>{{ summary?.visibleShops ?? 0 }}</strong>
        <small>全部店铺 {{ summary?.totalShops ?? 0 }}</small>
      </va-card-content>
    </va-card>
    <va-card class="metric-card">
      <va-card-content>
        <span class="metric-label">
          <Users :size="16" />
          系统用户
        </span>
        <strong>{{ summary?.totalUsers ?? 0 }}</strong>
        <small>{{ currentUserText }}</small>
      </va-card-content>
    </va-card>
    <va-card class="metric-card">
      <va-card-content>
        <span class="metric-label">
          <Package :size="16" />
          已上线工具
        </span>
        <strong>{{ activeTools.length }}</strong>
        <small>共 {{ toolPackages.length }} 个工具包</small>
      </va-card-content>
    </va-card>
    <va-card class="metric-card">
      <va-card-content>
        <span class="metric-label">
          <Blocks :size="16" />
          待上线/维护
        </span>
        <strong>{{ planningTools.length + pausedTools.length }}</strong>
        <small>规划 {{ planningTools.length }}，维护 {{ pausedTools.length }}</small>
      </va-card-content>
    </va-card>
  </section>

  <section class="module-section">
    <div class="section-heading">
      <div>
        <p class="section-label">工具入口</p>
        <h2>当前工具包</h2>
      </div>
      <va-button preset="secondary" :loading="isLoading" @click="loadDashboard">
        <RefreshCw :size="17" />
        刷新
      </va-button>
    </div>

    <div class="module-grid">
      <va-card v-for="tool in visibleToolPackages" :key="tool.id" class="module-card">
        <va-card-content>
          <div class="module-card-header">
            <div class="module-icon">
              <Package :size="21" />
            </div>
            <va-chip size="small" :color="toolStatusColor(tool.status)">
              {{ toolStatusLabel(tool.status) }}
            </va-chip>
          </div>
          <h3>{{ tool.name }}</h3>
          <p>{{ tool.description }}</p>
          <div class="tool-card-meta">
            <small>{{ tool.category }}</small>
            <small>{{ tool.packageType === 'builtin' ? '内置工具' : '已安装工具' }}</small>
            <small v-if="tool.recommended">推荐</small>
          </div>
          <va-button preset="secondary" :disabled="tool.status !== 'active'" to="/tools">
            进入工具中心
            <ArrowRight :size="17" />
          </va-button>
        </va-card-content>
      </va-card>
    </div>

    <div v-if="!isLoading && visibleToolPackages.length === 0" class="empty-state">
      暂无可显示的工具包
    </div>
  </section>
</template>

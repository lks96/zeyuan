<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ArrowRight, Package, RefreshCw } from '@lucide/vue'
import {
  fetchHealth,
  fetchModules,
  fetchTenantSummary,
  type HealthPayload,
  type TenantSummary,
  type ToolModule,
} from '@/services/api'

const modules = ref<ToolModule[]>([])
const health = ref<HealthPayload | null>(null)
const summary = ref<TenantSummary | null>(null)
const isLoading = ref(true)
const apiError = ref('')

const fallbackModules: ToolModule[] = [
  {
    id: 'product-research',
    name: '商品采集',
    description: '预留 Temu 商品链接采集、SKU 信息解析和批量导入能力。',
    status: 'planning',
  },
  {
    id: 'price-monitor',
    name: '价格监控',
    description: '预留商品价格、库存和竞品变化监控能力。',
    status: 'planning',
  },
  {
    id: 'order-assistant',
    name: '订单助手',
    description: '预留订单同步、异常提醒和履约跟踪能力。',
    status: 'planning',
  },
  {
    id: 'analytics',
    name: '数据看板',
    description: '预留销售趋势、利润估算和运营指标分析能力。',
    status: 'planning',
  },
]

const healthText = computed(() => {
  if (apiError.value) return '后端未连接'
  if (health.value?.status === 'ok') return '后端在线'
  return '检测中'
})

const currentUserText = computed(() => {
  if (!summary.value) return '未加载'
  return `${summary.value.currentUser.displayName} · ${summary.value.currentUser.role === 'admin' ? '管理员' : '普通用户'}`
})

onMounted(async () => {
  try {
    const [healthPayload, modulePayload, summaryPayload] = await Promise.all([
      fetchHealth(),
      fetchModules(),
      fetchTenantSummary(),
    ])
    health.value = healthPayload
    modules.value = modulePayload
    summary.value = summaryPayload
  } catch {
    apiError.value = '无法连接到 Go API'
    modules.value = fallbackModules
  } finally {
    isLoading.value = false
  }
})
</script>

<template>
  <section class="dashboard-grid">
    <div class="status-panel">
      <div>
        <p class="section-label">系统状态</p>
        <h2>{{ healthText }}</h2>
        <p class="muted">
          {{ apiError || `当前身份：${currentUserText}` }}
        </p>
      </div>
      <va-chip :color="apiError ? 'warning' : 'success'" square>
        {{ apiError ? '待启动' : '正常' }}
      </va-chip>
    </div>

    <div class="metric-row">
      <va-card class="metric-card">
        <va-card-content>
          <span class="metric-label">可见店铺</span>
          <strong>{{ summary?.visibleShops ?? 0 }}</strong>
        </va-card-content>
      </va-card>
      <va-card class="metric-card">
        <va-card-content>
          <span class="metric-label">系统用户</span>
          <strong>{{ summary?.totalUsers ?? 0 }}</strong>
        </va-card-content>
      </va-card>
      <va-card class="metric-card">
        <va-card-content>
          <span class="metric-label">工具模块</span>
          <strong>{{ modules.length }}</strong>
        </va-card-content>
      </va-card>
    </div>
  </section>

  <section class="module-section">
    <div class="section-heading">
      <div>
        <p class="section-label">工具入口</p>
        <h2>核心能力模块</h2>
      </div>
      <va-button preset="secondary" :loading="isLoading">
        <RefreshCw :size="17" />
        刷新
      </va-button>
    </div>

    <div class="module-grid">
      <va-card v-for="module in modules" :key="module.id" class="module-card">
        <va-card-content>
          <div class="module-card-header">
            <div class="module-icon">
              <Package :size="21" />
            </div>
            <va-chip size="small" color="secondary">规划中</va-chip>
          </div>
          <h3>{{ module.name }}</h3>
          <p>{{ module.description }}</p>
          <va-button preset="secondary">
            进入
            <ArrowRight :size="17" />
          </va-button>
        </va-card-content>
      </va-card>
    </div>
  </section>
</template>

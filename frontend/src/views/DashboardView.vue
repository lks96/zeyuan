<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowRight, BarChart3, Blocks, Database, Package, RefreshCw, Store, Upload, Users, Wallet, Warehouse } from '@lucide/vue'
import {
  cachedImageUrl,
  fetchHealth,
  fetchSalesDashboard,
  fetchTenantSummary,
  fetchToolPackages,
  hasPermission,
  importSalesOverallJson,
  type HealthPayload,
  type SalesDashboard,
  type SalesDashboardPeriodMetric,
  type TenantSummary,
  type ToolPackage,
} from '@/services/api'

const router = useRouter()

const toolPackages = ref<ToolPackage[]>([])
const health = ref<HealthPayload | null>(null)
const summary = ref<TenantSummary | null>(null)
const salesDashboard = ref<SalesDashboard | null>(null)
const isLoading = ref(true)
const isImportingSales = ref(false)
const isSalesImportModalOpen = ref(false)
const isFieldMappingOpen = ref(false)
const apiError = ref('')
const salesImportNotice = ref('')
const salesImportError = ref('')
const salesSourceName = ref('listOverall.json')
const salesJsonContent = ref('')

const activeTools = computed(() => toolPackages.value.filter((tool) => tool.status === 'active'))
const planningTools = computed(() => toolPackages.value.filter((tool) => tool.status === 'planning'))
const pausedTools = computed(() => toolPackages.value.filter((tool) => tool.status === 'paused'))
const visibleToolPackages = computed(() => toolPackages.value.slice(0, 6))
const canManageTools = computed(() => hasPermission('tools:manage'))
const dashboardPeriods = computed(() => salesDashboard.value?.periods ?? defaultSalesPeriods())
const topProducts = computed(() => salesDashboard.value?.topProducts ?? [])
const fieldMappings = computed(() => salesDashboard.value?.fieldMapping ?? [])
const inventorySummary = computed(() => salesDashboard.value?.inventory)

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
    const [healthPayload, packagePayload, summaryPayload, salesPayload] = await Promise.all([
      fetchHealth(),
      fetchToolPackages(),
      fetchTenantSummary(),
      fetchSalesDashboard().catch(() => null),
    ])
    health.value = healthPayload
    toolPackages.value = packagePayload
    summary.value = summaryPayload
    salesDashboard.value = salesPayload
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

function openTool(tool: ToolPackage) {
  if (tool.status !== 'active') return

  router.push({
    name: 'tools',
    query: {
      tool: tool.id,
    },
  })
}

function defaultSalesPeriods(): SalesDashboardPeriodMetric[] {
  return [
    { key: 'today', label: '今日', salesVolume: 0, salesAmount: 0, grossProfit: 0 },
    { key: 'sevenDays', label: '7日', salesVolume: 0, salesAmount: 0, grossProfit: 0 },
    { key: 'thirtyDays', label: '30日', salesVolume: 0, salesAmount: 0, grossProfit: 0 },
  ]
}

function formatMoney(cents?: number) {
  const yuan = (cents ?? 0) / 100
  if (Math.abs(yuan) >= 10000) {
    return `¥ ${(yuan / 10000).toFixed(2)} 万`
  }
  return `¥ ${yuan.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
}

function formatDateTime(value?: string) {
  if (!value) return '暂无快照'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { hour12: false })
}

function openSalesImportModal() {
  salesJsonContent.value = ''
  salesSourceName.value = 'listOverall.json'
  salesImportError.value = ''
  isSalesImportModalOpen.value = true
}

function closeSalesImportModal() {
  isSalesImportModalOpen.value = false
}

async function submitSalesOverallJson() {
  if (!salesJsonContent.value.trim()) return
  isImportingSales.value = true
  salesImportError.value = ''
  salesImportNotice.value = ''
  try {
    const result = await importSalesOverallJson(salesSourceName.value, salesJsonContent.value)
    salesDashboard.value = result.dashboard
    salesImportNotice.value = salesImportSuccessText(result.dashboard, result.batch.importedTotal)
    isSalesImportModalOpen.value = false
  } catch (error) {
    salesImportError.value = apiErrorMessage(error)
  } finally {
    isImportingSales.value = false
  }
}

function salesImportSuccessText(dashboard: SalesDashboard, importedTotal: number) {
  const totalSalesVolume = dashboard.periods.reduce((sum, period) => sum + period.salesVolume, 0)
  if (totalSalesVolume === 0) {
    return `已导入 ${importedTotal} 条SKU明细，但当前JSON里今日/7日/30日销量都是0，销售卡片会保持0。`
  }
  return `已导入 ${importedTotal} 条SKU明细，销售大屏已刷新。`
}

function apiErrorMessage(error: unknown) {
  const maybeAxiosError = error as { response?: { data?: { error?: string } }; message?: string }
  return maybeAxiosError.response?.data?.error || maybeAxiosError.message || '导入失败，请检查JSON格式或数据库迁移是否已执行。'
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

  <section class="sales-screen">
    <div class="sales-hero">
      <div>
        <p class="section-label">经营大屏</p>
        <h2>销售总览</h2>
        <p class="muted">
          {{ salesDashboard?.latestBatch?.supplierName || '暂无店铺快照' }}
          <span v-if="salesDashboard?.latestBatch"> · {{ formatDateTime(salesDashboard.latestBatch.createdAt) }}</span>
        </p>
        <p v-if="salesImportNotice" class="sales-import-notice">{{ salesImportNotice }}</p>
      </div>
      <div class="sales-hero-actions">
        <va-button preset="secondary" @click="isFieldMappingOpen = true">
          <BarChart3 :size="17" />
          字段对应
        </va-button>
        <va-button v-if="canManageTools" color="primary" :loading="isImportingSales" @click="openSalesImportModal">
          <Upload :size="17" />
          导入销售JSON
        </va-button>
      </div>
    </div>

    <div class="sales-period-grid">
      <article v-for="period in dashboardPeriods" :key="period.key" class="sales-period-card">
        <div class="sales-card-top">
          <strong>{{ period.label }}</strong>
          <span class="sales-card-icon"><Wallet :size="18" /></span>
        </div>
        <dl>
          <div>
            <dt>销量</dt>
            <dd>{{ period.salesVolume }}</dd>
          </div>
          <div>
            <dt>销售额</dt>
            <dd>{{ formatMoney(period.salesAmount) }}</dd>
          </div>
          <div>
            <dt>毛利</dt>
            <dd>{{ formatMoney(period.grossProfit) }}</dd>
          </div>
        </dl>
      </article>
    </div>

    <div class="sales-detail-grid">
      <section class="sales-panel">
        <div class="section-heading compact-heading">
          <div>
            <p class="section-label">库存与备货</p>
            <h2>风险概览</h2>
          </div>
          <Warehouse :size="22" />
        </div>
        <div class="inventory-grid">
          <span><small>缺货数量</small><strong>{{ inventorySummary?.lackQuantity ?? 0 }}</strong></span>
          <span><small>建议备货量</small><strong>{{ inventorySummary?.adviceQuantity ?? 0 }}</strong></span>
          <span><small>仓内可用</small><strong>{{ inventorySummary?.warehouseInventoryNum ?? 0 }}</strong></span>
          <span><small>预占用</small><strong>{{ inventorySummary?.expectedOccupiedInventoryNum ?? 0 }}</strong></span>
          <span><small>暂不可用</small><strong>{{ inventorySummary?.unavailableWarehouseInventoryNum ?? 0 }}</strong></span>
          <span><small>待发货</small><strong>{{ inventorySummary?.waitReceiveNum ?? 0 }}</strong></span>
        </div>
      </section>

      <section class="sales-panel">
        <div class="section-heading compact-heading">
          <div>
            <p class="section-label">近30日</p>
            <h2>商品排行</h2>
          </div>
        </div>
        <div v-if="topProducts.length" class="top-product-list">
          <button v-for="product in topProducts" :key="product.productSkcId" class="top-product-item" type="button">
            <img :src="cachedImageUrl(product.productImage)" alt="" />
            <span>
              <strong>{{ product.productName }}</strong>
              <small>SKC {{ product.productSkcId }} · 30日销量 {{ product.lastThirtyDaysSaleVolume }}</small>
            </span>
            <em>{{ formatMoney(product.salesAmount) }}</em>
          </button>
        </div>
        <div v-else class="empty-state compact-empty">导入销售 JSON 后显示商品排行</div>
      </section>
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
      <va-card
        v-for="tool in visibleToolPackages"
        :key="tool.id"
        class="module-card module-card-clickable"
        :class="{ disabled: tool.status !== 'active' }"
        role="button"
        tabindex="0"
        @click="openTool(tool)"
        @keydown.enter.prevent="openTool(tool)"
        @keydown.space.prevent="openTool(tool)"
      >
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
          <va-button preset="secondary" :disabled="tool.status !== 'active'" @click.stop="openTool(tool)">
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

  <div v-if="isSalesImportModalOpen" class="modal-backdrop" @click.self="closeSalesImportModal">
    <form class="modal-panel modal-panel-wide" @submit.prevent="submitSalesOverallJson">
      <div class="modal-header">
        <h2>导入销售总览 JSON</h2>
        <va-button preset="secondary" @click="closeSalesImportModal">关闭</va-button>
      </div>
      <p class="modal-hint">粘贴 listOverall 接口返回数据，系统会保存为最新销售快照并刷新首页大屏。</p>
      <p v-if="salesImportError" class="modal-error">{{ salesImportError }}</p>
      <label class="field-control">
        <span>来源名称</span>
        <input v-model.trim="salesSourceName" placeholder="listOverall.json" />
      </label>
      <label class="field-control">
        <span>JSON内容</span>
        <textarea v-model.trim="salesJsonContent" class="json-textarea" required placeholder="粘贴接口返回 JSON"></textarea>
      </label>
      <div class="modal-actions">
        <va-button preset="secondary" type="button" @click="closeSalesImportModal">取消</va-button>
        <va-button color="primary" type="submit" :loading="isImportingSales">导入</va-button>
      </div>
    </form>
  </div>

  <div v-if="isFieldMappingOpen" class="modal-backdrop" @click.self="isFieldMappingOpen = false">
    <section class="modal-panel modal-panel-wide">
      <div class="modal-header">
        <h2>字段对应</h2>
        <va-button preset="secondary" @click="isFieldMappingOpen = false">关闭</va-button>
      </div>
      <div class="field-mapping-list">
        <article v-for="mapping in fieldMappings" :key="mapping.label">
          <strong>{{ mapping.label }}</strong>
          <code>{{ mapping.path }}</code>
          <p>{{ mapping.note }}</p>
        </article>
      </div>
    </section>
  </div>
</template>

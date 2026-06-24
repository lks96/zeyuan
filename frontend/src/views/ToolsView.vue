<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Blocks,
  ChevronLeft,
  ChevronRight,
  ClipboardPaste,
  Download,
  FileJson,
  Plus,
  RefreshCw,
  Search,
  Upload,
  X,
} from '@lucide/vue'
import {
  batchUpdateProductCollectionMaintenance,
  exportLatestDeliveryExtractBatch,
  exportToolPackageArchive,
  fetchLatestDeliveryExtractBatch,
  fetchProductCollectionProducts,
  fetchShops,
  fetchToolPackages,
  hasPermission,
  importDeliveryExtractJson,
  importProductCollectionJson,
  updateProductCollectionProduct,
  type DeliveryExtractBatch,
  type ProductCollectionList,
  type ProductCollectionProduct,
  type Shop,
  type ToolPackage,
} from '@/services/api'

const defaultPageSize = 10
const activeToolStorageKey = 'temu-tools-active-tool'
const recentToolsStorageKey = 'temu-tools-recent-tools'
const extensionArchivePath = '/downloads/temu-seller-sync-extension.zip'

const route = useRoute()
const router = useRouter()

const canManageTools = computed(() => hasPermission('tools:manage'))

const toolPackages = ref<ToolPackage[]>([])
const shops = ref<Shop[]>([])
const latestBatch = ref<DeliveryExtractBatch | null>(null)
const productCollection = ref<ProductCollectionList | null>(null)
const isLoading = ref(true)
const isImporting = ref(false)
const isProductImporting = ref(false)
const isProductSearching = ref(false)
const isProductSaving = ref(false)
const isExporting = ref(false)
const exportingToolId = ref('')
const isSearching = ref(false)
const apiError = ref('')
const successMessage = ref('')
const productSearchText = ref('')
const productActiveQuery = ref('')
const productStatusFilter = ref('100')
const productCurrentPage = ref(1)
const selectedShopId = ref<number | ''>('')
const searchText = ref('')
const activeQuery = ref('')
const currentPage = ref(1)
const selectedDeliveryRowIds = ref<number[]>([])
const selectedDate = ref(todayInputValue())
const fileInput = ref<HTMLInputElement | null>(null)
const productFileInput = ref<HTMLInputElement | null>(null)
const isPasteModalOpen = ref(false)
const pastedJson = ref('')
const isProductPasteModalOpen = ref(false)
const pastedProductJson = ref('')
const isProductBatchModalOpen = ref(false)
const productBatchContent = ref('')
const isProductBatchUpdating = ref(false)
const editingProduct = ref<ProductCollectionProduct | null>(null)
const editingProductConfig = ref('')
const editingCostPrice = ref('')
const previewImageUrl = ref('')
const previewImageTitle = ref('')
const isToolPickerOpen = ref(false)
const toolSearchText = ref('')
const selectedToolCategory = ref('全部')
const activeToolId = ref(routeToolId() || localStorage.getItem(activeToolStorageKey) || 'product-research')
const recentToolIds = ref(loadRecentToolIds())

const latestRows = computed(() => latestBatch.value?.data ?? [])
const rowsTotal = computed(() => latestBatch.value?.rowsTotal ?? latestBatch.value?.extractedTotal ?? 0)
const totalPages = computed(() => Math.max(1, Math.ceil(rowsTotal.value / defaultPageSize)))
const rangeStart = computed(() => (rowsTotal.value === 0 ? 0 : (currentPage.value - 1) * defaultPageSize + 1))
const rangeEnd = computed(() => Math.min(currentPage.value * defaultPageSize, rowsTotal.value))
const currentDeliveryRowIds = computed(() => latestRows.value.map((row) => row.id))
const selectedDeliveryRowCount = computed(() => selectedDeliveryRowIds.value.length)
const allCurrentDeliveryRowsSelected = computed(() => {
  return currentDeliveryRowIds.value.length > 0 && currentDeliveryRowIds.value.every((rowId) => selectedDeliveryRowIds.value.includes(rowId))
})
const productRows = computed(() => productCollection.value?.data ?? [])
const productRowsTotal = computed(() => productCollection.value?.rowsTotal ?? 0)
const productTotalPages = computed(() => Math.max(1, Math.ceil(productRowsTotal.value / defaultPageSize)))
const productRangeStart = computed(() => (productRowsTotal.value === 0 ? 0 : (productCurrentPage.value - 1) * defaultPageSize + 1))
const productRangeEnd = computed(() => Math.min(productCurrentPage.value * defaultPageSize, productRowsTotal.value))
const toolCards = computed(() => {
  return toolPackages.value.map((toolPackage, index) => {
    return {
      ...toolPackage,
      iconComponent: toolIcon(toolPackage.icon),
      heightClass: index % 3 === 1 ? 'tool-card-tall' : index % 3 === 2 ? 'tool-card-compact' : '',
      isCurrent: toolPackage.id === activeToolId.value,
      isRecent: recentToolIds.value.includes(toolPackage.id),
      isRecommended: toolPackage.recommended,
      isAvailable: toolPackage.status === 'active' && canRenderToolPackage(toolPackage),
    }
  })
})
const filteredToolCards = computed(() => {
  const keyword = toolSearchText.value.trim().toLowerCase()
  return toolCards.value.filter((tool) => {
    const matchesCategory =
      selectedToolCategory.value === '全部' ||
      (selectedToolCategory.value === '最近使用' && tool.isRecent) ||
      tool.category === selectedToolCategory.value
    const matchesKeyword =
      !keyword ||
      tool.name.toLowerCase().includes(keyword) ||
      tool.description.toLowerCase().includes(keyword) ||
      tool.category.toLowerCase().includes(keyword)
    return matchesCategory && matchesKeyword
  })
})
const toolCategoryOptions = computed(() => {
  const categories = new Set(toolCards.value.map((tool) => tool.category))
  return ['全部', '最近使用', ...Array.from(categories)]
})
const activeToolCard = computed(() => {
  return toolCards.value.find((tool) => tool.id === activeToolId.value) ?? toolCards.value.find((tool) => tool.isAvailable) ?? null
})

onMounted(loadToolCenter)

watch(
  () => route.query.tool,
  () => {
    applyRouteToolSelection()
  },
)

async function loadToolCenter() {
  isLoading.value = true
  apiError.value = ''
  currentPage.value = 1
  clearSelectedDeliveryRows()

  try {
    const [packagePayload, latestPayload, productPayload, shopPayload] = await Promise.all([
      fetchToolPackages(),
      fetchLatestDeliveryExtractBatch(buildLatestExtractQuery()),
      fetchProductCollectionProducts(buildProductQuery()),
      fetchShops(),
    ])
    toolPackages.value = packagePayload
    latestBatch.value = latestPayload
    const latestDate = batchDateToInputValue(latestPayload?.date)
    selectedDate.value = latestDate || todayInputValue()
    productCollection.value = productPayload
    shops.value = shopPayload
    ensureActiveTool(packagePayload)
    if (!selectedShopId.value && shopPayload.length > 0) {
      selectedShopId.value = shopPayload[0].id
    }
  } catch {
    apiError.value = '无法加载工具中心数据，请检查后端服务或当前账号权限。'
  } finally {
    isLoading.value = false
  }
}

function ensureActiveTool(packagePayload: ToolPackage[]) {
  const routeTool = routeToolId()
  if (routeTool) {
    const target = packagePayload.find((toolPackage) => toolPackage.id === routeTool)
    if (target?.status === 'active' && canRenderToolPackage(target)) {
      activateTool(target.id)
      return
    }
  }

  const current = packagePayload.find((toolPackage) => toolPackage.id === activeToolId.value)
  if (current?.status === 'active' && canRenderToolPackage(current)) return

  const fallback = packagePayload.find((toolPackage) => toolPackage.status === 'active' && canRenderToolPackage(toolPackage))
  if (fallback) {
    activeToolId.value = fallback.id
    localStorage.setItem(activeToolStorageKey, fallback.id)
  }
}

function openToolPicker() {
  toolSearchText.value = ''
  selectedToolCategory.value = '全部'
  isToolPickerOpen.value = true
}

function closeToolPicker() {
  isToolPickerOpen.value = false
}

function selectTool(tool: (typeof toolCards.value)[number]) {
  if (!tool.isAvailable) {
    apiError.value = `${tool.name} 当前不可用，请稍后再试。`
    return
  }

  activateTool(tool.id)
  apiError.value = ''
  successMessage.value = ''
  isToolPickerOpen.value = false

  if (route.query.tool !== tool.id) {
    router.replace({
      name: 'tools',
      query: {
        ...route.query,
        tool: tool.id,
      },
    })
  }
}

function activateTool(toolId: string) {
  activeToolId.value = toolId
  localStorage.setItem(activeToolStorageKey, toolId)
  recordRecentTool(toolId)
}

function applyRouteToolSelection() {
  const targetId = routeToolId()
  if (!targetId || targetId === activeToolId.value) return

  const target = toolPackages.value.find((toolPackage) => toolPackage.id === targetId)
  if (target?.status === 'active' && canRenderToolPackage(target)) {
    activateTool(target.id)
    isToolPickerOpen.value = false
    apiError.value = ''
  }
}

function routeToolId() {
  const tool = route.query.tool
  return typeof tool === 'string' ? tool.trim() : ''
}

async function exportToolPackage(tool: (typeof toolCards.value)[number]) {
  if (!canManageTools.value || exportingToolId.value) return

  exportingToolId.value = tool.id
  apiError.value = ''
  successMessage.value = ''

  try {
    const { blob, filename } = await exportToolPackageArchive(tool.id)
    downloadBlob(blob, filename)
    successMessage.value = `已导出工具包：${tool.name}。`
  } catch {
    apiError.value = '导出工具包失败，请确认后端服务已更新并且当前账号有工具管理权限。'
  } finally {
    exportingToolId.value = ''
  }
}

function canRenderToolPackage(toolPackage: ToolPackage) {
  if (toolPackage.entryType === 'iframe') return Boolean(toolPackage.entryPath)
  return toolPackage.panelKey === 'product-research' || toolPackage.panelKey === 'delivery-json-extract'
}

function toolIcon(icon: string) {
  if (icon === 'file-json') return FileJson
  if (icon === 'search') return Search
  return Blocks
}

function toolStatusLabel(status: ToolPackage['status'], available: boolean) {
  if (status === 'active' && available) return '可用'
  if (status === 'paused') return '维护中'
  return '即将上线'
}

function toolStatusColor(status: ToolPackage['status'], available: boolean) {
  if (status === 'active' && available) return 'success'
  if (status === 'paused') return 'warning'
  return 'secondary'
}

function loadRecentToolIds() {
  try {
    const parsed = JSON.parse(localStorage.getItem(recentToolsStorageKey) || '[]') as string[]
    return Array.isArray(parsed) ? parsed.slice(0, 4) : []
  } catch {
    return []
  }
}

function recordRecentTool(toolId: string) {
  recentToolIds.value = [toolId, ...recentToolIds.value.filter((id) => id !== toolId)].slice(0, 4)
  localStorage.setItem(recentToolsStorageKey, JSON.stringify(recentToolIds.value))
}

async function loadProductCollection() {
  isProductSearching.value = true
  apiError.value = ''

  try {
    productCollection.value = await fetchProductCollectionProducts(buildProductQuery())
  } catch {
    apiError.value = '无法加载商品采集数据，请稍后重试。'
  } finally {
    isProductSearching.value = false
  }
}

async function loadLatestExtract() {
  isSearching.value = true
  apiError.value = ''

  try {
    latestBatch.value = await fetchLatestDeliveryExtractBatch(buildExtractQuery())
  } catch {
    apiError.value = '无法加载导入结果，请稍后重试。'
  } finally {
    isSearching.value = false
  }
}

function buildProductQuery() {
  return {
    page: productCurrentPage.value,
    pageSize: defaultPageSize,
    q: productActiveQuery.value || undefined,
    status: productStatusFilter.value === '' ? undefined : Number(productStatusFilter.value),
  }
}

function buildExtractQuery() {
  return {
    page: currentPage.value,
    pageSize: defaultPageSize,
    q: activeQuery.value || undefined,
    batchDate: selectedDate.value || undefined,
  }
}

function buildLatestExtractQuery() {
  return {
    page: 1,
    pageSize: defaultPageSize,
    q: activeQuery.value || undefined,
  }
}

function buildExtractExportQuery() {
  return {
    q: activeQuery.value || undefined,
    batchDate: selectedDate.value || undefined,
    rowIds: selectedDeliveryRowIds.value.length > 0 ? selectedDeliveryRowIds.value : undefined,
  }
}

async function changeExtractDate() {
  clearSelectedDeliveryRows()
  currentPage.value = 1
  await loadLatestExtract()
}

function selectedShopIdNumber() {
  return Number(selectedShopId.value || 0)
}

function ensureSelectedShop() {
  if (selectedShopIdNumber() > 0) return true
  apiError.value = '请先选择店铺后再导入商品数据。'
  return false
}

function triggerProductFileImport() {
  productFileInput.value?.click()
}

function triggerFileImport() {
  fileInput.value?.click()
}

async function importProductSelectedFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  try {
    const content = await file.text()
    await importProductJsonContent(content, file.name)
  } finally {
    input.value = ''
  }
}

async function importSelectedFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  try {
    const content = await file.text()
    await importJsonContent(content, file.name)
  } finally {
    input.value = ''
  }
}

function openProductPasteModal() {
  pastedProductJson.value = ''
  apiError.value = ''
  successMessage.value = ''
  isProductPasteModalOpen.value = true
}

function closeProductPasteModal() {
  isProductPasteModalOpen.value = false
}

function openProductBatchModal() {
  productBatchContent.value = ''
  apiError.value = ''
  successMessage.value = ''
  isProductBatchModalOpen.value = true
}

function closeProductBatchModal() {
  isProductBatchModalOpen.value = false
}

function openPasteModal() {
  pastedJson.value = ''
  apiError.value = ''
  successMessage.value = ''
  isPasteModalOpen.value = true
}

function closePasteModal() {
  isPasteModalOpen.value = false
}

async function exportDeliveryRows() {
  if (!latestBatch.value) return

  isExporting.value = true
  apiError.value = ''

  try {
    const { blob, filename } = await exportLatestDeliveryExtractBatch(buildExtractExportQuery())
    downloadBlob(blob, filename)
  } catch {
    apiError.value = '导出失败，请稍后重试。'
  } finally {
    isExporting.value = false
  }
}

function downloadBlob(blob: Blob, filename: string) {
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  link.remove()
  window.URL.revokeObjectURL(url)
}

async function importPastedProductJson() {
  await importProductJsonContent(pastedProductJson.value, 'pasted-product-json')
  closeProductPasteModal()
}

async function submitProductBatchMaintenance() {
  const content = productBatchContent.value.trim()
  if (!content) {
    apiError.value = '请先粘贴批量设置内容。'
    return
  }

  isProductBatchUpdating.value = true
  apiError.value = ''
  successMessage.value = ''

  try {
    const result = await batchUpdateProductCollectionMaintenance(content, buildProductQuery())
    productCollection.value = result.products
    const missing = result.notFoundSkcs.length > 0 ? `，未找到 SKC：${result.notFoundSkcs.join('、')}` : ''
    successMessage.value = `已批量处理 ${result.total} 条，成功更新 ${result.updated} 条${missing}。`
    closeProductBatchModal()
  } catch {
    apiError.value = '批量设置失败，请检查格式是否为 SKC,成本,配置，并用分号分隔每条记录。'
  } finally {
    isProductBatchUpdating.value = false
  }
}

async function importProductJsonContent(content: string, sourceName: string) {
  const cleanContent = content.trim()
  if (!cleanContent) {
    apiError.value = '请先选择 JSON 文件或粘贴商品 JSON 内容。'
    return
  }
  if (!ensureSelectedShop()) return

  isProductImporting.value = true
  apiError.value = ''
  successMessage.value = ''

  try {
    productSearchText.value = ''
    productActiveQuery.value = ''
    productCurrentPage.value = 1
    const result = await importProductCollectionJson(
      {
        sourceName,
        content: cleanContent,
        shopId: selectedShopIdNumber(),
      },
      buildProductQuery(),
    )
    productCollection.value = result.products
    successMessage.value = `已导入 ${result.imported} 条商品记录，绑定店铺：${result.shop.shopName}。`
  } catch {
    apiError.value = '商品导入失败，请确认 JSON 内容格式正确。'
  } finally {
    isProductImporting.value = false
  }
}

async function importPastedJson() {
  await importJsonContent(pastedJson.value, 'pasted-json')
  closePasteModal()
}

async function importJsonContent(content: string, sourceName: string) {
  const cleanContent = content.trim()
  if (!cleanContent) {
    apiError.value = '请先选择 JSON 文件或粘贴 JSON 内容。'
    return
  }

  isImporting.value = true
  apiError.value = ''
  successMessage.value = ''

    try {
      searchText.value = ''
      activeQuery.value = ''
      clearSelectedDeliveryRows()
      currentPage.value = 1
      latestBatch.value = await importDeliveryExtractJson({
        sourceName,
        content: cleanContent,
    })
    const importedDate = batchDateToInputValue(latestBatch.value.date)
    if (importedDate) {
      selectedDate.value = importedDate
    }
    successMessage.value = `已提取 ${latestBatch.value.extractedTotal} 条记录，并保存到数据库。`
  } catch {
    apiError.value = '导入失败，请确认 JSON 内容格式正确。'
  } finally {
    isImporting.value = false
  }
}

async function searchProducts() {
  productActiveQuery.value = productSearchText.value.trim()
  productCurrentPage.value = 1
  await loadProductCollection()
}

async function changeProductStatusFilter() {
  productCurrentPage.value = 1
  await loadProductCollection()
}

async function clearProductSearch() {
  if (!productSearchText.value && !productActiveQuery.value && productStatusFilter.value === '') return

  productSearchText.value = ''
  productActiveQuery.value = ''
  productStatusFilter.value = ''
  productCurrentPage.value = 1
  await loadProductCollection()
}

async function searchDeliveryRows() {
  activeQuery.value = searchText.value.trim()
  clearSelectedDeliveryRows()
  currentPage.value = 1
  await loadLatestExtract()
}

async function clearDeliverySearch() {
  if (!searchText.value && !activeQuery.value) return

  searchText.value = ''
  activeQuery.value = ''
  clearSelectedDeliveryRows()
  currentPage.value = 1
  await loadLatestExtract()
}

async function goToProductPage(page: number) {
  if (page < 1 || page > productTotalPages.value || page === productCurrentPage.value) return

  productCurrentPage.value = page
  await loadProductCollection()
}

async function goToPage(page: number) {
  if (page < 1 || page > totalPages.value || page === currentPage.value) return

  currentPage.value = page
  await loadLatestExtract()
}

function isDeliveryRowSelected(rowId: number) {
  return selectedDeliveryRowIds.value.includes(rowId)
}

function toggleDeliveryRow(rowId: number, checked: boolean) {
  if (checked && !selectedDeliveryRowIds.value.includes(rowId)) {
    selectedDeliveryRowIds.value = [...selectedDeliveryRowIds.value, rowId]
  }
  if (!checked) {
    selectedDeliveryRowIds.value = selectedDeliveryRowIds.value.filter((selectedRowId) => selectedRowId !== rowId)
  }
}

function toggleCurrentDeliveryRows(checked: boolean) {
  if (checked) {
    const rowIds = new Set([...selectedDeliveryRowIds.value, ...currentDeliveryRowIds.value])
    selectedDeliveryRowIds.value = Array.from(rowIds)
    return
  }

  const currentRowIds = new Set(currentDeliveryRowIds.value)
  selectedDeliveryRowIds.value = selectedDeliveryRowIds.value.filter((rowId) => !currentRowIds.has(rowId))
}

function clearSelectedDeliveryRows() {
  selectedDeliveryRowIds.value = []
}

function openEditProduct(product: ProductCollectionProduct) {
  editingProduct.value = product
  editingProductConfig.value = product.productConfig || ''
  editingCostPrice.value = centsToYuanInput(product.costPrice)
}

function closeEditProduct() {
  editingProduct.value = null
}

function openImagePreview(url: string, title = '') {
  previewImageUrl.value = url
  previewImageTitle.value = title
}

function closeImagePreview() {
  previewImageUrl.value = ''
  previewImageTitle.value = ''
}

async function saveEditingProduct() {
  if (!editingProduct.value) return

  isProductSaving.value = true
  apiError.value = ''

  try {
    const updated = await updateProductCollectionProduct(editingProduct.value.id, {
      productConfig: editingProductConfig.value.trim(),
      costPrice: yuanToCents(editingCostPrice.value),
    })
    productCollection.value = {
      ...(productCollection.value ?? { data: [], rowsTotal: 0, page: productCurrentPage.value, pageSize: defaultPageSize }),
      data: productRows.value.map((product) => (product.id === updated.id ? updated : product)),
    }
    successMessage.value = '商品维护字段已保存。'
    closeEditProduct()
  } catch {
    apiError.value = '保存商品维护字段失败，请稍后重试。'
  } finally {
    isProductSaving.value = false
  }
}

function centsToYuan(value?: number) {
  return ((value ?? 0) / 100).toFixed(2)
}

function centsToYuanInput(value?: number) {
  if (!value) return ''
  return centsToYuan(value)
}

function yuanToCents(value: string) {
  const parsed = Number(value)
  if (!Number.isFinite(parsed) || parsed <= 0) return 0
  return Math.round(parsed * 100)
}

function productStatusLabel(status: number) {
  const labels: Record<number, string> = {
    0: '未发布到站点',
    100: '在售中',
    200: '已下架/已终止',
    300: '已删除',
  }
  return labels[status] ?? `未知状态 ${status}`
}

function formatDateTime(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString('zh-CN', { hour12: false })
}

function todayInputValue() {
  const now = new Date()
  const local = new Date(now.getTime() - now.getTimezoneOffset() * 60_000)
  return local.toISOString().slice(0, 10)
}

function batchDateToInputValue(batchDate?: string) {
  if (!batchDate || batchDate.length !== 6) return ''
  return `20${batchDate.slice(0, 2)}-${batchDate.slice(2, 4)}-${batchDate.slice(4, 6)}`
}
</script>

<template>
  <section class="tool-hub-shell">
    <div class="tool-command-panel">
      <div>
        <p class="section-label">工具中心</p>
        <h2>{{ activeToolCard?.name || '选择工具' }}</h2>
        <p>{{ activeToolCard?.description || '选择一个工具开始工作。' }}</p>
      </div>
      <div class="tool-command-actions">
        <va-chip v-if="activeToolCard" size="small" color="success">
          当前工具
        </va-chip>
        <a class="tool-extension-download" :href="extensionArchivePath" download>
          <Download :size="18" />
          下载浏览器插件
        </a>
        <va-button @click="openToolPicker">
          <Blocks :size="18" />
          切换工具
        </va-button>
      </div>
    </div>

    <va-alert v-if="apiError" color="warning" dense>
      {{ apiError }}
    </va-alert>
    <va-alert v-if="successMessage" color="success" dense>
      {{ successMessage }}
    </va-alert>

    <Transition name="tool-stage" mode="out-in">
      <section v-if="isToolPickerOpen" class="tool-picker-panel" key="tool-picker">
        <div class="tool-picker-header">
          <div>
            <p class="section-label">应用抽屉</p>
            <h2>选择工具</h2>
          </div>
          <va-button preset="secondary" @click="closeToolPicker">
            <X :size="18" />
            返回当前工具
          </va-button>
        </div>

        <div class="tool-picker-controls">
          <va-input v-model="toolSearchText" placeholder="搜索工具名称、说明或分类">
            <template #prependInner>
              <Search :size="16" />
            </template>
          </va-input>
          <div class="tool-category-tabs">
            <button
              v-for="category in toolCategoryOptions"
              :key="category"
              type="button"
              :class="{ active: selectedToolCategory === category }"
              @click="selectedToolCategory = category"
            >
              {{ category }}
            </button>
          </div>
        </div>

        <div class="tool-card-grid">
          <article
            v-for="tool in filteredToolCards"
            :key="tool.id"
            class="tool-switch-card"
            :class="[tool.heightClass, { active: tool.isCurrent, disabled: !tool.isAvailable }]"
            role="button"
            tabindex="0"
            @click="selectTool(tool)"
            @keydown.enter.prevent="selectTool(tool)"
            @keydown.space.prevent="selectTool(tool)"
          >
            <span class="tool-card-glow"></span>
            <span class="tool-card-topline">
              <span class="tool-card-icon">
                <component :is="tool.iconComponent" :size="22" />
              </span>
              <va-chip size="small" :color="toolStatusColor(tool.status, tool.isAvailable)">
                {{ toolStatusLabel(tool.status, tool.isAvailable) }}
              </va-chip>
            </span>
            <strong>{{ tool.name }}</strong>
            <span class="tool-card-desc">{{ tool.description }}</span>
            <span class="tool-card-meta">
              <small>{{ tool.category }}</small>
              <small v-if="tool.isRecommended">推荐</small>
              <small v-if="tool.isRecent">最近使用</small>
              <small v-if="tool.isCurrent">当前选中</small>
            </span>
            <span class="tool-card-actions">
              <span class="tool-card-action">{{ tool.isAvailable ? '进入工具' : '暂不可用' }}</span>
              <button
                v-if="canManageTools"
                class="tool-export-button"
                type="button"
                :disabled="Boolean(exportingToolId)"
                @click.stop="exportToolPackage(tool)"
              >
                <Download :size="15" />
                {{ exportingToolId === tool.id ? '导出中' : '导出工具包' }}
              </button>
            </span>
          </article>
        </div>

        <div v-if="filteredToolCards.length === 0" class="empty-state">
          没有匹配的工具
        </div>
      </section>
    </Transition>
  </section>

  <section v-if="false" class="page-panel">
    <div class="section-heading">
      <div>
        <p class="section-label">工具中心</p>
        <h2>模块管理</h2>
      </div>
      <va-button v-if="canManageTools" preset="secondary">
        <Plus :size="18" />
        添加模块
      </va-button>
    </div>

    <va-alert v-if="apiError" color="warning" dense>
      {{ apiError }}
    </va-alert>
    <va-alert v-if="successMessage" color="success" dense>
      {{ successMessage }}
    </va-alert>

    <div class="tool-list" :aria-busy="isLoading">
      <div v-for="module in toolPackages" :key="module.id" class="tool-row">
        <div>
          <strong>{{ module.name }}</strong>
          <span>{{ module.description }}</span>
        </div>
        <va-chip size="small" :color="module.status === 'active' ? 'success' : module.status === 'paused' ? 'warning' : 'secondary'">
          {{ module.status }}
        </va-chip>
      </div>
    </div>
  </section>

  <section v-if="!isToolPickerOpen && activeToolCard?.entryType === 'iframe'" class="page-panel tool-panel-card tool-iframe-panel">
    <iframe
      :src="activeToolCard.entryPath"
      :title="activeToolCard.name"
      loading="lazy"
      sandbox="allow-forms allow-same-origin allow-scripts allow-downloads"
    ></iframe>
  </section>

  <section v-if="!isToolPickerOpen && activeToolCard?.panelKey === 'product-research'" class="page-panel tool-panel-card">
    <div class="section-heading">
      <div>
        <p class="section-label">商品工具</p>
        <h2>商品采集</h2>
      </div>
      <div class="segmented-actions">
        <va-button preset="secondary" :disabled="isLoading || isProductSearching" @click="loadProductCollection">
          <RefreshCw :size="18" />
          刷新
        </va-button>
      </div>
    </div>

    <div class="extract-control-bar product-control-bar">
      <label class="field-control product-shop-field">
        <span>导入店铺</span>
        <select v-model="selectedShopId">
          <option value="">请选择店铺</option>
          <option v-for="shop in shops" :key="shop.id" :value="shop.id">
            {{ shop.shopName }}（{{ shop.externalCode || '无编号' }}）
          </option>
        </select>
      </label>
      <div class="segmented-actions extract-import-actions">
        <input
          ref="productFileInput"
          class="hidden-file-input"
          type="file"
          accept=".json,application/json"
          @change="importProductSelectedFile"
        />
        <va-button v-if="canManageTools" preset="secondary" :disabled="isProductImporting" @click="triggerProductFileImport">
          <Upload :size="18" />
          上传 JSON 文件
        </va-button>
        <va-button v-if="canManageTools" preset="secondary" :disabled="isProductImporting" @click="openProductPasteModal">
          <ClipboardPaste :size="18" />
          粘贴 JSON
        </va-button>
        <va-button v-if="canManageTools" preset="secondary" :disabled="isProductBatchUpdating" @click="openProductBatchModal">
          <ClipboardPaste :size="18" />
          批量设置成本/配置
        </va-button>
      </div>
    </div>

    <div class="extract-toolbar product-toolbar">
      <va-input
        v-model="productSearchText"
        class="extract-search-input"
        :disabled="isProductSearching"
        placeholder="搜索 SKC / SKU / 名称"
        @keyup.enter="searchProducts"
      >
        <template #prependInner>
          <Search :size="16" />
        </template>
      </va-input>
      <label class="product-status-filter" aria-label="商品状态">
        <select v-model="productStatusFilter" :disabled="isProductSearching" @change="changeProductStatusFilter">
          <option value="">全部</option>
          <option value="0">未发布到站点</option>
          <option value="100">在售中</option>
          <option value="200">已下架/已终止</option>
          <option value="300">已删除</option>
        </select>
      </label>
      <va-button :disabled="isProductSearching" :loading="isProductSearching" @click="searchProducts">
        查询
      </va-button>
      <va-button preset="secondary" :disabled="!productActiveQuery && !productSearchText && productStatusFilter === ''" @click="clearProductSearch">
        <X :size="17" />
      </va-button>
    </div>

    <div v-if="productActiveQuery" class="extract-filter-hint">
      当前筛选：{{ productActiveQuery }}
    </div>

    <div class="data-table">
      <div class="data-table-row data-table-head product-table-row">
        <span>主图</span>
        <span>商品</span>
        <span>根数 / 配置</span>
        <span>价格 / 成本</span>
        <span>状态</span>
        <span>创建时间</span>
        <span>操作</span>
      </div>
      <div v-if="productRows.length === 0" class="empty-state">
        暂无商品采集数据
      </div>
      <div v-for="product in productRows" :key="product.id" class="data-table-row product-table-row">
        <span class="product-image-cell">
          <button
            v-if="product.mainImageUrl"
            class="image-preview-trigger"
            type="button"
            aria-label="预览商品图片"
            @click="openImagePreview(product.mainImageUrl, product.productName)"
          >
            <img :src="product.mainImageUrl" alt="" />
          </button>
          <span v-else>-</span>
        </span>
        <span class="product-info-cell">
          <strong>{{ product.productName }}</strong>
          <small>SKC {{ product.productSkcId }} / SKU {{ product.productSkuId || '-' }}</small>
        </span>
        <span>
          {{ product.numberOfPiecesNew || 0 }}P
          <small>{{ product.productConfig || '未维护配置' }}</small>
        </span>
        <span>
          ¥{{ centsToYuan(product.supplierPrice) }}
          <small>成本 ¥{{ centsToYuan(product.costPrice) }}</small>
        </span>
        <span>{{ productStatusLabel(product.skcTopStatus) }}</span>
        <span>{{ formatDateTime(product.createdAt) }}</span>
        <span class="row-actions">
          <va-button v-if="canManageTools" preset="secondary" size="small" @click="openEditProduct(product)">
            编辑
          </va-button>
        </span>
      </div>
    </div>

    <div class="pagination-bar">
      <span>共 {{ productRowsTotal }} 条，显示 {{ productRangeStart }}-{{ productRangeEnd }}，每页 {{ defaultPageSize }} 条</span>
      <div class="segmented-actions">
        <va-button preset="secondary" :disabled="productCurrentPage <= 1 || isProductSearching" @click="goToProductPage(productCurrentPage - 1)">
          <ChevronLeft :size="18" />
          上一页
        </va-button>
        <span class="page-indicator">{{ productCurrentPage }} / {{ productTotalPages }}</span>
        <va-button preset="secondary" :disabled="productCurrentPage >= productTotalPages || isProductSearching" @click="goToProductPage(productCurrentPage + 1)">
          下一页
          <ChevronRight :size="18" />
        </va-button>
      </div>
    </div>
  </section>

  <section v-if="!isToolPickerOpen && activeToolCard?.panelKey === 'delivery-json-extract'" class="page-panel tool-panel-card">
    <div class="section-heading">
      <div>
        <p class="section-label">JSON 工具</p>
        <h2>发货 JSON 提取</h2>
      </div>
      <div class="segmented-actions">
        <va-button preset="secondary" :disabled="isLoading || isSearching" @click="loadToolCenter">
          <RefreshCw :size="18" />
          刷新
        </va-button>
      </div>
    </div>

    <div class="extract-control-bar">
      <label class="field-control extract-date-field">
        <span>查询日期</span>
        <input v-model="selectedDate" type="date" @change="changeExtractDate" />
      </label>
      <div class="segmented-actions extract-import-actions">
        <input
          ref="fileInput"
          class="hidden-file-input"
          type="file"
          accept=".json,application/json"
          @change="importSelectedFile"
        />
        <va-button v-if="canManageTools" :loading="isImporting" @click="triggerFileImport">
          <Upload :size="18" />
          上传 JSON 文件
        </va-button>
        <va-button v-if="canManageTools" preset="secondary" :disabled="isImporting" @click="openPasteModal">
          <ClipboardPaste :size="18" />
          粘贴 JSON
        </va-button>
        <va-button preset="secondary" :loading="isExporting" :disabled="!latestBatch || isSearching" @click="exportDeliveryRows">
          <Download :size="18" />
          导出 Excel
        </va-button>
      </div>
    </div>

    <div class="extract-toolbar">
      <va-input
        v-model="searchText"
        class="extract-search-input"
        :disabled="!latestBatch || isSearching"
        placeholder="搜索 SKC / SKU / 商品名 / 发货单 / 发货批次"
        @keyup.enter="searchDeliveryRows"
      >
        <template #prependInner>
          <Search :size="16" />
        </template>
      </va-input>
      <va-button :disabled="!latestBatch || isSearching" :loading="isSearching" @click="searchDeliveryRows">
        查询
      </va-button>
      <va-button preset="secondary" :disabled="!activeQuery && !searchText" @click="clearDeliverySearch">
        <X :size="17" />
      </va-button>
    </div>

    <div v-if="activeQuery" class="extract-filter-hint">
      当前筛选：{{ activeQuery }}
    </div>

    <div v-if="!latestBatch" class="empty-state">
      当前日期暂无提取结果
    </div>

    <div v-else class="data-table">
      <div class="data-table-row data-table-head extract-table-row">
        <span class="selection-cell">
          <input
            type="checkbox"
            aria-label="选择当前页"
            :checked="allCurrentDeliveryRowsSelected"
            :disabled="latestRows.length === 0"
            @change="toggleCurrentDeliveryRows(($event.target as HTMLInputElement).checked)"
          />
        </span>
        <span>店铺</span>
        <span>商品</span>
        <span>发货批次</span>
        <span>发货单</span>
        <span>SKC / 数量</span>
        <span>SKU / 数量</span>
        <span>收货人</span>
      </div>
      <div v-if="latestRows.length === 0" class="empty-state">
        没有匹配的导入记录
      </div>
      <div v-for="row in latestRows" :key="row.id" class="data-table-row extract-table-row">
        <span class="selection-cell">
          <input
            type="checkbox"
            :aria-label="`选择 ${row.deliveryOrderSn}`"
            :checked="isDeliveryRowSelected(row.id)"
            @change="toggleDeliveryRow(row.id, ($event.target as HTMLInputElement).checked)"
          />
        </span>
        <span>
          <strong>{{ row.shopName || row.supplierId || '-' }}</strong>
        </span>
        <span class="extract-product-cell">
          <button
            v-if="row.productSkcPicture"
            class="image-preview-trigger"
            type="button"
            aria-label="预览商品图片"
            @click="openImagePreview(row.productSkcPicture, row.productName)"
          >
            <img :src="row.productSkcPicture" alt="" />
          </button>
          <strong>{{ row.productName }}</strong>
        </span>
        <span>{{ row.expressBatchSn || '-' }}</span>
        <span>{{ row.deliveryOrderSn }}</span>
        <span>{{ row.SKC }} / {{ row.skcNum }}</span>
        <span>{{ row.SKU }} / {{ row.skuNum }}</span>
        <span>{{ row.receiverName }}</span>
      </div>
    </div>

    <div v-if="latestBatch" class="pagination-bar">
      <span>
        共 {{ rowsTotal }} 条，显示 {{ rangeStart }}-{{ rangeEnd }}，每页 {{ defaultPageSize }} 条
        <template v-if="selectedDeliveryRowCount > 0">，已选 {{ selectedDeliveryRowCount }} 条</template>
      </span>
      <div class="segmented-actions">
        <va-button preset="secondary" :disabled="currentPage <= 1 || isSearching" @click="goToPage(currentPage - 1)">
          <ChevronLeft :size="18" />
          上一页
        </va-button>
        <span class="page-indicator">{{ currentPage }} / {{ totalPages }}</span>
        <va-button preset="secondary" :disabled="currentPage >= totalPages || isSearching" @click="goToPage(currentPage + 1)">
          下一页
          <ChevronRight :size="18" />
        </va-button>
      </div>
    </div>
  </section>

  <div v-if="previewImageUrl" class="modal-backdrop image-preview-backdrop" @click.self="closeImagePreview">
    <div class="image-preview-panel">
      <button type="button" class="icon-only image-preview-close" aria-label="关闭图片预览" @click="closeImagePreview">
        <X :size="18" />
      </button>
      <img :src="previewImageUrl" :alt="previewImageTitle || '商品图片预览'" />
      <strong v-if="previewImageTitle">{{ previewImageTitle }}</strong>
    </div>
  </div>

  <div v-if="isProductBatchModalOpen" class="modal-backdrop" @click.self="closeProductBatchModal">
    <form class="modal-panel modal-panel-wide" @submit.prevent="submitProductBatchMaintenance">
      <div class="modal-header">
        <h2>批量设置成本/配置</h2>
        <button type="button" class="icon-only" aria-label="关闭批量设置弹窗" @click="closeProductBatchModal">
          <X :size="18" />
        </button>
      </div>

      <p class="modal-hint">
        每条记录用分号分隔；单条记录按 SKC、成本、配置解析。成本按元填写，保存时自动转成分。
      </p>

      <label class="field-control">
        <span>批量内容</span>
        <textarea
          v-model.trim="productBatchContent"
          class="json-textarea"
          required
          placeholder="88353153792,10,&#10;白色尤加利2P&#10;银色铁丝兰2P;&#10;58322371048,12.5,&#10;黄色泡沫玫瑰1P;"
        ></textarea>
      </label>

      <div class="modal-actions">
        <va-button preset="secondary" type="button" @click="closeProductBatchModal">取消</va-button>
        <va-button type="submit" :loading="isProductBatchUpdating">
          批量保存
        </va-button>
      </div>
    </form>
  </div>

  <div v-if="isProductPasteModalOpen" class="modal-backdrop" @click.self="closeProductPasteModal">
    <form class="modal-panel modal-panel-wide" @submit.prevent="importPastedProductJson">
      <div class="modal-header">
        <h2>粘贴商品 JSON 数据</h2>
        <button type="button" class="icon-only" aria-label="关闭商品 JSON 粘贴弹窗" @click="closeProductPasteModal">
          <X :size="18" />
        </button>
      </div>

      <label class="field-control">
        <span>JSON 内容</span>
        <textarea v-model.trim="pastedProductJson" class="json-textarea" required placeholder="粘贴商品 JSON 内容"></textarea>
      </label>

      <div class="modal-actions">
        <va-button preset="secondary" type="button" @click="closeProductPasteModal">取消</va-button>
        <va-button type="submit" :loading="isProductImporting">
          <FileJson :size="18" />
          解析入库
        </va-button>
      </div>
    </form>
  </div>

  <div v-if="editingProduct" class="modal-backdrop" @click.self="closeEditProduct">
    <form class="modal-panel" @submit.prevent="saveEditingProduct">
      <div class="modal-header">
        <h2>编辑商品维护字段</h2>
        <button type="button" class="icon-only" aria-label="关闭商品编辑弹窗" @click="closeEditProduct">
          <X :size="18" />
        </button>
      </div>

      <div class="edit-product-summary">
        <img v-if="editingProduct.mainImageUrl" :src="editingProduct.mainImageUrl" alt="" />
        <div>
          <strong>{{ editingProduct.productName }}</strong>
          <span>SKC {{ editingProduct.productSkcId }}</span>
        </div>
      </div>

      <label class="field-control">
        <span>产品配置</span>
        <textarea v-model.trim="editingProductConfig" class="json-textarea product-config-textarea" placeholder="手动维护产品配置"></textarea>
      </label>

      <label class="field-control">
        <span>成本价格（元）</span>
        <input v-model.trim="editingCostPrice" type="number" min="0" step="0.01" placeholder="0.00" />
      </label>

      <div class="modal-actions">
        <va-button preset="secondary" type="button" @click="closeEditProduct">取消</va-button>
        <va-button type="submit" :loading="isProductSaving">保存</va-button>
      </div>
    </form>
  </div>

  <div v-if="isPasteModalOpen" class="modal-backdrop" @click.self="closePasteModal">
    <form class="modal-panel modal-panel-wide" @submit.prevent="importPastedJson">
      <div class="modal-header">
        <h2>粘贴 JSON 数据</h2>
        <button type="button" class="icon-only" aria-label="关闭 JSON 粘贴弹窗" @click="closePasteModal">
          <X :size="18" />
        </button>
      </div>

      <label class="field-control">
        <span>JSON 内容</span>
        <textarea v-model.trim="pastedJson" class="json-textarea" required placeholder="粘贴 source JSON 内容"></textarea>
      </label>

      <div class="modal-actions">
        <va-button preset="secondary" type="button" @click="closePasteModal">取消</va-button>
        <va-button type="submit" :loading="isImporting">
          <FileJson :size="18" />
          解析入库
        </va-button>
      </div>
    </form>
  </div>
</template>

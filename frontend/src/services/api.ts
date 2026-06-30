import axios from 'axios'

export interface ApiEnvelope<T> {
  data: T
}

export interface HealthPayload {
  status: string
  service: string
  database?: string
  timestamp: string
}

export interface ToolModule {
  id: string
  name: string
  description: string
  status: 'planning' | 'active' | 'paused'
  sortOrder?: number
}

export interface ToolPackage {
  id: string
  version: string
  name: string
  description: string
  category: string
  icon: string
  status: 'planning' | 'active' | 'paused'
  packageType: 'builtin' | 'installed'
  entryType: 'native' | 'iframe'
  entryPath: string
  panelKey: string
  removable: boolean
  recommended: boolean
  sortOrder: number
  permissions: string[]
  manifestJson: string
  installedAt: string
  updatedAt: string
}

export interface DeliveryExtractRow {
  id: number
  batchId?: number
  supplierId: string
  shopName?: string
  productName: string
  productSkcPicture: string
  deliveryOrderSn: string
  expressBatchSn: string
  expectPickUpGoodsTime: number
  SKC: string
  skcNum: number
  SKU: string
  skuNum: number
  receiverName: string
  createdAt?: string
}

export interface DeliveryExtractBatch {
  id: number
  sourceFile: string
  date?: string
  sourceTotal: number
  extractedTotal: number
  rowsTotal?: number
  page?: number
  pageSize?: number
  query?: string
  data?: DeliveryExtractRow[]
  createdAt: string
}

export interface DeliveryExtractQuery {
  q?: string
  batchDate?: string
  rowIds?: number[]
  page?: number
  pageSize?: number
}

export interface DeliveryExtractImportPayload {
  sourceName: string
  content: string
}

export interface ProductCollectionProduct {
  id: number
  productSkcId: string
  productSkuId: string
  mainImageUrl: string
  productName: string
  numberOfPiecesNew: number
  productConfig: string
  supplierPrice: number
  costPrice: number
  skcTopStatus: number
  createdAt?: string
  supplierId: string
  shopId?: number
  shopName?: string
  importedAt: string
  updatedAt: string
}

export interface ProductCollectionList {
  data: ProductCollectionProduct[]
  rowsTotal: number
  page: number
  pageSize: number
  query?: string
}

export interface ProductCollectionQuery {
  q?: string
  shopId?: number
  status?: number
  page?: number
  pageSize?: number
}

export interface ProductCollectionImportPayload {
  sourceName?: string
  content?: string
  shopId: number
}

export interface ProductCollectionImportResult {
  sourceTotal: number
  imported: number
  shop: Shop
  products: ProductCollectionList
}

export interface ProductCollectionBatchUpdateResult {
  total: number
  updated: number
  notFoundSkcs: string[]
  products: ProductCollectionList
}

export interface UpdateProductCollectionProductPayload {
  productConfig: string
  costPrice: number
}

export interface User {
  id: number
  username: string
  displayName: string
  role: 'admin' | 'user'
  status: 'active' | 'disabled'
  createdAt: string
}

export interface LoginCredentials {
  username: string
  password: string
}

export interface LoginPayload {
  user: User
  token: string
  permissions: string[]
}

export interface CreateUserPayload {
  username: string
  password: string
  displayName: string
  role: User['role']
  status?: User['status']
}

export interface UpdateUserPayload {
  password?: string
  displayName: string
  role: User['role']
  status: User['status']
}

export interface CreateShopPayload {
  shopName: string
  platform?: string
  externalCode?: string
  euRepresentative?: string
  status?: Shop['status']
}

export interface UpdateShopPayload {
  shopName: string
  platform: string
  externalCode?: string
  euRepresentative?: string
  status: Shop['status']
}

export interface AssignShopPayload {
  shopId: number
  shopRole: 'owner' | 'operator' | 'viewer'
}

export interface SystemSetting {
  key: string
  value: string
  description: string
  updatedAt: string
}

export interface Permission {
  code: string
  name: string
  category: string
  description: string
  createdAt: string
}

export interface Shop {
  id: number
  shopName: string
  platform: string
  externalCode: string
  euRepresentative: string
  shopUrl: string
  status: 'active' | 'paused' | 'closed'
  shopRole?: 'owner' | 'operator' | 'viewer'
  createdAt: string
}

export interface UserShop {
  userId: number
  shopId: number
  shopName: string
  shopRole: 'owner' | 'operator' | 'viewer'
  createdAt: string
}

export interface TenantSummary {
  currentUser: User
  totalUsers: number
  totalShops: number
  visibleShops: number
  adminCanViewAll: boolean
}

export interface SalesDashboardPeriodMetric {
  key: string
  label: string
  salesVolume: number
  salesAmount: number
  grossProfit: number
}

export interface SalesInventorySummary {
  lackQuantity: number
  adviceQuantity: number
  warehouseInventoryNum: number
  expectedOccupiedInventoryNum: number
  unavailableWarehouseInventoryNum: number
  waitDeliveryInventoryNum: number
  waitReceiveNum: number
  waitApproveInventoryNum: number
  sellerWarehouseStock: number
}

export interface SalesTopProduct {
  productSkcId: string
  productName: string
  productImage: string
  supplierId: string
  supplierName: string
  lastThirtyDaysSaleVolume: number
  lastSevenDaysSaleVolume: number
  todaySaleVolume: number
  warehouseInventoryNum: number
  waitReceiveNum: number
  unavailableWarehouseInventoryNum: number
  lackQuantity: number
  salesAmount: number
  grossProfit: number
}

export interface SalesOverallBatch {
  id: number
  sourceName: string
  supplierId: string
  supplierName: string
  sourceTotal: number
  importedTotal: number
  createdAt: string
}

export interface SalesFieldMapping {
  label: string
  path: string
  note: string
}

export interface SalesDashboard {
  latestBatch?: SalesOverallBatch
  periods: SalesDashboardPeriodMetric[]
  inventory: SalesInventorySummary
  topProducts: SalesTopProduct[]
  fieldMapping: SalesFieldMapping[]
}

export interface SalesOverallImportResult {
  batch: SalesOverallBatch
  dashboard: SalesDashboard
}

export const api = axios.create({
  baseURL: '/api',
  timeout: 8000,
})

const authUserKey = 'temu-tools-user'
const authTokenKey = 'temu-tools-token'
const authPermissionsKey = 'temu-tools-permissions'

api.interceptors.request.use((config) => {
  const token = getStoredToken()
  const user = getStoredUser()
  if (token) {
    config.headers.set('Authorization', `Bearer ${token}`)
  } else if (user) {
    config.headers.set('X-User-ID', String(user.id))
  }
  return config
})

export function getStoredUser() {
  const rawUser = localStorage.getItem(authUserKey)
  if (!rawUser) return null

  try {
    return JSON.parse(rawUser) as User
  } catch {
    localStorage.removeItem(authUserKey)
    return null
  }
}

export function getStoredToken() {
  return localStorage.getItem(authTokenKey)
}

export function getStoredPermissions() {
  const rawPermissions = localStorage.getItem(authPermissionsKey)
  if (!rawPermissions) return []

  try {
    const permissions = JSON.parse(rawPermissions) as string[]
    return Array.isArray(permissions) ? permissions : []
  } catch {
    localStorage.removeItem(authPermissionsKey)
    return []
  }
}

export function hasPermission(permissionCode?: string) {
  if (!permissionCode) return true
  return getStoredPermissions().includes(permissionCode)
}

export function setStoredSession(payload: LoginPayload) {
  localStorage.setItem(authUserKey, JSON.stringify(payload.user))
  if (payload.token) {
    localStorage.setItem(authTokenKey, payload.token)
  }
  localStorage.setItem(authPermissionsKey, JSON.stringify(payload.permissions))
}

export function clearStoredSession() {
  localStorage.removeItem(authUserKey)
  localStorage.removeItem(authTokenKey)
  localStorage.removeItem(authPermissionsKey)
}

export async function login(credentials: LoginCredentials) {
  const response = await api.post<ApiEnvelope<LoginPayload>>('/auth/login', credentials)
  setStoredSession(response.data.data)
  return response.data.data
}

export async function fetchCurrentSession() {
  const response = await api.get<ApiEnvelope<LoginPayload>>('/me')
  setStoredSession(response.data.data)
  return response.data.data
}

export async function fetchHealth() {
  const response = await api.get<ApiEnvelope<HealthPayload>>('/health')
  return response.data.data
}

export async function fetchModules() {
  const response = await api.get<ApiEnvelope<ToolModule[]>>('/modules')
  return response.data.data
}

export async function fetchToolPackages() {
  try {
    const response = await api.get<ApiEnvelope<ToolPackage[]>>('/tool-packages')
    return response.data.data
  } catch {
    const modules = await fetchModules()
    return modules.map(moduleToToolPackage)
  }
}

export async function exportToolPackageArchive(toolId: string) {
  const response = await api.get<Blob>(`/tool-packages/${toolId}/export`, {
    responseType: 'blob',
  })
  const disposition = response.headers['content-disposition'] as string | undefined
  const filename = filenameFromDisposition(disposition) || `${toolId}.tool.zip`
  return { blob: response.data, filename }
}

export async function exportExtensionArchive(apiBase: string) {
  const response = await api.get<Blob>('/extension/archive', {
    params: { apiBase },
    responseType: 'blob',
  })
  const disposition = response.headers['content-disposition'] as string | undefined
  const filename = filenameFromDisposition(disposition) || 'temu-seller-sync-extension.zip'
  return { blob: response.data, filename }
}

function moduleToToolPackage(module: ToolModule): ToolPackage {
  const meta = legacyToolMeta(module)
  const now = new Date().toISOString()
  return {
    id: module.id,
    version: '1.0.0',
    name: meta.name,
    description: meta.description,
    category: meta.category,
    icon: meta.icon,
    status: module.status,
    packageType: 'builtin',
    entryType: 'native',
    entryPath: '',
    panelKey: module.id,
    removable: false,
    recommended: module.id === 'product-research',
    sortOrder: module.sortOrder ?? 100,
    permissions: ['tools:view', 'tools:manage'],
    manifestJson: '{}',
    installedAt: now,
    updatedAt: now,
  }
}

function legacyToolMeta(module: ToolModule) {
  const overrides: Record<string, { name: string; description: string; category: string; icon: string }> = {
    'product-research': {
      name: '商品采集',
      description: '导入店铺商品 JSON，维护 SKC、SKU、价格、成本和产品配置。',
      category: '店铺运营工具',
      icon: 'search',
    },
    'delivery-json-extract': {
      name: '发货 JSON 提取',
      description: '解析发货单 JSON，支持查询、分页和 Excel 导出。',
      category: '数据工具',
      icon: 'file-json',
    },
    'price-monitor': {
      name: '价格监控',
      description: '跟踪商品价格、库存和竞品变化。',
      category: '自动化工具',
      icon: 'blocks',
    },
    'order-assistant': {
      name: '订单助手',
      description: '订单同步、异常提醒和履约跟踪。',
      category: '店铺运营工具',
      icon: 'blocks',
    },
    analytics: {
      name: '数据看板',
      description: '销售趋势、利润估算和运营指标分析。',
      category: '数据工具',
      icon: 'blocks',
    },
  }

  return overrides[module.id] ?? {
    name: module.name,
    description: module.description,
    category: '店铺运营工具',
    icon: 'blocks',
  }
}

export async function saveModule(payload: ToolModule) {
  const response = await api.post<ApiEnvelope<ToolModule>>('/modules', payload)
  return response.data.data
}

export async function updateModule(moduleId: string, payload: Omit<ToolModule, 'id'>) {
  const response = await api.put<ApiEnvelope<ToolModule>>(`/modules/${moduleId}`, payload)
  return response.data.data
}

export async function deleteModule(moduleId: string) {
  const response = await api.delete<ApiEnvelope<{ deleted: boolean }>>(`/modules/${moduleId}`)
  return response.data.data
}

export async function fetchLatestDeliveryExtractBatch(params: DeliveryExtractQuery = {}) {
  const response = await api.get<ApiEnvelope<DeliveryExtractBatch | null>>('/tools/delivery-extractions/latest', { params })
  return response.data.data
}

export async function exportLatestDeliveryExtractBatch(params: DeliveryExtractQuery = {}) {
  const response = await api.get<Blob>('/tools/delivery-extractions/latest/export', {
    params,
    responseType: 'blob',
  })
  const disposition = response.headers['content-disposition'] as string | undefined
  const filename = filenameFromDisposition(disposition) || 'delivery-extract.xlsx'
  return { blob: response.data, filename }
}

export async function importDeliveryExtractJson(payload: DeliveryExtractImportPayload) {
  const response = await api.post<ApiEnvelope<DeliveryExtractBatch>>('/tools/delivery-extractions/import-json', payload)
  return response.data.data
}

export async function fetchProductCollectionProducts(params: ProductCollectionQuery = {}) {
  const response = await api.get<ApiEnvelope<ProductCollectionList>>('/tools/product-collection/products', { params: cleanQueryParams(params) })
  return response.data.data
}

export async function exportProductCollectionProducts(params: ProductCollectionQuery = {}) {
  const response = await api.get<Blob>('/tools/product-collection/products/export', {
    params: cleanQueryParams(params),
    responseType: 'blob',
    timeout: 120000,
  })
  const disposition = response.headers['content-disposition'] as string | undefined
  const filename = filenameFromDisposition(disposition) || 'product-collection.xlsx'
  return { blob: response.data, filename }
}

export async function importProductCollectionJson(payload: ProductCollectionImportPayload, params: ProductCollectionQuery = {}) {
  const response = await api.post<ApiEnvelope<ProductCollectionImportResult>>('/tools/product-collection/import-json', payload, { params: cleanQueryParams(params) })
  return response.data.data
}

export async function updateProductCollectionProduct(productId: number, payload: UpdateProductCollectionProductPayload) {
  const response = await api.put<ApiEnvelope<ProductCollectionProduct>>(`/tools/product-collection/products/${productId}`, payload)
  return response.data.data
}

export async function batchUpdateProductCollectionMaintenance(content: string, params: ProductCollectionQuery = {}) {
  const response = await api.post<ApiEnvelope<ProductCollectionBatchUpdateResult>>(
    '/tools/product-collection/products/batch-maintenance',
    { content },
    { params: cleanQueryParams(params) },
  )
  return response.data.data
}

function cleanQueryParams(params: object) {
  return Object.fromEntries(
    Object.entries(params).filter(([, value]) => {
      if (value === undefined || value === null || value === '') return false
      if (typeof value === 'number' && Number.isNaN(value)) return false
      return true
    }),
  )
}

export function cachedImageUrl(url?: string) {
  if (!url) return ''
  return `/api/image-cache?url=${encodeURIComponent(url)}`
}

function filenameFromDisposition(disposition?: string) {
  if (!disposition) return ''
  const encodedMatch = disposition.match(/filename\*\s*=\s*([^;]+)/i)
  if (encodedMatch?.[1]) {
    const encodedValue = encodedMatch[1].trim().replace(/^"|"$/g, '')
    const filenameValue = encodedValue.includes("''") ? encodedValue.split("''").slice(1).join("''") : encodedValue
    try {
      return decodeURIComponent(filenameValue)
    } catch {
      return filenameValue
    }
  }

  const match = disposition.match(/filename="?([^";]+)"?/i)
  return match?.[1] ?? ''
}

export async function fetchTenantSummary() {
  const response = await api.get<ApiEnvelope<TenantSummary>>('/tenant/summary')
  return response.data.data
}

export async function fetchSalesDashboard() {
  const response = await api.get<ApiEnvelope<SalesDashboard>>('/dashboard/sales-overall')
  return response.data.data
}

export async function importSalesOverallJson(sourceName: string, content: string) {
  const response = await api.post<ApiEnvelope<SalesOverallImportResult>>('/dashboard/sales-overall/import-json', {
    sourceName,
    content,
  })
  return response.data.data
}

export async function fetchShops() {
  const response = await api.get<ApiEnvelope<Shop[]>>('/shops')
  return response.data.data
}

export async function fetchUsers() {
  const response = await api.get<ApiEnvelope<User[]>>('/users')
  return response.data.data
}

export async function fetchSettings() {
  const response = await api.get<ApiEnvelope<SystemSetting[]>>('/settings')
  return response.data.data
}

export async function updateSettings(values: Record<string, string>) {
  const response = await api.put<ApiEnvelope<SystemSetting[]>>('/settings', { values })
  return response.data.data
}

export async function fetchPermissions() {
  const response = await api.get<ApiEnvelope<Permission[]>>('/permissions')
  return response.data.data
}

export async function fetchRolePermissions(role: User['role']) {
  const response = await api.get<ApiEnvelope<string[]>>(`/roles/${role}/permissions`)
  return response.data.data
}

export async function updateRolePermissions(role: User['role'], permissions: string[]) {
  const response = await api.put<ApiEnvelope<string[]>>(`/roles/${role}/permissions`, { permissions })
  return response.data.data
}

export async function createUser(payload: CreateUserPayload) {
  const response = await api.post<ApiEnvelope<User>>('/users', payload)
  return response.data.data
}

export async function updateUser(userId: number, payload: UpdateUserPayload) {
  const response = await api.put<ApiEnvelope<User>>(`/users/${userId}`, payload)
  return response.data.data
}

export async function fetchUserShops(userId: number) {
  const response = await api.get<ApiEnvelope<UserShop[]>>(`/users/${userId}/shops`)
  return response.data.data
}

export async function disableUser(userId: number) {
  const response = await api.delete<ApiEnvelope<{ disabled: boolean }>>(`/users/${userId}`)
  return response.data.data
}

export async function createShop(payload: CreateShopPayload) {
  const response = await api.post<ApiEnvelope<Shop>>('/shops', payload)
  return response.data.data
}

export async function updateShop(shopId: number, payload: UpdateShopPayload) {
  const response = await api.put<ApiEnvelope<Shop>>(`/shops/${shopId}`, payload)
  return response.data.data
}

export async function closeShop(shopId: number) {
  const response = await api.delete<ApiEnvelope<{ closed: boolean }>>(`/shops/${shopId}`)
  return response.data.data
}

export async function assignShopToUser(userId: number, payload: AssignShopPayload) {
  const response = await api.post<ApiEnvelope<{ assigned: boolean }>>(`/users/${userId}/shops`, payload)
  return response.data.data
}

export async function removeShopFromUser(userId: number, shopId: number) {
  const response = await api.delete<ApiEnvelope<{ removed: boolean }>>(`/users/${userId}/shops/${shopId}`)
  return response.data.data
}

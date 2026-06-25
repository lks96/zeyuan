import './config.js'

const settingsKey = 'settings'
const latestKey = 'latestCaptures'
const syncKey = 'lastSync'
const productSnapshotKey = 'productLatestSnapshot'
const deliverySnapshotKey = 'deliveryLatestSnapshot'

const packagedConfig = globalThis.TEMU_TOOLS_EXTENSION_CONFIG || {}

const defaultSettings = {
  apiBase: packagedConfig.apiBase || 'http://localhost:8080/api',
  token: '',
  shopId: '',
  shopName: '',
  autoProductSync: false,
  autoDeliverySync: false,
}

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  handleMessage(message, sender)
    .then(sendResponse)
    .catch((error) => sendResponse({ ok: false, error: error.message || String(error) }))
  return true
})

async function handleMessage(message, sender) {
  switch (message?.type) {
    case 'CAPTURE_RESPONSE':
      await saveCapture(message.capture, sender)
      return { ok: true }
    case 'PAGE_READY':
      scheduleAutoChecks(sender)
      return { ok: true }
    case 'GET_STATE':
      return { ok: true, ...(await getState()) }
    case 'SAVE_SETTINGS':
      return { ok: true, settings: await saveSettings(message.settings || {}) }
    case 'LOGIN':
      return { ok: true, settings: await login(message.credentials || {}) }
    case 'FETCH_SHOPS':
      return { ok: true, shops: await fetchShops() }
    case 'LOGOUT':
      return { ok: true, settings: await saveSettings({ token: '', shopId: '', shopName: '' }) }
    case 'SYNC_CAPTURE':
      return { ok: true, result: await syncCapture(message.kind) }
    case 'FETCH_ALL_CAPTURE':
      return { ok: true, capture: await fetchAllCapture(message.kind) }
    case 'CHECK_LATEST_PRODUCT':
      return { ok: true, result: await checkLatestProduct({ manual: true }) }
    case 'CHECK_LATEST_DELIVERY':
      return { ok: true, result: await checkLatestDelivery({ manual: true }) }
    case 'SYNC_SELLER_SHOP':
      return { ok: true, result: await syncSellerShop() }
    case 'CLEAR_CAPTURE':
      return { ok: true, latestCaptures: await clearCapture(message.kind) }
    case 'INJECT_ACTIVE_TAB':
      return { ok: true, injected: await injectActiveTab() }
    default:
      return { ok: false, error: 'Unknown message type' }
  }
}

async function getState() {
  const data = await chrome.storage.local.get([settingsKey, latestKey, syncKey])
  const storedSettings = data[settingsKey] || {}
  return {
    settings: {
      ...defaultSettings,
      ...storedSettings,
      apiBase: defaultSettings.apiBase,
    },
    latestCaptures: data[latestKey] || {},
    lastSync: data[syncKey] || null,
  }
}

async function saveSettings(partial) {
  const current = (await getState()).settings
  const next = {
    ...current,
    apiBase: defaultSettings.apiBase,
    token: String(partial.token ?? current.token ?? '').trim(),
    shopId: String(partial.shopId ?? current.shopId ?? '').trim(),
    shopName: String(partial.shopName ?? current.shopName ?? '').trim(),
    autoProductSync: Boolean(partial.autoProductSync ?? current.autoProductSync),
    autoDeliverySync: Boolean(partial.autoDeliverySync ?? current.autoDeliverySync),
  }
  await chrome.storage.local.set({ [settingsKey]: next })
  return next
}

function scheduleAutoChecks(sender) {
  const tabId = sender.tab?.id || 0
  if (!tabId) return
  setTimeout(() => {
    checkLatestProduct({ tabId, frameId: sender.frameId || 0, manual: false }).catch(() => {})
    checkLatestDelivery({ tabId, frameId: sender.frameId || 0, manual: false }).catch(() => {})
  }, 3000)
}

async function login(credentials) {
  const username = String(credentials.username || '').trim()
  const password = String(credentials.password || '')
  if (!username || !password) throw new Error('请输入用户名和密码')

  const response = await apiRequest('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
    skipAuth: true,
  })

  return saveSettings({ token: response.token })
}

async function fetchShops() {
  const response = await apiRequest('/shops')
  return Array.isArray(response) ? response : []
}

async function syncSellerShop() {
  const state = await getState()
  if (!state.settings.token) {
    throw new Error('请先登录 Temu Tools')
  }

  const target = await productTabTarget({}, state)
  const response = await chrome.tabs.sendMessage(
    target.tabId,
    { type: 'FETCH_SELLER_USER_INFO' },
    { frameId: target.frameId || 0 },
  )
  if (!response?.ok) {
    throw new Error(response?.error || '获取卖家店铺信息失败')
  }

  const sourceName = `seller-user-info-${compactTimestamp(response.capture.capturedAt)}.json`
  const result = await apiRequest('/extension/seller-shops/sync', {
    method: 'POST',
    body: JSON.stringify({
      sourceName,
      content: JSON.stringify(response.capture.data),
    }),
  })

  const shops = Array.isArray(result?.shops) ? result.shops : []
  if (shops.length > 0) {
    await saveSettings({
      shopId: String(shops[0].id),
      shopName: shops[0].shopName || '',
    })
  }

  return result
}

async function syncCapture(kind) {
  if (kind !== 'product' && kind !== 'delivery') {
    throw new Error('不支持的数据类型')
  }

  const state = await getState()
  const capture = state.latestCaptures?.[kind]
  if (!capture?.data) throw new Error('还没有捕获到可同步的数据')

  const sourceName = `temu-seller-${kind}-${compactTimestamp(capture.capturedAt)}.json`
  const body = {
    sourceName,
    content: JSON.stringify(capture.data),
  }

  let endpoint = '/tools/delivery-extractions/import-json'
  if (kind === 'product') {
    const shopId = Number(state.settings.shopId || 0)
    if (!shopId) throw new Error('请先选择要绑定的店铺')
    endpoint = '/tools/product-collection/import-json'
    body.shopId = shopId
  }

  const result = await apiRequest(endpoint, {
    method: 'POST',
    body: JSON.stringify(body),
  })

  const syncInfo = {
    kind,
    syncedAt: new Date().toISOString(),
    sourceName,
    requestUrl: capture.requestUrl,
  }
  await chrome.storage.local.set({ [syncKey]: syncInfo })
  return result
}

async function clearCapture(kind) {
  const state = await getState()
  const next = { ...(state.latestCaptures || {}) }
  if (kind) {
    delete next[kind]
  } else {
    delete next.product
    delete next.delivery
  }
  await chrome.storage.local.set({ [latestKey]: next })
  return next
}

async function saveCapture(capture, sender) {
  if (!capture?.kind || !capture.data) return
  const kind = capture.kind === 'product' ? 'product' : capture.kind === 'delivery' ? 'delivery' : ''
  if (!kind) return

  const state = await getState()
  const latest = {
    ...(state.latestCaptures || {}),
    [kind]: {
      kind,
      data: capture.data,
      method: capture.method || 'GET',
      requestUrl: capture.requestUrl || '',
      requestHeaders: capture.requestHeaders || {},
      requestBody: capture.requestBody || '',
      status: capture.status || 0,
      pageUrl: capture.pageUrl || sender.tab?.url || '',
      pageTitle: capture.pageTitle || sender.tab?.title || '',
      tabId: sender.tab?.id || 0,
      frameId: sender.frameId || 0,
      capturedAt: capture.capturedAt || new Date().toISOString(),
      itemCount: capture.itemCount || countLikelyItems(capture.data, kind),
      autoFetched: Boolean(capture.autoFetched),
      autoFetchTotal: capture.autoFetchTotal || 0,
    },
  }
  await chrome.storage.local.set({ [latestKey]: latest })
}

async function fetchAllCapture(kind) {
  if (kind !== 'product' && kind !== 'delivery') {
    throw new Error('不支持的数据类型')
  }

  const state = await getState()
  const capture = state.latestCaptures?.[kind]
  if (!capture?.requestUrl || !capture.requestBody) {
    throw new Error('还没有捕获到可复用的列表请求')
  }
  if (!capture.tabId) {
    throw new Error('缺少来源标签页，请在卖家中心页面重新触发一次接口')
  }

  const response = await chrome.tabs.sendMessage(
    capture.tabId,
    {
      type: 'FETCH_ALL_CAPTURE',
      kind,
      capture,
    },
    { frameId: capture.frameId || 0 },
  )

  if (!response?.ok) {
    throw new Error(response?.error || '抓取全部失败')
  }

  await saveCapture(response.capture, {
    tab: { id: capture.tabId, url: capture.pageUrl, title: capture.pageTitle },
    frameId: capture.frameId || 0,
  })

  return response.capture
}

async function checkLatestProduct(options = {}) {
  const state = await getState()
  if (!options.manual && !state.settings.autoProductSync) {
    return { checked: false, reason: 'auto disabled' }
  }
  if (!state.settings.token) {
    if (options.manual) throw new Error('请先登录或填写本项目访问 token')
    return { checked: false, reason: 'missing api token' }
  }
  if (!Number(state.settings.shopId || 0)) {
    if (options.manual) throw new Error('请先选择商品导入店铺')
    return { checked: false, reason: 'missing shop' }
  }

  const target = await productTabTarget(options, state)
  const response = await chrome.tabs.sendMessage(
    target.tabId,
    { type: 'FETCH_LATEST_PRODUCT' },
    { frameId: target.frameId || 0 },
  )
  if (!response?.ok) {
    throw new Error(response?.error || '主动获取最新商品失败')
  }

  await saveCapture(response.capture, {
    tab: { id: target.tabId, url: response.capture.pageUrl, title: response.capture.pageTitle },
    frameId: target.frameId || 0,
  })

  const currentSkcs = extractProductSkcs(response.capture.data)
  if (currentSkcs.length === 0) {
    throw new Error('最新商品响应中没有识别到 SKC')
  }

  const snapshot = await chrome.storage.local.get(productSnapshotKey)
  const previousSkcs = snapshot[productSnapshotKey]?.skcs || []
  const previousSet = new Set(previousSkcs)
  const newSkcs = currentSkcs.filter((skc) => !previousSet.has(skc))

  await chrome.storage.local.set({
    [productSnapshotKey]: {
      skcs: currentSkcs,
      checkedAt: new Date().toISOString(),
    },
  })

  if (previousSkcs.length === 0 || newSkcs.length === 0) {
    return {
      checked: true,
      synced: false,
      baseline: previousSkcs.length === 0,
      latestCount: currentSkcs.length,
      newSkcs,
    }
  }

  const allCapture = await fetchAllCapture('product')
  const syncResult = await syncCapture('product')
  return {
    checked: true,
    synced: true,
    latestCount: currentSkcs.length,
    newSkcs,
    autoFetchTotal: allCapture.autoFetchTotal || allCapture.itemCount || 0,
    syncResult,
  }
}

async function checkLatestDelivery(options = {}) {
  const state = await getState()
  if (!options.manual && !state.settings.autoDeliverySync) {
    return { checked: false, reason: 'auto disabled' }
  }
  if (!state.settings.token) {
    if (options.manual) throw new Error('请先登录或填写本项目访问 token')
    return { checked: false, reason: 'missing api token' }
  }

  const target = await deliveryTabTarget(options, state)
  const response = await chrome.tabs.sendMessage(
    target.tabId,
    { type: 'FETCH_LATEST_DELIVERY' },
    { frameId: target.frameId || 0 },
  )
  if (!response?.ok) {
    throw new Error(response?.error || '主动获取最新发货失败')
  }

  await saveCapture(response.capture, {
    tab: { id: target.tabId, url: response.capture.pageUrl, title: response.capture.pageTitle },
    frameId: target.frameId || 0,
  })

  const currentKeys = extractDeliveryKeys(response.capture.data)
  if (currentKeys.length === 0) {
    throw new Error('最新发货响应中没有识别到发货记录')
  }

  const snapshot = await chrome.storage.local.get(deliverySnapshotKey)
  const previousKeys = snapshot[deliverySnapshotKey]?.keys || []
  const previousSet = new Set(previousKeys)
  const newKeys = currentKeys.filter((key) => !previousSet.has(key))

  await chrome.storage.local.set({
    [deliverySnapshotKey]: {
      keys: currentKeys,
      checkedAt: new Date().toISOString(),
    },
  })

  if (previousKeys.length === 0 || newKeys.length === 0) {
    return {
      checked: true,
      synced: false,
      baseline: previousKeys.length === 0,
      latestCount: currentKeys.length,
      newKeys,
    }
  }

  const allCapture = await fetchAllCapture('delivery')
  const syncResult = await syncCapture('delivery')
  return {
    checked: true,
    synced: true,
    latestCount: currentKeys.length,
    newKeys,
    autoFetchTotal: allCapture.autoFetchTotal || allCapture.itemCount || 0,
    syncResult,
  }
}

async function productTabTarget(options, state) {
  if (options.tabId) {
    return { tabId: options.tabId, frameId: options.frameId || 0 }
  }

  const capture = state.latestCaptures?.product
  if (capture?.tabId) {
    return { tabId: capture.tabId, frameId: capture.frameId || 0 }
  }

  const tabs = await chrome.tabs.query({ url: 'https://agentseller.temu.com/*' })
  const tab = tabs.find((item) => item.id)
  if (!tab?.id) {
    throw new Error('请先打开 https://agentseller.temu.com/ 页面')
  }
  return { tabId: tab.id, frameId: 0 }
}

async function deliveryTabTarget(options, state) {
  if (options.tabId) {
    return { tabId: options.tabId, frameId: options.frameId || 0 }
  }

  const capture = state.latestCaptures?.delivery
  if (capture?.tabId) {
    return { tabId: capture.tabId, frameId: capture.frameId || 0 }
  }

  const tabs = await chrome.tabs.query({ url: 'https://seller.kuajingmaihuo.com/*' })
  const tab = tabs.find((item) => item.id)
  if (!tab?.id) {
    throw new Error('请先打开 https://seller.kuajingmaihuo.com/ 页面')
  }
  return { tabId: tab.id, frameId: 0 }
}

async function injectActiveTab() {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true })
  if (!tab?.id) throw new Error('没有可注入的当前标签页')
  await chrome.scripting.executeScript({
    target: { tabId: tab.id, allFrames: true },
    files: ['content-script.js'],
  })
  return true
}

async function apiRequest(endpoint, options = {}) {
  const settings = (await getState()).settings
  const apiBase = normalizeApiBase(defaultSettings.apiBase)
  const headers = {
    'Content-Type': 'application/json',
    ...(options.headers || {}),
  }
  if (!options.skipAuth && settings.token) {
    headers.Authorization = `Bearer ${settings.token}`
  }

  const response = await fetch(`${apiBase}${endpoint}`, {
    method: options.method || 'GET',
    headers,
    body: options.body,
  })

  const text = await response.text()
  let payload = null
  if (text) {
    try {
      payload = JSON.parse(text)
    } catch {
      payload = text
    }
  }

  if (!response.ok) {
    const message = payload?.error || payload?.message || `请求失败：HTTP ${response.status} (${apiBase}${endpoint})`
    throw new Error(message)
  }

  return payload?.data ?? payload
}

function normalizeApiBase(value) {
  const apiBase = String(value || defaultSettings.apiBase).trim()
  return apiBase.replace(/\/+$/, '')
}

function compactTimestamp(value) {
  const date = value ? new Date(value) : new Date()
  const safeDate = Number.isNaN(date.getTime()) ? new Date() : date
  return safeDate.toISOString().replace(/[-:]/g, '').replace(/\.\d{3}Z$/, 'Z')
}

function countLikelyItems(data, kind) {
  const matcher = kind === 'product' ? isProductItem : isDeliveryItem
  const matches = []
  collectMatches(data, matcher, matches, 0)
  return matches.length
}

function collectMatches(value, predicate, matches, depth) {
  if (!value || depth > 8 || matches.length > 9999) return
  if (Array.isArray(value)) {
    value.forEach((item) => collectMatches(item, predicate, matches, depth + 1))
    return
  }
  if (typeof value !== 'object') return
  if (predicate(value)) {
    matches.push(value)
    return
  }
  Object.values(value).forEach((child) => collectMatches(child, predicate, matches, depth + 1))
}

function isProductItem(value) {
  return (
    Object.prototype.hasOwnProperty.call(value, 'productSkcId') &&
    (Object.prototype.hasOwnProperty.call(value, 'productSkuSummaries') ||
      Object.prototype.hasOwnProperty.call(value, 'productSkuId') ||
      Object.prototype.hasOwnProperty.call(value, 'mainImageUrl'))
  )
}

function isDeliveryItem(value) {
  return (
    Object.prototype.hasOwnProperty.call(value, 'deliveryOrderSn') &&
    (Object.prototype.hasOwnProperty.call(value, 'subPurchaseOrderBasicVO') ||
      Object.prototype.hasOwnProperty.call(value, 'packageDetailList') ||
      Object.prototype.hasOwnProperty.call(value, 'deliveryOrderList'))
  )
}

function extractProductSkcs(data) {
  const items = getPagedItems(data)
  const ids = []
  const seen = new Set()
  for (const item of items) {
    const skc = String(item?.productSkcId || '').trim()
    if (!skc || seen.has(skc)) continue
    seen.add(skc)
    ids.push(skc)
  }
  return ids
}

function extractDeliveryKeys(data) {
  const items = flattenDeliveryItems(getPagedItems(data))
  const keys = []
  const seen = new Set()
  for (const item of items) {
    const key = [item?.expressBatchSn, item?.deliveryOrderSn].filter(Boolean).join('|')
    if (!key || seen.has(key)) continue
    seen.add(key)
    keys.push(key)
  }
  return keys
}

function flattenDeliveryItems(items) {
  const flattened = []
  for (const item of items || []) {
    if (Array.isArray(item?.deliveryOrderList) && item.deliveryOrderList.length > 0) {
      for (const child of item.deliveryOrderList) {
        flattened.push({
          ...child,
          expressBatchSn: child.expressBatchSn || item.expressBatchSn,
          deliveryOrderSn: child.deliveryOrderSn || item.deliveryOrderSn,
        })
      }
      continue
    }
    flattened.push(item)
  }
  return flattened
}

function getPagedItems(data) {
  const candidates = [
    data?.result?.pageItems,
    data?.result?.list,
    data?.result?.data,
    data?.list,
    data?.data,
    Array.isArray(data) ? data : null,
  ]
  return candidates.find((items) => Array.isArray(items)) || []
}

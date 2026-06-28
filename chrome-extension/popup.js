const loginView = document.querySelector('#loginView')
const appView = document.querySelector('#appView')
const lockedView = document.querySelector('#lockedView')
const usernameInput = document.querySelector('#username')
const passwordInput = document.querySelector('#password')
const loginButton = document.querySelector('#loginButton')
const logoutButton = document.querySelector('#logoutButton')
const apiBaseHint = document.querySelector('#apiBaseHint')
const openProductBackendButton = document.querySelector('#openProductBackendButton')
const openDeliveryBackendButton = document.querySelector('#openDeliveryBackendButton')
const fetchShopsButton = document.querySelector('#fetchShopsButton')
const syncSellerShopButton = document.querySelector('#syncSellerShopButton')
const injectButton = document.querySelector('#injectButton')
const shopSelect = document.querySelector('#shopSelect')
const autoProductSyncInput = document.querySelector('#autoProductSync')
const autoDeliverySyncInput = document.querySelector('#autoDeliverySync')
const checkLatestProductButton = document.querySelector('#checkLatestProductButton')
const checkLatestDeliveryButton = document.querySelector('#checkLatestDeliveryButton')
const capturesEl = document.querySelector('#captures')
const statusEl = document.querySelector('#status')

let currentState = null
let shops = []

document.addEventListener('DOMContentLoaded', init)
loginButton.addEventListener('click', login)
logoutButton.addEventListener('click', logout)
openProductBackendButton.addEventListener('click', () => openBackend('https://agentseller.temu.com/'))
openDeliveryBackendButton.addEventListener('click', () => openBackend('https://seller.kuajingmaihuo.com/'))
fetchShopsButton.addEventListener('click', loadShops)
syncSellerShopButton.addEventListener('click', () => syncSellerShop({ manual: true }))
injectButton.addEventListener('click', injectActiveTab)
checkLatestProductButton.addEventListener('click', checkLatestProduct)
checkLatestDeliveryButton.addEventListener('click', checkLatestDelivery)
shopSelect.addEventListener('change', saveSelectedShop)
autoProductSyncInput.addEventListener('change', saveSettings)
autoDeliverySyncInput.addEventListener('change', saveSettings)

async function init() {
  await loadState()
  if (currentState?.sellerContext?.allowed && currentState?.settings?.token) {
    await loadShops({ silent: true })
  }
}

async function loadState() {
  const response = await sendMessage({ type: 'GET_STATE' })
  currentState = response
  autoProductSyncInput.checked = Boolean(response.settings.autoProductSync)
  autoDeliverySyncInput.checked = Boolean(response.settings.autoDeliverySync)
  apiBaseHint.textContent = apiHostText(response.settings.apiBase)
  renderAuthState()
  renderCaptures(response.latestCaptures || {})
}

async function login() {
  await withBusy(loginButton, async () => {
    const response = await sendMessage({
      type: 'LOGIN',
      credentials: {
        username: usernameInput.value,
        password: passwordInput.value,
      },
    })
    currentState.settings = response.settings
    passwordInput.value = ''
    renderAuthState()
    setStatus('登录成功，已连接同步服务。', 'success')
    await syncSellerShop({ manual: false })
    await loadShops({ silent: true })
  })
}

async function saveSettings() {
  const response = await sendMessage({
    type: 'SAVE_SETTINGS',
    settings: {
      shopId: currentState?.settings?.shopId || '',
      shopName: currentState?.settings?.shopName || '',
      autoProductSync: autoProductSyncInput.checked,
      autoDeliverySync: autoDeliverySyncInput.checked,
    },
  })
  currentState.settings = response.settings
  setStatus('设置已保存。', 'success')
}

async function logout() {
  await withBusy(logoutButton, async () => {
    const response = await sendMessage({ type: 'LOGOUT' })
    currentState.settings = response.settings
    usernameInput.value = ''
    passwordInput.value = ''
    renderAuthState()
    setStatus('已退出登录。', 'success')
  })
}

async function loadShops(options = {}) {
  await saveSettings()
  await withBusy(fetchShopsButton, async () => {
    const response = await sendMessage({ type: 'FETCH_SHOPS' })
    shops = response.shops || []
    renderShops()
    if (!options.silent) {
      setStatus(`已加载 ${shops.length} 个店铺。`, 'success')
    }
  })
}

async function syncSellerShop(options = {}) {
  const run = async () => {
    const response = await sendMessage({ type: 'SYNC_SELLER_SHOP' })
    await loadState()
    await loadShops({ silent: true })
    const shops = response.result?.shops || []
    if (shops.length > 0) {
      setStatus(`已同步店铺：${shops.map((shop) => shop.shopName).join('、')}。`, 'success')
    }
  }

  if (options.manual) {
    await withBusy(syncSellerShopButton, run)
    return
  }

  try {
    await run()
  } catch (error) {
    setStatus('已登录。打开 agentseller.temu.com 后可同步当前卖家店铺。')
  }
}

async function saveSelectedShop() {
  const selected = shops.find((shop) => String(shop.id) === shopSelect.value)
  const response = await sendMessage({
    type: 'SAVE_SETTINGS',
    settings: {
      shopId: selected ? String(selected.id) : '',
      shopName: selected?.shopName || '',
      autoProductSync: autoProductSyncInput.checked,
      autoDeliverySync: autoDeliverySyncInput.checked,
    },
  })
  currentState.settings = response.settings
  setStatus('店铺已保存。', 'success')
}

function renderAuthState() {
  const allowed = Boolean(currentState?.sellerContext?.allowed)
  const loggedIn = Boolean(currentState?.settings?.token)
  lockedView.hidden = allowed
  loginView.hidden = !allowed || loggedIn
  appView.hidden = !allowed || !loggedIn
  if (!allowed) {
    const host = currentState?.sellerContext?.host
    setStatus(host ? `当前页面 ${host} 不是卖家后台。` : '当前页面不是卖家后台。')
  }
}

async function checkLatestProduct() {
  await saveSettings()
  await withBusy(checkLatestProductButton, async () => {
    const response = await sendMessage({ type: 'CHECK_LATEST_PRODUCT' })
    await loadState()
    const result = response.result
    if (result.synced) {
      setStatus(`发现 ${result.newSkcs.length} 个新 SKC，已抓全并同步。`, 'success')
    } else if (result.baseline) {
      setStatus(`已记录当前最新商品基线：${result.latestCount} 个 SKC。`, 'success')
    } else {
      setStatus('没有发现新增商品。', 'success')
    }
  })
}

async function checkLatestDelivery() {
  await saveSettings()
  await withBusy(checkLatestDeliveryButton, async () => {
    const response = await sendMessage({ type: 'CHECK_LATEST_DELIVERY' })
    await loadState()
    const result = response.result
    if (result.synced) {
      setStatus(`发现 ${result.newKeys.length} 条新发货记录，已抓全并同步。`, 'success')
    } else if (result.baseline) {
      setStatus(`已记录当前最新发货基线：${result.latestCount} 条。`, 'success')
    } else {
      setStatus('没有发现新增发货记录。', 'success')
    }
  })
}

async function injectActiveTab() {
  await withBusy(injectButton, async () => {
    await sendMessage({ type: 'INJECT_ACTIVE_TAB' })
    setStatus('已向当前页面注入捕获器，请刷新或操作卖家中心列表。', 'success')
  })
}

function renderShops() {
  const selectedShopId = String(currentState?.settings?.shopId || '')
  shopSelect.innerHTML = '<option value="">请选择商品导入店铺</option>'
  for (const shop of shops) {
    const option = document.createElement('option')
    option.value = String(shop.id)
    option.textContent = `${shop.shopName} (${shop.externalCode || '无编号'})`
    option.selected = String(shop.id) === selectedShopId
    shopSelect.appendChild(option)
  }
}

function renderCaptures(latestCaptures) {
  const entries = [
    ['product', '商品数据'],
    ['delivery', '发货数据'],
  ]
  capturesEl.innerHTML = ''

  for (const [kind, label] of entries) {
    const capture = latestCaptures[kind]
    const card = document.createElement('article')
    card.className = `capture-card${capture ? '' : ' empty'}`

    const head = document.createElement('div')
    head.className = 'capture-head'

    const title = document.createElement('strong')
    title.textContent = label
    head.appendChild(title)

    const pill = document.createElement('span')
    pill.className = 'capture-pill'
    pill.textContent = capture ? `${capture.itemCount || 0} 条` : '未捕获'
    head.appendChild(pill)
    card.appendChild(head)

    const meta = document.createElement('p')
    meta.className = 'capture-meta'
    if (capture) {
      const summary = requestSummary(capture.requestBody)
      meta.textContent = `${formatTime(capture.capturedAt)}${summary ? ` · ${summary}` : ''} · ${sourceText(capture)}`
    } else {
      meta.textContent = kind === 'product' ? '打开商品后台后可主动检查或捕获列表。' : '打开发货后台后可主动检查或捕获列表。'
    }
    card.appendChild(meta)

    const actions = document.createElement('div')
    actions.className = 'capture-actions'

    const syncButton = document.createElement('button')
    syncButton.type = 'button'
    syncButton.className = 'secondary'
    syncButton.textContent = '同步缓存'
    syncButton.disabled = !capture
    syncButton.addEventListener('click', () => syncCapture(kind, syncButton))
    actions.appendChild(syncButton)

    const fetchAllButton = document.createElement('button')
    fetchAllButton.type = 'button'
    fetchAllButton.className = 'secondary'
    fetchAllButton.textContent = '抓取全部'
    fetchAllButton.disabled = !capture
    fetchAllButton.addEventListener('click', () => fetchAllCapture(kind, fetchAllButton, false))
    actions.appendChild(fetchAllButton)

    const fetchAllAndSyncButton = document.createElement('button')
    fetchAllAndSyncButton.type = 'button'
    fetchAllAndSyncButton.className = 'primary-action'
    fetchAllAndSyncButton.textContent = '抓全同步'
    fetchAllAndSyncButton.disabled = !capture
    fetchAllAndSyncButton.addEventListener('click', () => fetchAllCapture(kind, fetchAllAndSyncButton, true))
    actions.appendChild(fetchAllAndSyncButton)

    const clearButton = document.createElement('button')
    clearButton.type = 'button'
    clearButton.className = 'secondary'
    clearButton.textContent = '清除'
    clearButton.disabled = !capture
    clearButton.addEventListener('click', () => clearCapture(kind, clearButton))
    actions.appendChild(clearButton)

    card.appendChild(actions)
    capturesEl.appendChild(card)
  }
}

function sourceText(capture) {
  const rawUrl = capture.requestUrl || capture.pageUrl || ''
  if (!rawUrl) return '来源未知'
  try {
    const url = new URL(rawUrl)
    return url.hostname
  } catch {
    return rawUrl
  }
}

function apiHostText(apiBase) {
  try {
    const url = new URL(apiBase)
    return `同步到 ${url.host}`
  } catch {
    return '同步地址由系统下载包自动配置'
  }
}

async function syncCapture(kind, button) {
  await saveSettings()
  await withBusy(button, async () => {
    const response = await sendMessage({ type: 'SYNC_CAPTURE', kind })
    await loadState()
    const total = kind === 'product' ? response.result.imported : response.result.extractedTotal
    setStatus(`同步完成：${total ?? 0} 条。`, 'success')
  })
}

async function fetchAllCapture(kind, button, shouldSync) {
  await saveSettings()
  await withBusy(button, async () => {
    const response = await sendMessage({ type: 'FETCH_ALL_CAPTURE', kind })
    await loadState()
    const total = response.capture?.autoFetchTotal || response.capture?.itemCount || 0
    if (!shouldSync) {
      setStatus(`已抓取全部：${total} 条，确认后可同步入库。`, 'success')
      return
    }

    const syncResponse = await sendMessage({ type: 'SYNC_CAPTURE', kind })
    await loadState()
    const syncedTotal = kind === 'product' ? syncResponse.result.imported : syncResponse.result.extractedTotal
    setStatus(`抓取并同步完成：${syncedTotal ?? total} 条。`, 'success')
  })
}

async function clearCapture(kind, button) {
  await withBusy(button, async () => {
    const response = await sendMessage({ type: 'CLEAR_CAPTURE', kind })
    renderCaptures(response.latestCaptures || {})
    setStatus('已清除缓存。', 'success')
  })
}

async function openBackend(url) {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true })
  if (tab?.id) {
    await chrome.tabs.update(tab.id, { url })
  } else {
    await chrome.tabs.create({ url })
  }
  window.close()
}

async function sendMessage(message) {
  const response = await chrome.runtime.sendMessage(message)
  if (!response?.ok) throw new Error(response?.error || '操作失败')
  return response
}

async function withBusy(button, fn) {
  const originalText = button.textContent
  button.disabled = true
  button.textContent = '处理中'
  setStatus('')
  try {
    await fn()
  } catch (error) {
    setStatus(error.message || String(error), 'error')
  } finally {
    button.disabled = false
    button.textContent = originalText
  }
}

function setStatus(message, type = '') {
  statusEl.textContent = message
  statusEl.className = `status ${type}`.trim()
}

function formatTime(value) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString('zh-CN', { hour12: false })
}

function requestSummary(body) {
  if (!body || body[0] !== '{') return ''
  try {
    const payload = JSON.parse(body)
    const parts = []
    const page = payload.pageNo || payload.page
    if (page) parts.push(`第 ${page} 页`)
    if (payload.pageSize) parts.push(`每页 ${payload.pageSize}`)
    if (payload.status !== undefined) parts.push(`状态 ${payload.status}`)
    return parts.join(' / ')
  } catch {
    return ''
  }
}

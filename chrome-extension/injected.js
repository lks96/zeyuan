(() => {
  if (window.__temuToolsPageCaptureInstalled) return
  window.__temuToolsPageCaptureInstalled = true

  const channel = 'TEMU_TOOLS_CAPTURE_RESPONSE'
  const maxDepth = 8
  const deliveryBatchEndpoint = '/bgSongbird-api/supplier/deliverGoods/management/pageQueryDeliveryBatch'
  const productSkcEndpoint = '/visage-agent-seller/product/skc/pageQuery'
  const sellerUserInfoEndpoint = '/api/seller/auth/userInfo'
  const maxAutoPages = 300
  const latestReusableHeaders = {
    product: {},
    delivery: {},
  }

  const originalFetch = window.fetch
  window.fetch = async function patchedFetch(input, init) {
    const response = await originalFetch.apply(this, arguments)
    inspectFetchResponse(input, init, response, getFetchBody(input, init))
    return response
  }

  const originalOpen = XMLHttpRequest.prototype.open
  const originalSend = XMLHttpRequest.prototype.send

  XMLHttpRequest.prototype.open = function patchedOpen(method, url) {
    this.__temuToolsRequestMeta = {
      method: String(method || 'GET').toUpperCase(),
      url: stringifyUrl(url),
      headers: {},
    }
    return originalOpen.apply(this, arguments)
  }

  const originalSetRequestHeader = XMLHttpRequest.prototype.setRequestHeader
  XMLHttpRequest.prototype.setRequestHeader = function patchedSetRequestHeader(name, value) {
    if (this.__temuToolsRequestMeta?.headers) {
      this.__temuToolsRequestMeta.headers[name] = value
    }
    return originalSetRequestHeader.apply(this, arguments)
  }

  XMLHttpRequest.prototype.send = function patchedSend(body) {
    this.addEventListener('loadend', () => {
      const responseType = this.responseType || 'text'
      if (responseType !== 'text' && responseType !== '') return
      inspectTextResponse({
        method: this.__temuToolsRequestMeta?.method || 'GET',
        url: this.__temuToolsRequestMeta?.url || this.responseURL || '',
        requestHeaders: this.__temuToolsRequestMeta?.headers || {},
        requestBody: stringifyBody(body),
        status: this.status,
        text: this.responseText,
      })
    })
    return originalSend.apply(this, arguments)
  }

  async function inspectFetchResponse(input, init, response, requestBody) {
    try {
      const clone = response.clone()
      const text = await clone.text()
      inspectTextResponse({
        method: getFetchMethod(input, init),
        url: response.url || stringifyUrl(input),
        requestHeaders: getFetchHeaders(input, init),
        requestBody,
        status: response.status,
        text,
      })
    } catch {
      // Ignore opaque, streaming, or already-consumed responses.
    }
  }

  function inspectTextResponse(meta) {
    rememberReusableHeaders(meta)

    if (!meta.text || meta.text.length < 2) return
    const trimmed = meta.text.trim()
    if (!trimmed || (trimmed[0] !== '{' && trimmed[0] !== '[')) return

    let data
    try {
      data = JSON.parse(trimmed)
    } catch {
      return
    }

    const kind = detectPayloadKind(data, meta.url)
    if (!kind) return

    window.postMessage(
      {
        source: 'temu-tools-page-capture',
        type: channel,
        payload: {
          kind,
          data,
          method: meta.method,
          requestUrl: meta.url,
          requestHeaders: meta.requestHeaders || {},
          requestBody: meta.requestBody || '',
          status: meta.status,
          pageUrl: location.href,
          pageTitle: document.title,
          capturedAt: new Date().toISOString(),
        },
      },
      '*',
    )
  }

  window.addEventListener('message', async (event) => {
    if (event.source !== window) return
    const message = event.data
    if (!message || message.source !== 'temu-tools-extension') return
    if (
      message.type !== 'TEMU_TOOLS_FETCH_ALL' &&
      message.type !== 'TEMU_TOOLS_FETCH_LATEST_PRODUCT' &&
      message.type !== 'TEMU_TOOLS_FETCH_LATEST_DELIVERY' &&
      message.type !== 'TEMU_TOOLS_FETCH_SELLER_USER_INFO'
    ) {
      return
    }

    try {
      const capture =
        message.type === 'TEMU_TOOLS_FETCH_ALL'
          ? await fetchAllPages(message.payload?.kind, message.payload?.capture)
          : message.type === 'TEMU_TOOLS_FETCH_LATEST_DELIVERY'
            ? await fetchLatestDelivery()
            : message.type === 'TEMU_TOOLS_FETCH_SELLER_USER_INFO'
              ? await fetchSellerUserInfo()
            : await fetchLatestProduct()
      window.postMessage(
        {
          source: 'temu-tools-page-capture',
          type: 'TEMU_TOOLS_COMMAND_RESULT',
          requestId: message.requestId,
          payload: { ok: true, capture },
        },
        '*',
      )
    } catch (error) {
      window.postMessage(
        {
          source: 'temu-tools-page-capture',
          type: 'TEMU_TOOLS_COMMAND_RESULT',
          requestId: message.requestId,
          payload: { ok: false, error: error.message || String(error) },
        },
        '*',
      )
    }
  })

  async function fetchLatestProduct() {
    const requestUrl = productRequestUrl()
    const requestBody = JSON.stringify({ page: 1, pageSize: 20 })
    const headers = sanitizeHeaders(latestReusableHeaders.product || {})
    const response = await originalFetch(requestUrl, {
      method: 'POST',
      headers,
      body: requestBody,
      credentials: 'include',
      cache: 'no-store',
    })

    if (!response.ok) {
      throw new Error(`商品最新数据请求失败：HTTP ${response.status}`)
    }

    const data = await response.json()
    const pageInfo = getPagedListInfo(data)
    if (!pageInfo.items.length) {
      throw new Error('商品最新数据为空，请确认当前账号有商品列表权限')
    }

    return {
      kind: 'product',
      data,
      method: 'POST',
      requestUrl,
      requestHeaders: headers,
      requestBody,
      status: response.status,
      pageUrl: location.href,
      pageTitle: document.title,
      capturedAt: new Date().toISOString(),
      itemCount: pageInfo.items.length,
      autoFetched: false,
      proactive: true,
    }
  }

  async function fetchLatestDelivery() {
    const requestUrl = deliveryRequestUrl()
    const requestBody = JSON.stringify({
      pageNo: 1,
      pageSize: 100,
      status: 1,
      productLabelCodeStyle: 0,
      onlyTaxWarehouseWaitApply: false,
      onlyCanceledExpress: false,
    })
    const headers = sanitizeHeaders(latestReusableHeaders.delivery || {})
    const response = await originalFetch(requestUrl, {
      method: 'POST',
      headers,
      body: requestBody,
      credentials: 'include',
      cache: 'no-store',
    })

    if (!response.ok) {
      throw new Error(`发货最新数据请求失败：HTTP ${response.status}`)
    }

    const data = await response.json()
    const pageInfo = getPagedListInfo(data)
    if (!pageInfo.items.length) {
      throw new Error('发货最新数据为空，请确认当前账号有发货列表权限')
    }

    return {
      kind: 'delivery',
      data,
      method: 'POST',
      requestUrl,
      requestHeaders: headers,
      requestBody,
      status: response.status,
      pageUrl: location.href,
      pageTitle: document.title,
      capturedAt: new Date().toISOString(),
      itemCount: pageInfo.items.length,
      autoFetched: false,
      proactive: true,
    }
  }

  async function fetchSellerUserInfo() {
    const requestUrl = sellerUserInfoUrl()
    const requestBody = '{}'
    const headers = sanitizeHeaders(latestReusableHeaders.product || {})
    const response = await originalFetch(requestUrl, {
      method: 'POST',
      headers,
      body: requestBody,
      credentials: 'include',
      cache: 'no-store',
    })

    if (!response.ok) {
      throw new Error(`卖家用户信息请求失败：HTTP ${response.status}`)
    }

    const data = await response.json()
    return {
      kind: 'sellerUserInfo',
      data,
      method: 'POST',
      requestUrl,
      requestHeaders: headers,
      requestBody,
      status: response.status,
      pageUrl: location.href,
      pageTitle: document.title,
      capturedAt: new Date().toISOString(),
    }
  }

  async function fetchAllPages(kind, capture) {
    if (!capture?.requestUrl || !capture.requestBody) {
      throw new Error('当前缓存没有可复用的请求信息，请先刷新或翻页触发一次列表接口')
    }

    const detectedKind = kind || detectPayloadKind(capture.data, capture.requestUrl)
    const pageField = detectedKind === 'delivery' ? 'pageNo' : 'page'
    const body = JSON.parse(capture.requestBody)
    const pageSize = Number(body.pageSize || 20)
    if (!Number.isFinite(pageSize) || pageSize <= 0) {
      throw new Error('请求体里缺少有效 pageSize')
    }

    const headers = sanitizeHeaders(capture.requestHeaders || {})
    const pages = []
    let total = 0
    let listPath = null
    let template = null

    for (let page = 1; page <= maxAutoPages; page += 1) {
      const pageBody = { ...body, [pageField]: page, pageSize }
      const response = await originalFetch(capture.requestUrl, {
        method: capture.method || 'POST',
        headers,
        body: JSON.stringify(pageBody),
        credentials: 'include',
        cache: 'no-store',
      })

      if (!response.ok) {
        throw new Error(`第 ${page} 页请求失败：HTTP ${response.status}`)
      }

      const data = await response.json()
      const pageInfo = getPagedListInfo(data)
      if (!pageInfo.items.length) break

      if (!template) {
        template = data
        listPath = pageInfo.path
      }

      pages.push(...pageInfo.items)
      total = firstPositiveNumber(pageInfo.total, total)

      const expectedPages = total > 0 ? Math.ceil(total / pageSize) : 0
      if ((expectedPages > 0 && page >= expectedPages) || (total > 0 && pages.length >= total)) break
      if (pageInfo.items.length < pageSize) break
      await sleep(160)
    }

    if (!template || !listPath) {
      throw new Error('没有抓取到任何列表数据')
    }

    const merged = structuredCloneSafe(template)
    setPathValue(merged, listPath, pages)
    setTotalValue(merged, total || pages.length)

    return {
      ...capture,
      data: merged,
      requestBody: JSON.stringify({ ...body, [pageField]: 1, pageSize }),
      capturedAt: new Date().toISOString(),
      itemCount: pages.length,
      autoFetched: true,
      autoFetchTotal: pages.length,
    }
  }

  function detectPayloadKind(data, requestUrl) {
    if (isDeliveryBatchEndpoint(requestUrl)) return 'delivery'
    if (isProductSkcEndpoint(requestUrl)) return 'product'
    if (containsMatchingObject(data, isProductItem, 0)) return 'product'
    if (containsMatchingObject(data, isDeliveryItem, 0)) return 'delivery'
    return ''
  }

  function rememberReusableHeaders(meta) {
    const headers = meta.requestHeaders || {}
    if (!Object.keys(headers).length) return
    if (isProductHost(meta.url) || isProductSkcEndpoint(meta.url)) {
      latestReusableHeaders.product = { ...latestReusableHeaders.product, ...headers }
    }
    if (isDeliveryHost(meta.url) || isDeliveryBatchEndpoint(meta.url)) {
      latestReusableHeaders.delivery = { ...latestReusableHeaders.delivery, ...headers }
    }
  }

  function isProductHost(requestUrl) {
    try {
      return new URL(requestUrl, location.href).hostname === 'agentseller.temu.com'
    } catch {
      return location.hostname === 'agentseller.temu.com'
    }
  }

  function isDeliveryHost(requestUrl) {
    try {
      return new URL(requestUrl, location.href).hostname === 'seller.kuajingmaihuo.com'
    } catch {
      return location.hostname === 'seller.kuajingmaihuo.com'
    }
  }

  function productRequestUrl() {
    if (location.hostname !== 'agentseller.temu.com') {
      throw new Error('请先打开 https://agentseller.temu.com/ 页面')
    }
    return `${location.origin}${productSkcEndpoint}`
  }

  function sellerUserInfoUrl() {
    if (location.hostname !== 'agentseller.temu.com') {
      throw new Error('请先打开 https://agentseller.temu.com/ 页面')
    }
    return `${location.origin}${sellerUserInfoEndpoint}`
  }

  function deliveryRequestUrl() {
    if (location.hostname !== 'seller.kuajingmaihuo.com') {
      throw new Error('请先打开 https://seller.kuajingmaihuo.com/ 页面')
    }
    return `${location.origin}${deliveryBatchEndpoint}`
  }

  function isDeliveryBatchEndpoint(requestUrl) {
    return matchesEndpointPath(requestUrl, deliveryBatchEndpoint)
  }

  function isProductSkcEndpoint(requestUrl) {
    return matchesEndpointPath(requestUrl, productSkcEndpoint)
  }

  function matchesEndpointPath(requestUrl, endpointPath) {
    try {
      return new URL(requestUrl, location.href).pathname === endpointPath
    } catch {
      return String(requestUrl || '').includes(endpointPath)
    }
  }

  function containsMatchingObject(value, predicate, depth) {
    if (!value || depth > maxDepth) return false
    if (Array.isArray(value)) {
      return value.some((item) => containsMatchingObject(item, predicate, depth + 1))
    }
    if (typeof value !== 'object') return false
    if (predicate(value)) return true

    return Object.values(value).some((child) => containsMatchingObject(child, predicate, depth + 1))
  }

  function isProductItem(value) {
    return (
      hasOwn(value, 'productSkcId') &&
      (hasOwn(value, 'productSkuSummaries') ||
        hasOwn(value, 'productSkuId') ||
        hasOwn(value, 'mainImageUrl')) &&
      (hasOwn(value, 'productName') || hasOwn(value, 'supplierPrice'))
    )
  }

  function isDeliveryItem(value) {
    return (
      hasOwn(value, 'deliveryOrderSn') &&
      (hasOwn(value, 'subPurchaseOrderBasicVO') ||
        hasOwn(value, 'packageDetailList') ||
        hasOwn(value, 'deliveryOrderList'))
    )
  }

  function hasOwn(value, key) {
    return Object.prototype.hasOwnProperty.call(value, key)
  }

  function getFetchMethod(input, init) {
    if (init?.method) return String(init.method).toUpperCase()
    if (input instanceof Request) return String(input.method || 'GET').toUpperCase()
    return 'GET'
  }

  function getFetchBody(input, init) {
    if (init && 'body' in init) return stringifyBody(init.body)
    if (input instanceof Request) return '[Request body]'
    return ''
  }

  function getFetchHeaders(input, init) {
    if (init?.headers) return headersToObject(init.headers)
    if (input instanceof Request) return headersToObject(input.headers)
    return {}
  }

  function headersToObject(headers) {
    if (!headers) return {}
    if (headers instanceof Headers) return Object.fromEntries(headers.entries())
    if (Array.isArray(headers)) return Object.fromEntries(headers)
    if (typeof headers === 'object') return { ...headers }
    return {}
  }

  function sanitizeHeaders(headers) {
    const blocked = new Set([
      'accept-encoding',
      'connection',
      'content-length',
      'cookie',
      'host',
      'origin',
      'referer',
      'user-agent',
    ])
    const clean = {}
    for (const [name, value] of Object.entries(headers || {})) {
      const key = String(name).toLowerCase()
      if (!key || blocked.has(key) || key.startsWith('sec-') || key === 'priority') continue
      clean[name] = value
    }
    if (!Object.keys(clean).some((key) => key.toLowerCase() === 'content-type')) {
      clean['content-type'] = 'application/json'
    }
    return clean
  }

  function getPagedListInfo(data) {
    const candidates = [
      { path: ['result', 'list'], items: data?.result?.list, total: data?.result?.total },
      { path: ['result', 'pageItems'], items: data?.result?.pageItems, total: data?.result?.total },
      { path: ['result', 'data'], items: data?.result?.data, total: data?.result?.total },
      { path: ['list'], items: data?.list, total: data?.total },
      { path: ['data'], items: data?.data, total: data?.total },
      { path: [], items: Array.isArray(data) ? data : null, total: Array.isArray(data) ? data.length : 0 },
    ]
    const found = candidates.find((candidate) => Array.isArray(candidate.items))
    return {
      path: found?.path || [],
      items: found?.items || [],
      total: Number(found?.total || data?.total || 0),
    }
  }

  function setPathValue(target, path, value) {
    if (path.length === 0) return value
    let current = target
    for (let index = 0; index < path.length - 1; index += 1) {
      current = current[path[index]]
    }
    current[path[path.length - 1]] = value
    return target
  }

  function setTotalValue(target, total) {
    if (target?.result && typeof target.result === 'object') target.result.total = total
    if (target && typeof target === 'object' && !Array.isArray(target)) target.total = total
  }

  function firstPositiveNumber(...values) {
    for (const value of values) {
      const number = Number(value)
      if (Number.isFinite(number) && number > 0) return number
    }
    return 0
  }

  function structuredCloneSafe(value) {
    if (typeof structuredClone === 'function') return structuredClone(value)
    return JSON.parse(JSON.stringify(value))
  }

  function sleep(ms) {
    return new Promise((resolve) => setTimeout(resolve, ms))
  }

  function stringifyBody(body) {
    if (!body) return ''
    if (typeof body === 'string') return body
    if (body instanceof URLSearchParams) return body.toString()
    if (body instanceof FormData) return '[FormData]'
    if (body instanceof Blob) return `[Blob ${body.type || 'application/octet-stream'}]`
    if (body instanceof ArrayBuffer) return `[ArrayBuffer ${body.byteLength}]`
    if (ArrayBuffer.isView(body)) return `[ArrayBufferView ${body.byteLength}]`
    try {
      return JSON.stringify(body)
    } catch {
      return String(body)
    }
  }

  function stringifyUrl(value) {
    if (typeof value === 'string') return value
    if (value instanceof URL) return value.toString()
    if (value instanceof Request) return value.url
    return ''
  }
})()

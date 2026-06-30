(() => {
  const injectedFlag = 'temuToolsCaptureBridgeInstalled'
  if (window[injectedFlag]) return
  window[injectedFlag] = true

  const script = document.createElement('script')
  script.src = chrome.runtime.getURL('injected.js')
  script.async = false
  script.onload = () => script.remove()
  ;(document.documentElement || document.head || document.body).appendChild(script)

  chrome.runtime.sendMessage({
    type: 'PAGE_READY',
    pageUrl: location.href,
  })

  window.addEventListener('message', (event) => {
    if (event.source !== window) return
    const message = event.data
    if (!message || message.source !== 'temu-tools-page-capture') return
    if (message.type === 'TEMU_TOOLS_CAPTURE_RESPONSE') {
      chrome.runtime.sendMessage({
        type: 'CAPTURE_RESPONSE',
        capture: message.payload,
      })
    } else if (message.type === 'TEMU_TOOLS_CAPTURE_DIAGNOSTIC') {
      chrome.runtime.sendMessage({
        type: 'CAPTURE_DIAGNOSTIC',
        diagnostic: message.payload,
      })
    }
  })

  chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
    if (
      message?.type !== 'FETCH_ALL_CAPTURE' &&
      message?.type !== 'FETCH_LATEST_PRODUCT' &&
      message?.type !== 'FETCH_LATEST_DELIVERY' &&
      message?.type !== 'FETCH_SALES_OVERALL' &&
      message?.type !== 'FETCH_SELLER_USER_INFO'
    ) {
      return false
    }

    const requestId = `temu-tools-${Date.now()}-${Math.random().toString(16).slice(2)}`
    const onResult = (event) => {
      if (event.source !== window) return
      const result = event.data
      if (!result || result.source !== 'temu-tools-page-capture') return
      if (result.type !== 'TEMU_TOOLS_COMMAND_RESULT' || result.requestId !== requestId) return
      window.removeEventListener('message', onResult)
      sendResponse(result.payload)
    }

    window.addEventListener('message', onResult)
    window.postMessage(
      {
        source: 'temu-tools-extension',
        type:
          message.type === 'FETCH_ALL_CAPTURE'
            ? 'TEMU_TOOLS_FETCH_ALL'
            : message.type === 'FETCH_SALES_OVERALL'
              ? 'TEMU_TOOLS_FETCH_SALES_OVERALL'
            : message.type === 'FETCH_LATEST_DELIVERY'
              ? 'TEMU_TOOLS_FETCH_LATEST_DELIVERY'
              : message.type === 'FETCH_SELLER_USER_INFO'
                ? 'TEMU_TOOLS_FETCH_SELLER_USER_INFO'
                : 'TEMU_TOOLS_FETCH_LATEST_PRODUCT',
        requestId,
        payload: {
          kind: message.kind,
          capture: message.capture,
          salesSkcs: message.salesSkcs,
        },
      },
      '*',
    )

    return true
  })
})()

import { useEffect, useRef, useCallback } from 'react'
import { getToken } from '../api/client'

type EventHandler = (data: any) => void

const listeners = new Map<string, Set<EventHandler>>()

let eventSource: EventSource | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null

function connect() {
  const token = getToken()
  if (!token) return

  // Close existing connection
  if (eventSource) {
    eventSource.close()
  }

  const url = `${window.location.protocol}//${window.location.host}/api/v1/events?token=${token}`
  eventSource = new EventSource(url)

  eventSource.onopen = () => {
    console.log('[sse] connected')
  }

  eventSource.onmessage = (e) => {
    try {
      const msg = JSON.parse(e.data)
      const type = msg.type || 'message'
      const handlers = listeners.get(type)
      if (handlers) {
        handlers.forEach(fn => fn(msg.data))
      }
      // Also notify generic listeners
      const allHandlers = listeners.get('*')
      if (allHandlers) {
        allHandlers.forEach(fn => fn(msg))
      }
    } catch { /* ignore parse errors */ }
  }

  eventSource.onerror = () => {
    console.warn('[sse] connection error, reconnecting in 3s')
    eventSource?.close()
    reconnectTimer = setTimeout(connect, 3000)
  }
}

function disconnect() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
  if (eventSource) {
    eventSource.close()
    eventSource = null
  }
}

function subscribe(type: string, handler: EventHandler) {
  if (!listeners.has(type)) {
    listeners.set(type, new Set())
  }
  listeners.get(type)!.add(handler)
}

function unsubscribe(type: string, handler: EventHandler) {
  const set = listeners.get(type)
  if (set) {
    set.delete(handler)
    if (set.size === 0) listeners.delete(type)
  }
}

/**
 * Hook that connects to the SSE stream and calls onMessage for matching events.
 * @param type - event type to listen for ('proxy_update', 'stats_update', '*' for all)
 * @param onMessage - callback when event is received
 */
export function useSSE(type: string, onMessage: EventHandler) {
  const savedHandler = useRef<EventHandler>(onMessage)
  savedHandler.current = onMessage

  const callback = useCallback((data: any) => {
    savedHandler.current(data)
  }, [])

  useEffect(() => {
    // Ensure connection is active
    if (!eventSource) {
      connect()
    }
    subscribe(type, callback)
    return () => {
      unsubscribe(type, callback)
      // Don't disconnect here — other components may be listening
    }
  }, [type, callback])
}

// Start connection automatically when token is available
const origSetToken = (window as any).__origSetToken
export function initSSE() {
  const token = getToken()
  if (token) {
    connect()
  }
}

// Reconnect when token changes
let prevToken = getToken()
const checkInterval = setInterval(() => {
  const t = getToken()
  if (t !== prevToken) {
    prevToken = t
    if (t) connect()
    else disconnect()
  }
}, 5000)

// Cleanup on hot reload
if (import.meta.hot) {
  import.meta.hot.dispose(() => {
    disconnect()
    clearInterval(checkInterval)
  })
}

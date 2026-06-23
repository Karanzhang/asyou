import { useState, useRef, useEffect } from 'react'

interface Message {
  role: 'user' | 'assistant'
  content: string
}

export default function AiChat({ apiBase }: { apiBase?: string }) {
  const [open, setOpen] = useState(false)
  const [messages, setMessages] = useState<Message[]>([
    { role: 'assistant', content: '👋 Hi! I can help you with tunnels, frpc, nodes, and more. Ask me anything!' }
  ])
  const [input, setInput] = useState('')
  const [busy, setBusy] = useState(false)
  const listRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (listRef.current) {
      listRef.current.scrollTop = listRef.current.scrollHeight
    }
  }, [messages])

  const handleSend = async () => {
    const q = input.trim()
    if (!q || busy) return
    setInput('')
    setMessages(m => [...m, { role: 'user', content: q }])
    setBusy(true)
    try {
      const base = apiBase || ''
      const res = await fetch(`${base}/api/v1/ai/query`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: q }),
      })
      if (!res.ok) throw new Error('HTTP ' + res.status)
      const data = await res.json()
      setMessages(m => [...m, { role: 'assistant', content: data.answer }])
    } catch (err: any) {
      setMessages(m => [...m, { role: 'assistant', content: '❌ Failed to get answer: ' + err.message }])
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      {/* Floating toggle button */}
      <button className="aichat-toggle" onClick={() => setOpen(!open)} title="AI Assistant">
        {open ? '✕' : '🤖'}
      </button>

      {/* Chat panel */}
      {open && (
        <div className="aichat-panel">
          <div className="aichat-header">
            <span>🤖 AI Assistant</span>
            <button className="aichat-clear" onClick={() => setMessages([
              { role: 'assistant', content: '👋 Hi! I can help you with tunnels, frpc, nodes, and more. Ask me anything!' }
            ])}>Clear</button>
          </div>
          <div className="aichat-messages" ref={listRef}>
            {messages.map((m, i) => (
              <div key={i} className={`aichat-msg aichat-${m.role}`}>
                <div className="aichat-bubble">{renderMessage(m.content)}</div>
              </div>
            ))}
            {busy && <div className="aichat-msg aichat-assistant"><div className="aichat-bubble aichat-thinking">Thinking...</div></div>}
          </div>
          <div className="aichat-input">
            <input
              value={input}
              onChange={e => setInput(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter') handleSend() }}
              placeholder="Ask a question..."
              disabled={busy}
            />
            <button onClick={handleSend} disabled={busy || !input.trim()}>Send</button>
          </div>
        </div>
      )}
    </>
  )
}

/** Simple markdown-like rendering for inline code and line breaks */
function renderMessage(text: string) {
  // Split by triple backticks for code blocks
  const parts = text.split(/(```[\s\S]*?```)/g)
  return parts.map((part, i) => {
    if (part.startsWith('```') && part.endsWith('```')) {
      const code = part.slice(3, -3).replace(/^[a-z]+\n/, '') // strip language hint
      return <pre key={i} className="aichat-code"><code>{code}</code></pre>
    }
    // Split by single backticks for inline code
    const inlineParts = part.split(/(`[^`]+`)/g)
    return <span key={i}>{inlineParts.map((ip, j) => {
      if (ip.startsWith('`') && ip.endsWith('`')) {
        return <code key={j} className="aichat-inline-code">{ip.slice(1, -1)}</code>
      }
      // Render line breaks
      return <span key={j}>{ip.split('\n').map((line, k) => <span key={k}>{k > 0 ? <br /> : null}{line}</span>)}</span>
    })}</span>
  })
}

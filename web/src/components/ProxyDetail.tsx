import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useSSE } from '../hooks/useSSE'
import { getProxy, proxyAction, getProxyStats, deleteProxy, listNodes } from '../api/client'
import type { Proxy as AsyouProxy, ProxyStats, Node as AsyouNode } from '../types'
import TrafficChart from './TrafficChart'

export default function ProxyDetail() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [proxy, setProxy] = useState<AsyouProxy | null>(null)
  const [nodes, setNodes] = useState<AsyouNode[]>([])
  const [frpsClientId, setFrpsClientId] = useState('')
  const [toast, setToast] = useState<{ msg: string; type: string } | null>(null)

  const showToast = (msg: string, type = 'success') => {
    setToast({ msg, type })
    setTimeout(() => setToast(null), 3000)
  }

  useEffect(() => {
    if (!id) return
    Promise.all([
      getProxy(parseInt(id)),
      listNodes(),
    ])
      .then(([res, n]) => {
        setProxy(res.proxy)
        setFrpsClientId(res.frps_client_id || '')
        setNodes(n)
      })
      .catch(() => showToast('Failed to load', 'error'))
  }, [id])

  const nodeName = proxy?.node_id
    ? nodes.find(n => n.id === proxy.node_id)?.name || `Node #${proxy.node_id}`
    : '—'

  const [copied, setCopied] = useState(false)
  const FRP_VERSION = '0.69.1'

  // Auto-refresh on SSE proxy update for this proxy
  useSSE('proxy_update', (data: any) => {
    if (!id || !data) return
    if (data.id === parseInt(id)) {
      getProxy(parseInt(id)).then(res => {
        setProxy(res.proxy)
        if (res.frps_client_id) setFrpsClientId(res.frps_client_id)
      }).catch(() => {})
    }
  })

  if (!proxy) return <p>Loading…</p>

  const handleAction = async (action: 'start' | 'stop' | 'reload') => {
    try {
      await proxyAction(proxy.id, action)
      showToast(`Proxy ${action}ed`)
      const res = await getProxy(proxy.id)
      setProxy(res.proxy)
      if (res.frps_client_id) setFrpsClientId(res.frps_client_id)
    } catch (err: any) {
      showToast(err.message, 'error')
    }
  }

  const handleDelete = async () => {
    if (!confirm('Delete this tunnel?')) return
    try {
      await deleteProxy(proxy.id)
      navigate('/')
    } catch (err: any) {
      showToast(err.message, 'error')
    }
  }

  let annotation: { error?: string; when?: string } | null = null
  if (proxy.annotations) {
    try { annotation = JSON.parse(proxy.annotations) } catch {}
  }

  // Build frpc config
  const frpsHost = proxy.node_id
    ? nodes.find(n => n.id === proxy.node_id)?.host || window.location.hostname
    : window.location.hostname
  const frpsPort = proxy.node_id
    ? nodes.find(n => n.id === proxy.node_id)?.bind_port || 7000
    : 7000
  const sectionName = proxy.name
  const frpcINI = `[common]
server_addr = ${frpsHost}
server_port = ${frpsPort}

[${sectionName}]
type = ${proxy.type}
local_ip = 127.0.0.1
local_port = ${proxy.local_port}
${proxy.remote_port ? `remote_port = ${proxy.remote_port}` : ''}
${proxy.subdomain ? `subdomain = ${proxy.subdomain}` : ''}`
  const frpcCommand = `frpc -c ${sectionName}.ini`

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(frpcINI)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {}
  }

  const handleDownloadINI = () => {
    downloadFile(frpcINI, `${sectionName}.ini`, 'text/plain;charset=utf-8')
  }

  const handleDownloadScript = () => {
    const ua = navigator.userAgent
    const isWin = ua.includes('Windows')
    const isMac = ua.includes('Mac OS') || ua.includes('Darwin')
    const arch = ua.includes('x86_64') || ua.includes('Win64') || ua.includes('amd64') ? 'amd64' : '386'
    const osArch = isWin ? 'windows' : isMac ? 'darwin' : 'linux'
    const ext = isWin ? 'zip' : 'tar.gz'
    const pkg = `frp_${FRP_VERSION}_${osArch}_${arch}.${ext}`
    const url = `https://github.com/fatedier/frp/releases/download/v${FRP_VERSION}/${pkg}`

    if (isWin) {
      // Embed INI content as Base64 to avoid batch/ps special char issues
      const iniBase64 = btoa(unescape(encodeURIComponent(frpcINI)))
      const accessLine = proxy.remote_port
        ? `echo  Access: http://${frpsHost}:${proxy.remote_port}`
        : `echo  Remote port: (assigned by frps, check dashboard)`
      const script = `@echo off
setlocal

REM ===== Step 1: Generate config file =====
set CONFIG_FILE=%~dp0${sectionName}.ini
powershell -Command "$b=[System.Convert]::FromBase64String('${iniBase64}');[System.Text.Encoding]::ASCII.GetString($b)|Out-File -FilePath '%CONFIG_FILE%' -Encoding ascii -Force"
if not exist "%CONFIG_FILE%" (
    echo [FAIL] Failed to create config file.
    pause
    exit /b 1
)
echo [OK] Config written to %CONFIG_FILE%

REM ===== Step 2: Find or download frpc =====
set FRPC_PATH=frpc.exe
if exist "%~dp0frpc.exe" (
    set FRPC_PATH=%~dp0frpc.exe
    echo [OK] frpc.exe found locally
) else (
    where frpc.exe >nul 2>&1
    if not errorlevel 1 (
        echo [OK] frpc.exe found in PATH
    ) else (
        echo [..] frpc not found, downloading...
        powershell -Command "$wc = New-Object System.Net.WebClient; [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; try { $wc.DownloadFile('${url}', '%TEMP%\\${pkg}') } catch { Write-Error $_; exit 1 }"
        if not exist "%TEMP%\\${pkg}" (
            echo [FAIL] Download failed. Check your network connection.
            pause
            exit /b 1
        )
        echo [OK] Downloaded, extracting...
        powershell -Command "Expand-Archive -Path '%TEMP%\\${pkg}' -DestinationPath '%TEMP%' -Force"
        if exist "%TEMP%\\frp_${FRP_VERSION}_windows_${arch}\\frpc.exe" (
            copy /y "%TEMP%\\frp_${FRP_VERSION}_windows_${arch}\\frpc.exe" "%~dp0frpc.exe" >nul
            set FRPC_PATH=%~dp0frpc.exe
            echo [OK] frpc.exe saved to %~dp0frpc.exe
        ) else (
            echo [FAIL] Extraction failed.
            pause
            exit /b 1
        )
    )
)

echo.
echo ============================================
echo  Tunnel: ${sectionName}
echo  Local:  127.0.0.1:${proxy.local_port}
${accessLine}
echo ============================================
echo.
echo Starting frpc...
"%FRPC_PATH%" -c "%CONFIG_FILE%"
if errorlevel 1 (
    echo.
    echo [FAIL] frpc exited with error. Make sure ${sectionName}.ini is in the same folder as this script.
    pause
)

echo.
echo [OK] Tunnel is running. Press Ctrl+C to stop.
pause`
      downloadFile(script, `run-${sectionName}.bat`, 'text/plain;charset=utf-8')
    } else {
      const iniBase64 = btoa(unescape(encodeURIComponent(frpcINI)))
      const unixAccessLine = proxy.remote_port
        ? `echo "Access: http://${frpsHost}:${proxy.remote_port}"`
        : `echo "Remote port: (assigned by frps, check dashboard)"`
      const script = `#!/bin/sh
set -e

DIR="$(cd "$(dirname "$0")" && pwd)"
CONFIG_FILE="$DIR/${sectionName}.ini"

# ===== Step 1: Generate config file =====
printf '%s' "${iniBase64}" | base64 -d > "$CONFIG_FILE" 2>/dev/null || printf '%s' "${iniBase64}" | python3 -c "import sys,base64; sys.stdout.buffer.write(base64.b64decode(sys.stdin.read()))" > "$CONFIG_FILE"
echo "[OK] Config written to $CONFIG_FILE"

# ===== Step 2: Find or download frpc =====
FRPC_PATH="$DIR/frpc"

if command -v frpc >/dev/null 2>&1; then
    FRPC_PATH="frpc"
    echo "[✓] frpc found in PATH"
elif [ -f "$FRPC_PATH" ]; then
    echo "[✓] frpc found locally"
else
    echo "[~] frpc not found, downloading..."
    cd /tmp
    curl -sL "${url}" -o "${pkg}" 2>/dev/null || wget -q "${url}" -O "${pkg}" 2>/dev/null || {
        echo "[FAIL] Download failed. Please install frpc manually."
        exit 1
    }
    tar xzf "${pkg}"
    cp "frp_${FRP_VERSION}_${osArch}_${arch}/frpc" "$DIR/frpc"
    chmod +x "$DIR/frpc"
    rm -rf "frp_${FRP_VERSION}_${osArch}_${arch}" "${pkg}"
    FRPC_PATH="$DIR/frpc"
    echo "[✓] frpc downloaded"
fi

echo ""
echo "============================================"
echo " Tunnel: ${sectionName}"
echo " Local:  127.0.0.1:${proxy.local_port}"
${unixAccessLine}
echo "============================================"
echo ""
echo "Starting frpc..."
exec "$FRPC_PATH" -c "$CONFIG_FILE"`
      downloadFile(script, `run-${sectionName}.sh`, 'text/plain;charset=utf-8')
    }
  }

  function downloadFile(content: string, filename: string, mime: string) {
    const blob = new Blob([content], { type: mime })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }

  return (
    <div>
      {toast && <div className={`toast ${toast.type}`}>{toast.msg}</div>}
      <div className="page-header">
        <h1>{proxy.name}</h1>
        <div>
          <button className="btn btn-outline" onClick={() => navigate('/')}>← Back</button>
        </div>
      </div>

      <div className="card">
        <div className="detail-grid">
          <div className="detail-item"><div className="label">Status</div><div className="val"><span className={`badge badge-${proxy.status === 'running' ? 'running' : 'stopped'}`}>{proxy.status}</span></div></div>
          <div className="detail-item"><div className="label">Type</div><div className="val">{proxy.type}</div></div>
          <div className="detail-item"><div className="label">Local</div><div className="val">{proxy.local_ip}:{proxy.local_port}</div></div>
          <div className="detail-item"><div className="label">Remote Port</div><div className="val">{proxy.remote_port ?? '—'}</div></div>
          <div className="detail-item"><div className="label">Node</div><div className="val">{nodeName}</div></div>
          <div className="detail-item"><div className="label">Subdomain</div><div className="val">{proxy.subdomain ?? '—'}</div></div>
          {frpsClientId && <div className="detail-item"><div className="label">frps Client ID</div><div className="val" style={{ fontFamily: 'monospace', fontSize: '0.8rem' }}>{frpsClientId}</div></div>}
        </div>
      </div>

      {annotation && annotation.error && (
        <div className="card" style={{ borderColor: 'var(--danger)' }}>
          <h3>Last Error</h3>
          <p style={{ color: 'var(--danger)', fontSize: '0.9rem' }}>{annotation.error}</p>
          {annotation.when && <p style={{ color: 'var(--text-muted)', fontSize: '0.8rem', marginTop: '0.3rem' }}>when: {annotation.when}</p>}
        </div>
      )}

      <div className="card">
        <h3>Actions</h3>
        {proxy.status === 'stopped' ? (
          <button className="btn btn-success" onClick={() => handleAction('start')}>Start</button>
        ) : (
          <button className="btn btn-warning" onClick={() => handleAction('stop')}>Stop</button>
        )}
        <button className="btn btn-outline" onClick={() => handleAction('reload')}>Reload</button>
        <button className="btn btn-danger" onClick={handleDelete}>Delete</button>
      </div>

      <TrafficChart proxyId={proxy.id} />

      {/* Local frpc config */}
      <div className="card">
        <h3>Local frpc Setup</h3>
        <p style={{ fontSize: '0.85rem', color: 'var(--text-muted)', marginBottom: '0.8rem' }}>
          Run frpc on your local machine to connect this tunnel to the frps server.
        </p>

        <div style={{ marginBottom: '0.5rem' }}>
          <span style={{ fontSize: '0.85rem', fontWeight: 600 }}>frpc command:</span>
        </div>
        <div className="code-block" style={{ marginBottom: '1rem', userSelect: 'all' }}>{frpcCommand}</div>

        <div style={{ marginBottom: '0.5rem', display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <span style={{ fontSize: '0.85rem', fontWeight: 600 }}>Config file (save as <code>{sectionName}.ini</code>):</span>
          <button className="btn btn-outline btn-sm" onClick={handleCopy}>
            {copied ? '✓ Copied' : 'Copy'}
          </button>
          <button className="btn btn-outline btn-sm" onClick={handleDownloadINI}>
            ⬇ .ini
          </button>
          <button className="btn btn-outline btn-sm" onClick={handleDownloadScript}>
            ⬇ run script
          </button>
        </div>
        <div className="code-block">{frpcINI}</div>
      </div>
    </div>
  )
}

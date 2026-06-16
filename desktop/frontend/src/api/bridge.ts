// Bridge to call Go backend methods exposed via Wails Bind.
// In the Wails runtime, Go methods are available on the `window.go.main.App` object.
// For dev/testing without Wails, we provide fallback stubs.

type GoBridge = Record<string, any>

function getBridge(): GoBridge {
  if ((window as any).go?.main?.App) {
    return (window as any).go.main.App
  }
  // Fallback for dev mode — return a proxy that logs
  return new Proxy({} as GoBridge, {
    get(_, method: string) {
      return async (...args: any[]) => {
        console.warn(`[mock] ${method} called with`, args)
        throw new Error('Wails runtime not available. Run in Wails dev mode.')
      }
    }
  })
}

const bridge = getBridge()

// --- Exported API ---

export async function login(email: string, password: string): Promise<void> {
  await bridge.Login(email, password)
}

export async function register(email: string, password: string, displayName: string): Promise<void> {
  await bridge.Register(email, password, displayName)
}

export async function isLoggedIn(): Promise<boolean> {
  return bridge.IsLoggedIn()
}

export async function logout(): Promise<void> {
  await bridge.Logout()
}

export async function getCurrentUser(): Promise<any> {
  return bridge.GetCurrentUser()
}

export async function getServerURL(): Promise<string> {
  return bridge.GetServerURL()
}

export async function setServerURL(url: string): Promise<void> {
  await bridge.SetServerURL(url)
}

export async function listNodes(): Promise<any[]> {
  return bridge.ListNodes()
}

export async function createNode(name: string, host: string, bindPort: number): Promise<void> {
  await bridge.CreateNode(name, host, bindPort)
}

export async function listProxies(): Promise<any[]> {
  return bridge.ListProxies()
}

export async function createProxy(name: string, type_: string, localPort: number, nodeID: number): Promise<string> {
  return bridge.CreateProxy(name, type_, localPort, nodeID)
}

export async function startProxy(id: number): Promise<void> {
  await bridge.StartProxy(id)
}

export async function stopProxy(id: number): Promise<void> {
  await bridge.StopProxy(id)
}

export async function discoverPorts(): Promise<any[]> {
  return bridge.DiscoverPorts()
}

export async function quickTunnel(name: string, localPort: number, nodeID: number): Promise<number> {
  return bridge.QuickTunnel(name, localPort, nodeID)
}

# asyou Web Dashboard

React + Vite + TypeScript management UI for the asyou tunnel platform.

## Tech Stack

| Tool | Version | Purpose |
|------|---------|---------|
| React | 18 | UI framework |
| Vite | 5 | Build tool & dev server |
| TypeScript | 5 | Type safety |
| React Router | 6 | Client-side routing |
| Recharts | 2 | Traffic charts |
| SSE | native | Real-time updates |

## Quick Start

```bash
# Install dependencies
npm install

# Start dev server (proxies /api to localhost:8080)
npm run dev
# → http://localhost:5173

# Production build
npm run build
# → Output: dist/
```

## Project Structure

```
web/
├── src/
│   ├── main.tsx              # Entry point
│   ├── App.tsx               # Router + layout
│   ├── index.css             # Global styles (dark theme)
│   ├── api/
│   │   └── client.ts         # REST API client (all endpoints)
│   ├── components/
│   │   ├── Layout.tsx        # Sidebar + top bar shell
│   │   ├── LoginPage.tsx     # Login / Register page
│   │   ├── ProxyList.tsx     # Home: tunnel list + create form
│   │   ├── ProxyDetail.tsx   # Tunnel detail + frpc config download
│   │   ├── NodeList.tsx      # frps node management + cluster health
│   │   ├── ApiKeys.tsx       # API key CRUD
│   │   ├── AuditLogs.tsx     # Operation history
│   │   └── TrafficChart.tsx  # Real-time traffic chart (Recharts)
│   ├── hooks/
│   │   ├── useAuth.ts        # Auth state + login/logout
│   │   └── useSSE.ts         # Server-Sent Events subscription
│   └── types/
│       └── index.ts          # TypeScript interfaces (User, Node, Proxy, etc.)
├── index.html                # HTML template
├── package.json
├── tsconfig.json
└── vite.config.ts            # Dev proxy → localhost:8080
```

## Pages

| Route | Component | Description |
|-------|-----------|-------------|
| `/login` | `LoginPage` | Login / Register |
| `/` | `ProxyList` | Tunnel list, stats bar, create form, traffic chart |
| `/proxies/:id` | `ProxyDetail` | Tunnel info, actions, error annotations, frpc config download |
| `/nodes` | `NodeList` | frps node list with health status, weight, geo info |
| `/audit-logs` | `AuditLogs` | Operation history |
| `/api-keys` | `ApiKeys` | API key management |

## API Integration

All API calls go through `src/api/client.ts` which:

- Automatically attaches `Authorization: Bearer <token>` from `localStorage`
- Handles error responses consistently
- Returns typed data via TypeScript generics

### Dev Proxy

In development mode, Vite proxies `/api/*` to `http://localhost:8080` (asyou management server). No CORS configuration needed.

## Real-Time Updates

The dashboard uses **Server-Sent Events (SSE)** via `useSSE` hook:

| Event | Trigger | Effect |
|-------|---------|--------|
| `proxy_update` | Proxy create/start/stop/delete | Auto-refresh tunnel list and detail |
| `connected` | Initial connection | Confirms user auth |

## Features

### Tunnel Management
- List tunnels with status badges and remote port
- Create tunnels with node selection dropdown
- Start / Stop / Reload / Delete operations
- Error annotations display

### Local frpc Setup
Each tunnel detail page provides:
- **frpc command** — exact CLI command to run
- **INI config preview** — with copy button
- **⬇ .ini download** — save config file
- **⬇ run script download** — auto-downloads frpc for the target OS

### Node Cluster Management
- Node list with online/offline status, weight, max connections, geo region
- Add nodes with scheduling (weight, max connections) and geo (region, country, city, lat/lng) fields
- Scheduler score display
- Stats bar: total / online / offline nodes

## Building for Production

```bash
npm run build
# Output in dist/
# Serve with Nginx, Caddy, or the asyou management server
```

The production build is served by the asyou deployment's Nginx reverse proxy (see [DEPLOY.md](../docs/DEPLOY.md)).

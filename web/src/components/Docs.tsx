export default function Docs() {
  return (
    <div className="docs">
      <h1>asyou User Guide</h1>
      <p className="docs-subtitle">Tunnel management platform built on frp</p>

      <section>
        <h2>Quick Start</h2>
        <ol>
          <li><strong>Create a tunnel</strong> — Go to <strong>Proxies</strong> and click <em>New Tunnel</em></li>
          <li><strong>Download frpc</strong> — Open the tunnel detail page, click the download link for your OS</li>
          <li><strong>Run frpc</strong> — Use the pre-generated command or download the run script</li>
          <li><strong>Access your service</strong> — Connect to <code>frps-host:remote-port</code> (TCP) or <code>https://subdomain.domain</code> (HTTP)</li>
        </ol>
      </section>

      <section>
        <h2>Proxy Types</h2>
        <table className="docs-table">
          <thead>
            <tr><th>Type</th><th>Use Case</th><th>Access</th></tr>
          </thead>
          <tbody>
            <tr>
              <td><strong>TCP</strong></td>
              <td>SSH, RDP, databases, any TCP service</td>
              <td><code>host:remote_port</code></td>
            </tr>
            <tr>
              <td><strong>HTTP</strong></td>
              <td>Web apps, APIs, development servers</td>
              <td><code>http://&lt;subdomain&gt;.&lt;host&gt;</code></td>
            </tr>
            <tr>
              <td><strong>HTTPS</strong></td>
              <td>Web apps requiring TLS termination</td>
              <td><code>https://&lt;subdomain&gt;.&lt;host&gt;</code></td>
            </tr>
            <tr>
              <td><strong>UDP</strong></td>
              <td>Game servers, DNS, streaming</td>
              <td><code>host:remote_port</code> (UDP)</td>
            </tr>
          </tbody>
        </table>
      </section>

      <section>
        <h2>Subdomain (HTTP/HTTPS only)</h2>
        <p>Subdomain lets you access your tunnel through a friendly URL instead of a port number.</p>
        <div className="docs-info">
          <strong>Prerequisites:</strong>
          <ul>
            <li>frps must have <code>subdomain_host</code> configured (e.g. <code>tunnel.example.com</code>)</li>
            <li>A DNS wildcard record <code>*.tunnel.example.com</code> pointing to your frps server</li>
            <li>frps must enable <code>vhost_http_port</code> and/or <code>vhost_https_port</code></li>
          </ul>
        </div>
        <p>Example: Create an HTTP tunnel with subdomain <code>myapp</code>, then visit <code>http://myapp.tunnel.example.com</code></p>
      </section>

      <section>
        <h2>Local Port vs Remote Port</h2>
        <ul>
          <li><strong>Local Port</strong> — The port your service runs on your machine (e.g. <code>3000</code> for a dev server)</li>
          <li><strong>Remote Port</strong> — The port on the frps server that forwards to your local service. Leave empty for auto-assignment from the node's port range</li>
        </ul>
      </section>

      <section>
        <h2>Nodes</h2>
        <p>Nodes are frps server instances. You can register multiple nodes across different regions.</p>
        <ul>
          <li>Each node has its own <strong>port range</strong> for tunnel ports</li>
          <li>When creating a tunnel without specifying a node, the scheduler auto-selects the best node</li>
          <li>Node status is checked live via the frps admin API (dashboard)</li>
          <li>You can configure <code>subdomain_host</code> per node — tunnels with the same subdomain on nodes sharing the same host will be rejected</li>
        </ul>
      </section>

      <section>
        <h2>frpc Setup (Local Client)</h2>
        <p>After creating a tunnel, open its detail page for a complete setup guide including:</p>
        <ul>
          <li><strong>frp download</strong> — Links for Windows, Linux, macOS, and ARM</li>
          <li><strong>frpc command</strong> — Ready-to-run command</li>
          <li><strong>Config preview</strong> — The generated <code>frpc.ini</code> with Copy and Download buttons</li>
          <li><strong>Run script</strong> — Automatically downloads frpc, creates config, and starts (supports Windows .bat and Linux/macOS .sh)</li>
        </ul>
      </section>

      <section>
        <h2>CLI Usage</h2>
        <div className="docs-code">
          <pre>{`# Login
asyou login --s https://your-server.com admin@example.com

# Expose a local port (TCP)
asyou expose 3000 -n my-app

# Expose with HTTP + subdomain
asyou expose 8080 --type http --subdomain myapp -n my-web

# List tunnels
asyou list

# Delete a tunnel
asyou delete 1`}</pre>
        </div>
      </section>

      <section>
        <h2>API Keys</h2>
        <p>API keys allow programmatic access without JWT login. Go to <strong>API Keys</strong> to create one. The key is shown only once — store it securely.</p>
        <p>Use it via the <code>X-Api-Key</code> header:</p>
        <div className="docs-code">
          <pre>{`curl -H "X-Api-Key: your-api-key" \\
  https://your-server.com/api/v1/proxies`}</pre>
        </div>
      </section>

      <section>
        <h2>Audit Logs</h2>
        <p>All actions (create, update, delete tunnels, manage nodes) are recorded in the audit log with timestamp, actor, IP, and action details.</p>
      </section>

      <section>
        <h2>Common Issues</h2>
        <div className="docs-issue">
          <strong>❌ Connection refused</strong>
          <p>Check that the frps server is running and the port is open in the firewall. Verify the node's host and bind_port are correct.</p>
        </div>
        <div className="docs-issue">
          <strong>❌ Subdomain not working</strong>
          <p>Ensure frps has <code>subdomain_host</code> and <code>vhost_http_port</code> configured, and DNS wildcard record exists. The tunnel type must be <code>http</code> or <code>https</code>.</p>
        </div>
        <div className="docs-issue">
          <strong>❌ Tunnel shows "stopped"</strong>
          <p>This means frpc is not running on your machine. Start it manually or use the run script from the tunnel detail page.</p>
        </div>
      </section>
    </div>
  )
}

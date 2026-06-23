export default function DocsZh({ onSwitch }: { onSwitch: () => void }) {
  return (
    <div className="docs">
      <div className="docs-header">
        <h1>asyou 使用指南</h1>
        <button className="btn btn-sm" onClick={onSwitch}>English</button>
      </div>
      <p className="docs-subtitle">基于 frp 的隧道管理平台</p>

      <section>
        <h2>快速开始</h2>
        <ol>
          <li><strong>创建隧道</strong> — 进入 <strong>Proxies</strong> 点击 <em>New Tunnel</em></li>
          <li><strong>下载 frpc</strong> — 打开隧道详情页，点击对应操作系统的下载链接</li>
          <li><strong>运行 frpc</strong> — 使用预生成的命令或下载运行脚本</li>
          <li><strong>访问你的服务</strong> — TCP 连接 <code>frps-host:remote-port</code>，HTTP 访问 <code>https://subdomain.domain</code></li>
        </ol>
      </section>

      <section>
        <h2>代理类型</h2>
        <table className="docs-table">
          <thead>
            <tr><th>类型</th><th>适用场景</th><th>访问方式</th></tr>
          </thead>
          <tbody>
            <tr>
              <td><strong>TCP</strong></td>
              <td>SSH、RDP、数据库、任意 TCP 服务</td>
              <td><code>host:remote_port</code></td>
            </tr>
            <tr>
              <td><strong>HTTP</strong></td>
              <td>Web 应用、API、开发服务器</td>
              <td><code>http://&lt;subdomain&gt;.&lt;host&gt;</code></td>
            </tr>
            <tr>
              <td><strong>HTTPS</strong></td>
              <td>需要 TLS 加密的 Web 应用</td>
              <td><code>https://&lt;subdomain&gt;.&lt;host&gt;</code></td>
            </tr>
            <tr>
              <td><strong>UDP</strong></td>
              <td>游戏服务器、DNS、流媒体</td>
              <td><code>host:remote_port</code> (UDP)</td>
            </tr>
          </tbody>
        </table>
      </section>

      <section>
        <h2>子域名（仅 HTTP/HTTPS）</h2>
        <p>子域名让你用友好的 URL 代替端口号来访问隧道。</p>
        <div className="docs-info">
          <strong>前置条件：</strong>
          <ul>
            <li>frps 必须配置了 <code>subdomain_host</code>（例如 <code>tunnel.example.com</code>）</li>
            <li>DNS 通配符记录 <code>*.tunnel.example.com</code> 指向 frps 服务器</li>
            <li>frps 必须启用 <code>vhost_http_port</code> 和/或 <code>vhost_https_port</code></li>
          </ul>
        </div>
        <p>示例：创建 HTTP 隧道并设置子域名为 <code>myapp</code>，然后访问 <code>http://myapp.tunnel.example.com</code></p>
      </section>

      <section>
        <h2>本地端口 vs 远程端口</h2>
        <ul>
          <li><strong>本地端口</strong> — 你的服务在本机运行的端口（例如开发服务器的 <code>3000</code>）</li>
          <li><strong>远程端口</strong> — frps 服务器上用于转发到你本地服务的端口。留空则从节点端口范围自动分配</li>
        </ul>
      </section>

      <section>
        <h2>节点管理</h2>
        <p>节点就是 frps 服务器实例。你可以注册多个不同区域的节点。</p>
        <ul>
          <li>每个节点有自己的 <strong>端口范围</strong> 用于隧道端口</li>
          <li>创建隧道时不指定节点时，调度器会自动选择最佳节点</li>
          <li>节点状态通过 frps 管理 API 实时检查</li>
          <li>可以为每个节点配置 <code>subdomain_host</code> — 相同 host 的节点上不允许重复子域名</li>
        </ul>
      </section>

      <section>
        <h2>frpc 本地客户端设置</h2>
        <p>创建隧道后，打开详情页可以看到完整的设置指南：</p>
        <ul>
          <li><strong>frp 下载</strong> — Windows、Linux、macOS、ARM 的下载链接</li>
          <li><strong>frpc 命令</strong> — 可直接运行的命令</li>
          <li><strong>配置预览</strong> — 生成的 <code>frpc.ini</code>，支持复制和下载</li>
          <li><strong>运行脚本</strong> — 自动下载 frpc、创建配置并启动（支持 Windows .bat 和 Linux/macOS .sh）</li>
        </ul>
      </section>

      <section>
        <h2>CLI 命令行用法</h2>
        <div className="docs-code">
          <pre>{`# 登录
asyou login --s https://your-server.com admin@example.com

# 暴露本地端口（TCP）
asyou expose 3000 -n my-app

# HTTP + 子域名
asyou expose 8080 --type http --subdomain myapp -n my-web

# 查看隧道列表
asyou list

# 删除隧道
asyou delete 1`}</pre>
        </div>
      </section>

      <section>
        <h2>API 密钥</h2>
        <p>API 密钥允许通过编程方式访问，无需 JWT 登录。进入 <strong>API Keys</strong> 创建。密钥只显示一次，请妥善保存。</p>
        <p>通过 <code>X-Api-Key</code> 请求头使用：</p>
        <div className="docs-code">
          <pre>{`curl -H "X-Api-Key: your-api-key" \\
  https://your-server.com/api/v1/proxies`}</pre>
        </div>
      </section>

      <section>
        <h2>审计日志</h2>
        <p>所有操作（创建、更新、删除隧道，管理节点）都会记录在审计日志中，包含时间、操作人、IP 和操作详情。</p>
      </section>

      <section>
        <h2>常见问题</h2>
        <div className="docs-issue">
          <strong>❌ 连接被拒绝</strong>
          <p>检查 frps 是否运行，防火墙端口是否开放。确认节点的 host 和 bind_port 配置正确。</p>
        </div>
        <div className="docs-issue">
          <strong>❌ 子域名不工作</strong>
          <p>确保 frps 配置了 <code>subdomain_host</code> 和 <code>vhost_http_port</code>，DNS 通配符记录已配置。隧道类型必须为 <code>http</code> 或 <code>https</code>。</p>
        </div>
        <div className="docs-issue">
          <strong>❌ 隧道显示"stopped"</strong>
          <p>表示 frpc 未在你的机器上运行。手动启动或使用隧道详情页的运行脚本。</p>
        </div>
      </section>
    </div>
  )
}

# asyou SDK

Multi-language SDK for the asyou tunnel management API.

## Go

```go
import "github.com/asyou/sdk-go"

client := asyou.NewClient("http://localhost:8080")
client.Login("user@example.com", "password")

proxy, _ := client.CreateProxy("my-app", "tcp", 3000, 0)
client.ProxyAction(proxy.ID, "start")
```

## Python

```python
from asyou import Client

client = Client("http://localhost:8080")
client.login("user@example.com", "password")
proxy = client.expose(3000, name="my-app")
print(f"Tunnel #{proxy.id}: {proxy.status}")
```

## Node.js

```typescript
import { Client } from 'asyou-sdk'

const client = new Client('http://localhost:8080')
await client.login('user@example.com', 'password')
const proxy = await client.expose(3000, 'my-app')
console.log(`Tunnel #${proxy.id}: ${proxy.status}`)
```

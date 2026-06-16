"""
asyou Python SDK
================
Python client for the asyou tunnel management API.

Install::

    pip install asyou  # (not yet published)

Quick start::

    from asyou import Client

    client = Client("http://localhost:8080")
    client.login("user@example.com", "password")

    # One-click expose
    proxy = client.expose(3000, name="my-app")
    print(f"Tunnel #{proxy.id} is {proxy.status}")

    # List tunnels
    for p in client.list_proxies():
        print(f"  [{p.id}] {p.name}: {p.status}")
"""

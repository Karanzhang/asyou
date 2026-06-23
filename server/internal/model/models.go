package model

import "time"

// User represents a platform user
type User struct {
    ID           int64     `db:"id" json:"id"`
    Email        string    `db:"email" json:"email"`
    PasswordHash string    `db:"password_hash" json:"-"`
    DisplayName  string    `db:"display_name" json:"display_name"`
    Role         string    `db:"role" json:"role"`
    CreatedAt    time.Time `db:"created_at" json:"created_at"`
    UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// Node represents a frps node (server)
type Node struct {
    ID              int64     `db:"id" json:"id"`
    Name            string    `db:"name" json:"name"`
    Host            string    `db:"host" json:"host"`
    ApiPort         int       `db:"api_port" json:"api_port"`
    BindPort        int       `db:"bind_port" json:"bind_port"`
    TlsEnabled      bool      `db:"tls_enabled" json:"tls_enabled"`
    AuthToken       string    `db:"auth_token" json:"auth_token,omitempty"`
    DashboardPort   int       `db:"dashboard_port" json:"dashboard_port"`
    DashboardUser   string    `db:"dashboard_user" json:"dashboard_user"`
    DashboardPwd    string    `db:"dashboard_pwd" json:"-"`
    FrpVersion      string    `db:"frp_version" json:"frp_version"`
    Region          string    `db:"region" json:"region"`
    Country         string    `db:"country" json:"country"`
    City            string    `db:"city" json:"city"`
    Latitude        float64   `db:"latitude" json:"latitude"`
    Longitude       float64   `db:"longitude" json:"longitude"`
    MaxConnections  int       `db:"max_connections" json:"max_connections"`
    Weight          float64   `db:"weight" json:"weight"`
    IsActive        bool      `db:"is_active" json:"is_active"`
    PortRangeStart  int       `db:"port_range_start" json:"port_range_start"`
    PortRangeEnd    int       `db:"port_range_end" json:"port_range_end"`
    SubdomainHost   string    `db:"subdomain_host" json:"subdomain_host"`
    Score           float64   `db:"-" json:"score,omitempty"` // computed by scheduler
    LastHeartbeat   time.Time `db:"last_heartbeat" json:"last_heartbeat"`
    CreatedAt       time.Time `db:"created_at" json:"created_at"`
    UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

// NodeHealth records health metrics for a node at a point in time.
type NodeHealth struct {
    ID                int64     `db:"id" json:"id"`
    NodeID            int64     `db:"node_id" json:"node_id"`
    LatencyMs         int       `db:"latency_ms" json:"latency_ms"`
    CurrentConnections int      `db:"current_connections" json:"current_connections"`
    CPULoad           float64   `db:"cpu_load" json:"cpu_load"`
    MemoryUsage       float64   `db:"memory_usage" json:"memory_usage"`
    BandwidthMbps     float64   `db:"bandwidth_mbps" json:"bandwidth_mbps"`
    RecordedAt        time.Time `db:"recorded_at" json:"recorded_at"`
}

// Proxy represents a user-created tunnel
type Proxy struct {
    ID                 int64     `db:"id" json:"id"`
    UserID             int64     `db:"user_id" json:"user_id"`
    NodeID             *int64    `db:"node_id" json:"node_id"`
    Name               string    `db:"name" json:"name"`
    Type               string    `db:"type" json:"type"`
    LocalIP            string    `db:"local_ip" json:"local_ip"`
    LocalPort          int       `db:"local_port" json:"local_port"`
    RemotePort         *int      `db:"remote_port" json:"remote_port"`
    Subdomain          *string   `db:"subdomain" json:"subdomain"`
    CustomDomains      *string   `db:"custom_domains" json:"custom_domains"`
    HostHeaderRewrite  *string   `db:"host_header_rewrite" json:"host_header_rewrite"`
    HttpUser           *string   `db:"http_user" json:"http_user"`
    HttpPass           *string   `db:"http_pass" json:"http_pass"`
    EnableTls          bool      `db:"enable_tls" json:"enable_tls"`
    Status             string    `db:"status" json:"status"`
    Annotations        *string   `db:"annotations" json:"annotations"`
    CreatedAt          time.Time `db:"created_at" json:"created_at"`
    UpdatedAt          time.Time `db:"updated_at" json:"updated_at"`
}

// ProxyStats holds traffic stats for a proxy
type ProxyStats struct {
    ID        int64     `db:"id" json:"id"`
    ProxyID   int64     `db:"proxy_id" json:"proxy_id"`
    Timestamp time.Time `db:"timestamp" json:"timestamp"`
    BytesIn   int64     `db:"bytes_in" json:"bytes_in"`
    BytesOut  int64     `db:"bytes_out" json:"bytes_out"`
    ConnCount int       `db:"conn_count" json:"conn_count"`
}

// AuditLog records actions
type AuditLog struct {
    ID           int64     `db:"id" json:"id"`
    ActorUserID  *int64    `db:"actor_user_id" json:"actor_user_id"`
    ActionType   string    `db:"action_type" json:"action_type"`
    ResourceType string    `db:"resource_type" json:"resource_type"`
    ResourceID   *int64    `db:"resource_id" json:"resource_id"`
    Detail       *string   `db:"detail" json:"detail"`
    IP           *string   `db:"ip" json:"ip"`
    CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// ApiKey represents persistent API tokens
type ApiKey struct {
    ID        int64     `db:"id" json:"id"`
    UserID    int64     `db:"user_id" json:"user_id"`
    Name      *string   `db:"name" json:"name"`
    TokenHash string    `db:"token_hash" json:"-"`
    Scopes    *string   `db:"scopes" json:"scopes"`
    Revoked   bool      `db:"revoked" json:"revoked"`
    CreatedAt time.Time `db:"created_at" json:"created_at"`
}

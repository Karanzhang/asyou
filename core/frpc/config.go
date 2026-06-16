package frpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// ProxyConfig holds the user-facing tunnel definition needed to build an frpc ini.
type ProxyConfig struct {
	ID            int64
	Name          string
	Type          string
	LocalIP       string
	LocalPort     int
	RemotePort    *int
	Subdomain     *string
	CustomDomains []string
}

// ServerConfig holds the frps server connection details.
type ServerConfig struct {
	Host string
	Port int
	Token string
}

// BuildINI generates the frpc .ini content from proxy and server configs.
func BuildINI(proxy *ProxyConfig, server *ServerConfig) string {
	sectionName := fmt.Sprintf("proxy_%d", proxy.ID)
	localIP := proxy.LocalIP
	if localIP == "" {
		localIP = "127.0.0.1"
	}
	cfg := map[string]map[string]interface{}{
		"common": {
			"server_addr": server.Host,
			"server_port": server.Port,
		},
		sectionName: {
			"type":      proxy.Type,
			"local_ip":  localIP,
			"local_port": proxy.LocalPort,
		},
	}
	if server.Token != "" {
		cfg["common"]["token"] = server.Token
	}
	if proxy.RemotePort != nil {
		cfg[sectionName]["remote_port"] = *proxy.RemotePort
	}
	if proxy.Subdomain != nil {
		cfg[sectionName]["subdomain"] = *proxy.Subdomain
	}
	if len(proxy.CustomDomains) > 0 {
		cfg[sectionName]["custom_domains"] = strings.Join(proxy.CustomDomains, ",")
	}

	var buf bytes.Buffer
	for name, section := range cfg {
		buf.WriteString(fmt.Sprintf("[%s]\n", name))
		for key, value := range section {
			switch v := value.(type) {
			case string:
				buf.WriteString(fmt.Sprintf("%s = %s\n", key, v))
			case int:
				buf.WriteString(fmt.Sprintf("%s = %d\n", key, v))
			default:
				buf.WriteString(fmt.Sprintf("%s = %v\n", key, v))
			}
		}
		buf.WriteString("\n")
	}
	return buf.String()
}

// MarshalProxyConfig is a helper to convert a generic proxy object (with *int, *string fields)
// into a ProxyConfig. Fields are JSON-tagged so callers can pass a struct with JSON tags.
func MarshalProxyConfig(proxyID int64, name, proxyType, localIP string, localPort int, remotePort *int, subdomain *string, customDomains *string) *ProxyConfig {
	cfg := &ProxyConfig{
		ID:        proxyID,
		Name:      name,
		Type:      proxyType,
		LocalIP:   localIP,
		LocalPort: localPort,
		RemotePort: remotePort,
		Subdomain: subdomain,
	}
	if customDomains != nil && *customDomains != "" {
		var domains []string
		if err := json.Unmarshal([]byte(*customDomains), &domains); err == nil {
			cfg.CustomDomains = domains
		} else {
			cfg.CustomDomains = strings.Split(*customDomains, ",")
		}
	}
	return cfg
}

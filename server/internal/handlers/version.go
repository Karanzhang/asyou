package handlers

import (
	"encoding/json"
	"net/http"
)

// VersionHandler returns the recommended frpc version and deployment info.
// GET /api/v1/version
func (s *Server) VersionHandler(w http.ResponseWriter, r *http.Request) {
	// Get frp version from active nodes
	var frpVersion string
	err := s.DB.QueryRow(`SELECT frp_version FROM nodes WHERE frp_version != '' AND is_active = 1 ORDER BY id LIMIT 1`).Scan(&frpVersion)
	if err != nil {
		frpVersion = "0.69.1" // default fallback
	}

	// Count nodes by version
	rows, _ := s.DB.Query(`SELECT frp_version, COUNT(*) FROM nodes WHERE frp_version != '' GROUP BY frp_version ORDER BY COUNT(*) DESC`)
	type nodeVer struct {
		Version string `json:"version"`
		Count   int    `json:"count"`
	}
	var versions []nodeVer
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var v nodeVer
			if err := rows.Scan(&v.Version, &v.Count); err == nil {
				versions = append(versions, v)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"recommended_frpc_version": frpVersion,
		"server_version":           "0.1.0",
		"nodes_by_version":         versions,
		"frpc_download_url":        "https://github.com/fatedier/frp/releases",
		"min_compatible_version":   "0.61.0",
	})
}

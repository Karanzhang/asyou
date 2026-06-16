package handlers

import (
	"math"
	"sort"
	"time"

	"github.com/asyou/server/internal/model"
)

// NodeScorer computes scores for nodes based on health, geography, and capacity.
type NodeScorer struct {
	PreferRegion string // preferred region (e.g. "us-west")
	PreferLat    float64
	PreferLng    float64
	MaxDistance  float64 // max acceptable distance in km (0 = unlimited)
}

// Score computes a score for a single node. Higher is better.
// Factors:
//   - is_active / heartbeat recency (base eligibility)
//   - Capacity ratio (connections / max_connections)
//   - Geographic distance (if preference set)
//   - Weight multiplier
func (ns *NodeScorer) Score(node model.Node, health *model.NodeHealth) float64 {
	// Base score
	score := 50.0

	// Penalty for inactive or stale heartbeat
	if !node.IsActive {
		score -= 40
	}
	if time.Since(node.LastHeartbeat) > 5*time.Minute {
		score -= 30
	}

	// Capacity score (lower utilization = higher score)
	maxCon := node.MaxConnections
	if maxCon <= 0 {
		maxCon = 100
	}
	currentCon := 0
	if health != nil {
		currentCon = health.CurrentConnections
	}
	utilization := float64(currentCon) / float64(maxCon)
	if utilization > 0.9 {
		score -= 30 // nearly full
	} else if utilization > 0.7 {
		score -= 10
	} else {
		score += 10 * (1 - utilization) // bonus for low utilization
	}

	// Latency bonus
	if health != nil && health.LatencyMs > 0 {
		if health.LatencyMs < 50 {
			score += 15
		} else if health.LatencyMs < 150 {
			score += 8
		} else if health.LatencyMs > 500 {
			score -= 15
		}
	}

	// Geographic proximity bonus
	if ns.PreferLat != 0 || ns.PreferLng != 0 {
		dist := haversineDistance(ns.PreferLat, ns.PreferLng, node.Latitude, node.Longitude)
		if ns.MaxDistance > 0 && dist > ns.MaxDistance {
			score -= 50 // too far
		} else {
			// closer = higher bonus (max 20 points)
			if dist < 100 {
				score += 20
			} else if dist < 500 {
				score += 12
			} else if dist < 2000 {
				score += 5
			}
		}
	}

	// Region preference bonus (exact match)
	if ns.PreferRegion != "" && node.Region == ns.PreferRegion {
		score += 15
	}

	// Apply weight multiplier
	if node.Weight > 0 {
		score *= node.Weight
	}

	return score
}

// SelectBest returns the highest-scoring active node for scheduling.
func (ns *NodeScorer) SelectBest(nodes []model.Node, healthMap map[int64]*model.NodeHealth) *model.Node {
	if len(nodes) == 0 {
		return nil
	}

	type scored struct {
		node  model.Node
		score float64
	}
	var scoredNodes []scored

	for _, n := range nodes {
		if !n.IsActive {
			continue
		}
		h := healthMap[n.ID]
		s := ns.Score(n, h)
		scoredNodes = append(scoredNodes, scored{node: n, score: s})
	}

	if len(scoredNodes) == 0 {
		return nil
	}

	sort.Slice(scoredNodes, func(i, j int) bool {
		return scoredNodes[i].score > scoredNodes[j].score
	})

	scoredNodes[0].node.Score = scoredNodes[0].score
	return &scoredNodes[0].node
}

// haversineDistance calculates the great-circle distance in km between two points.
func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	if lat1 == 0 && lng1 == 0 {
		return math.MaxFloat64
	}
	if lat2 == 0 && lng2 == 0 {
		return math.MaxFloat64
	}
	const R = 6371 // Earth radius in km
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// LoadLatestHealth fetches the most recent health snapshot for each node.
func (s *Server) LoadLatestHealth() map[int64]*model.NodeHealth {
	rows, err := s.DB.Query(`SELECT nh.node_id, nh.latency_ms, nh.current_connections, nh.cpu_load, nh.memory_usage, nh.bandwidth_mbps
		FROM node_health nh
		INNER JOIN (
			SELECT node_id, MAX(recorded_at) AS max_ts
			FROM node_health GROUP BY node_id
		) latest ON nh.node_id = latest.node_id AND nh.recorded_at = latest.max_ts`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	healthMap := make(map[int64]*model.NodeHealth)
	for rows.Next() {
		var h model.NodeHealth
		if err := rows.Scan(&h.NodeID, &h.LatencyMs, &h.CurrentConnections, &h.CPULoad, &h.MemoryUsage, &h.BandwidthMbps); err == nil {
			healthMap[h.NodeID] = &h
		}
	}
	return healthMap
}

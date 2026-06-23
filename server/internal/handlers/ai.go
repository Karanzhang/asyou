package handlers

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"strings"
	"unicode"
)

//go:embed user_guide.md
var userGuideContent string

type knowledgeEntry struct {
	Keywords []string
	Answer   string
}

var knowledgeBase []knowledgeEntry

func init() {
	knowledgeBase = buildKnowledgeBase(userGuideContent)
}

// zhEnMap maps English terms to Chinese translations for bilingual keyword matching.
var zhEnMap = map[string][]string{
	"tunnel":    {"隧道", "通道"},
	"proxy":     {"代理"},
	"create":    {"创建", "新建", "新增"},
	"start":     {"启动", "开始"},
	"stop":      {"停止"},
	"delete":    {"删除"},
	"node":      {"节点", "结点"},
	"server":    {"服务器", "服务端"},
	"client":    {"客户端"},
	"frpc":      {"客户端"},
	"frps":      {"服务端"},
	"subdomain": {"子域名"},
	"domain":    {"域名"},
	"port":      {"端口"},
	"local":     {"本地"},
	"remote":    {"远程"},
	"config":    {"配置"},
	"login":     {"登录"},
	"logout":    {"登出"},
	"auth":      {"认证", "鉴权"},
	"api key":   {"密钥", "apikey"},
	"audit":     {"审计", "日志"},
	"log":       {"日志"},
	"status":    {"状态", "运行"},
	"running":   {"运行中"},
	"stopped":   {"已停止"},
	"version":   {"版本"},
	"update":    {"更新", "升级"},
	"error":     {"错误", "报错"},
	"connect":   {"连接"},
	"download":  {"下载"},
	"script":    {"脚本"},
	"schedule":  {"调度"},
	"monitor":   {"监控"},
	"traffic":   {"流量"},
	"help":      {"帮助", "怎么", "如何"},
	"question":  {"问题", "疑问"},
	"doc":       {"文档"},
}

// addZhKeywords adds Chinese translations for English keywords in the set.
func addZhKeywords(kwSet map[string]bool) {
	for en, zhs := range zhEnMap {
		if kwSet[en] {
			for _, zh := range zhs {
				kwSet[zh] = true
			}
		}
	}
}

// buildKnowledgeBase parses a markdown doc and builds a keyword-searchable index.
// It splits on "## " headings — each heading + its content becomes one entry.
// The heading text (words) are used as keywords; the section body is the answer.
func buildKnowledgeBase(md string) []knowledgeEntry {
	lines := strings.Split(md, "\n")
	var entries []knowledgeEntry
	var currentHeading string
	var currentBody strings.Builder
	inSection := false

	flush := func() {
		if currentHeading == "" {
			return
		}
		body := strings.TrimSpace(currentBody.String())
		if body == "" {
			return
		}
		// Build keywords from heading: split on spaces, punctuation, numbers
		kwSet := make(map[string]bool)
		for _, raw := range tokenize(currentHeading) {
			kw := strings.ToLower(raw)
			if len(kw) > 1 {
				kwSet[kw] = true
			}
		}
		// Also add first 3 meaningful words from body as keywords
		bodyWords := tokenize(body)
		added := 0
		for _, w := range bodyWords {
			w = strings.ToLower(w)
			if len(w) > 2 && !kwSet[w] {
				kwSet[w] = true
				added++
				if added >= 3 {
					break
				}
			}
		}
		// Always include the heading as a phrase
		phrase := strings.ToLower(strings.TrimSpace(currentHeading))
		if phrase != "" {
			kwSet[phrase] = true
		}
		// Add Chinese translations for better bilingual matching
		addZhKeywords(kwSet)
		keywords := make([]string, 0, len(kwSet))
		for k := range kwSet {
			keywords = append(keywords, k)
		}
		entries = append(entries, knowledgeEntry{
			Keywords: keywords,
			Answer:   "**" + currentHeading + "**\n\n" + body,
		})
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			flush()
			currentHeading = strings.TrimPrefix(trimmed, "## ")
			currentBody.Reset()
			inSection = true
		} else if inSection {
			// Skip table of contents and separators
			if strings.HasPrefix(trimmed, "---") {
				continue
			}
			if currentBody.Len() > 0 {
				currentBody.WriteString("\n")
			}
			currentBody.WriteString(line)
		}
	}
	flush()
	return entries
}

// AiQueryRequest is the chat request body.
type AiQueryRequest struct {
	Message string `json:"message"`
}

// AiQueryResponse is the chat response body.
type AiQueryResponse struct {
	Answer string `json:"answer"`
}

// AiQueryHandler answers questions using the knowledge base built from docs.
func (s *Server) AiQueryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "method not allowed", "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
		return
	}
	var req AiQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "bad request", "BAD_REQUEST", http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		writeJSONError(w, "message required", "BAD_REQUEST", http.StatusBadRequest)
		return
	}
	answer := findBestAnswer(req.Message)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AiQueryResponse{Answer: answer})
}

// findBestAnswer matches user input against the knowledge base using keyword scoring.
func findBestAnswer(query string) string {
	query = strings.ToLower(strings.TrimSpace(query))
	words := tokenize(query)
	if len(words) == 0 {
		return fallbackAnswer()
	}

	type scored struct {
		entry knowledgeEntry
		score int
	}
	var candidates []scored
	for _, entry := range knowledgeBase {
		score := 0
		for _, keyword := range entry.Keywords {
			kw := strings.ToLower(keyword)
			if strings.Contains(query, kw) {
				score += 10
			}
			for _, w := range words {
				if strings.Contains(kw, w) || strings.Contains(w, kw) {
					score++
				}
			}
		}
		if score > 0 {
			candidates = append(candidates, scored{entry, score})
		}
	}
	if len(candidates) == 0 {
		return fallbackAnswer()
	}
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.score > best.score {
			best = c
		}
	}
	return best.entry.Answer
}

func tokenize(s string) []string {
	var words []string
	var cur strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
			cur.WriteRune(r)
		} else {
			if cur.Len() > 0 {
				words = append(words, cur.String())
				cur.Reset()
			}
		}
	}
	if cur.Len() > 0 {
		words = append(words, cur.String())
	}
	return words
}

func fallbackAnswer() string {
	return "I couldn't find a specific answer. Try asking about:\n\n" +
		"**Getting Started**\n" +
		"- Creating and managing tunnels\n" +
		"- Running frpc locally\n" +
		"- Proxy types (TCP, HTTP, HTTPS, UDP)\n\n" +
		"**Configuration**\n" +
		"- Subdomain setup\n" +
		"- Port assignment\n" +
		"- Node management\n\n" +
		"**Troubleshooting**\n" +
		"- Connection issues\n" +
		"- Status problems\n\n" +
		"Or check the **Docs** page for the full user guide."
}

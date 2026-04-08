package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

// Node represents a network node to monitor.
type Node struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Status   string `json:"status"`
	Latency  string `json:"latency"`
	LastSeen string `json:"last_seen"`
}

var (
	nodes = []struct {
		Name string
		Host string
		Port int
	}{
		{"Mother Ship", "141.136.47.94", 22},
		{"Mother Ship (Tailscale)", "100.119.206.43", 22},
		{"Beast", "148.230.100.223", 22},
		{"Beast (Tailscale)", "100.95.62.64", 22},
		{"CAT S62 Pro", "100.120.233.93", 0},
		{"Dashboard", "141.136.47.94", 3003},
		{"OpenClaw (Mother)", "141.136.47.94", 18789},
		{"OpenClaw (Beast)", "148.230.100.223", 64780},
		{"n8n", "141.136.47.94", 5678},
		{"Ollama (Beast)", "148.230.100.223", 11434},
		{"ComfyUI (Beast)", "148.230.100.223", 8188},
		{"IvonHarris API", "148.230.100.223", 3081},
		{"Reel Pipeline", "148.230.100.223", 7880},
		{"ManaMetaMaori", "148.230.100.223", 3080},
	}

	statusMu sync.RWMutex
	statuses []Node
)

func checkNode(name, host string, port int) Node {
	n := Node{
		Name: name,
		Host: host,
		Port: port,
	}

	if port == 0 {
		// Ping-only check (ICMP not available without raw sockets, just mark as monitor-only)
		n.Status = "monitor-only"
		n.Latency = "-"
		n.LastSeen = time.Now().UTC().Format(time.RFC3339)
		return n
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	elapsed := time.Since(start)

	if err != nil {
		n.Status = "down"
		n.Latency = "-"
	} else {
		conn.Close()
		n.Status = "up"
		n.Latency = elapsed.Round(time.Millisecond).String()
	}
	n.LastSeen = time.Now().UTC().Format(time.RFC3339)
	return n
}

func runChecks() {
	for {
		var results []Node
		var wg sync.WaitGroup

		resultCh := make(chan Node, len(nodes))
		for _, nd := range nodes {
			wg.Add(1)
			go func(name, host string, port int) {
				defer wg.Done()
				resultCh <- checkNode(name, host, port)
			}(nd.Name, nd.Host, nd.Port)
		}
		wg.Wait()
		close(resultCh)

		for r := range resultCh {
			results = append(results, r)
		}

		statusMu.Lock()
		statuses = results
		statusMu.Unlock()

		time.Sleep(30 * time.Second)
	}
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	statusMu.RLock()
	defer statusMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"instance":  "mana-node",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"nodes":     statuses,
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"uptime": time.Since(startTime).Round(time.Second).String(),
	})
}

var startTime = time.Now()

func main() {
	port := os.Getenv("MANA_HEALTH_PORT")
	if port == "" {
		port = "8080"
	}

	go runChecks()

	http.HandleFunc("/", handleHealth)
	http.HandleFunc("/status", handleStatus)
	http.HandleFunc("/health", handleHealth)

	log.Printf("mana-health listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

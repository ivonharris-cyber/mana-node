package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

var startTime = time.Now()

// SystemInfo holds basic system information.
type SystemInfo struct {
	Hostname  string `json:"hostname"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	CPUs      int    `json:"cpus"`
	GoVersion string `json:"go_version"`
	Uptime    string `json:"uptime"`
}

func getHostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}

func getTailscaleStatus() map[string]interface{} {
	out, err := exec.Command("tailscale", "status", "--json").Output()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		return map[string]interface{}{"error": "parse error"}
	}
	return result
}

func getTailscaleIP() string {
	out, err := exec.Command("tailscale", "ip", "-4").Output()
	if err != nil {
		return "unavailable"
	}
	return strings.TrimSpace(string(out))
}

func handleInfo(w http.ResponseWriter, r *http.Request) {
	info := SystemInfo{
		Hostname:  getHostname(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		CPUs:      runtime.NumCPU(),
		GoVersion: runtime.Version(),
		Uptime:    time.Since(startTime).Round(time.Second).String(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func handleTailscale(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ip":     getTailscaleIP(),
		"status": getTailscaleStatus(),
	})
}

func handleExec(w http.ResponseWriter, r *http.Request) {
	// Only allow GET with a ?cmd= param, limited to safe commands
	cmd := r.URL.Query().Get("cmd")
	allowed := map[string]bool{
		"uptime": true, "df": true, "free": true,
		"ps": true, "date": true, "whoami": true,
		"tailscale status": true,
	}
	if !allowed[cmd] {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "command not allowed",
			"allowed": "uptime, df, free, ps, date, whoami, tailscale status",
		})
		return
	}

	parts := strings.Fields(cmd)
	out, err := exec.Command(parts[0], parts[1:]...).CombinedOutput()
	w.Header().Set("Content-Type", "application/json")
	result := map[string]string{"command": cmd, "output": string(out)}
	if err != nil {
		result["error"] = err.Error()
	}
	json.NewEncoder(w).Encode(result)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service": "mana-agent",
		"version": "1.0.0",
		"endpoints": []string{
			"/info",
			"/tailscale",
			"/exec?cmd=<command>",
		},
		"hostname":     getHostname(),
		"tailscale_ip": getTailscaleIP(),
		"uptime":       time.Since(startTime).Round(time.Second).String(),
	})
}

func main() {
	port := os.Getenv("MANA_AGENT_PORT")
	if port == "" {
		port = "8082"
	}

	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/info", handleInfo)
	http.HandleFunc("/tailscale", handleTailscale)
	http.HandleFunc("/exec", handleExec)

	hostname := getHostname()
	tsIP := getTailscaleIP()
	log.Printf("mana-agent starting on %s (tailscale: %s) port :%s", hostname, tsIP, port)
	fmt.Printf("Mana Agent v1.0.0 — %s\n", hostname)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

package main

import (
"encoding/json"
"log"
"math/rand"
"net/http"
"os"
"path/filepath"
"strconv"
"strings"
"time"
)

type jsonMap map[string]interface{}

type statusRecorder struct {
http.ResponseWriter
status int
}

func (r *statusRecorder) WriteHeader(status int) {
r.status = status
r.ResponseWriter.WriteHeader(status)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(status)
if payload != nil {
_ = json.NewEncoder(w).Encode(payload)
}
}

func withCORS(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Access-Control-Allow-Origin", "*")
w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
if r.Method == http.MethodOptions {
w.WriteHeader(http.StatusNoContent)
return
}
next.ServeHTTP(w, r)
})
}

func withRequestLogging(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
start := time.Now()
rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
next.ServeHTTP(rec, r)
log.Printf("%s %s status=%d duration_ms=%d", r.Method, r.URL.Path, rec.status, time.Since(start).Milliseconds())
})
}

func parseJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
defer r.Body.Close()
if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
writeJSON(w, http.StatusBadRequest, jsonMap{"error": "invalid JSON: " + err.Error()})
return false
}
return true
}

func normalizeURL(base string) string {
base = strings.TrimSpace(base)
if base == "" {
return ""
}
return strings.TrimRight(base, "/")
}

func projectRoot() string {
cwd, err := os.Getwd()
if err != nil {
return "."
}
return cwd
}

func registerRoutes(mux *http.ServeMux) {
// Agents
mux.HandleFunc("/api/agents/list", handleAgentsList)
mux.HandleFunc("/api/agents/start-arb", handleAgentsStartArb)
mux.HandleFunc("/api/agents/start-liq", handleAgentsStartLiq)

// DePIN
mux.HandleFunc("/api/nodes/status", handleDepinStatus)
mux.HandleFunc("/api/nodes/register", handleDepinRegister)

// Micro‑tx
mux.HandleFunc("/api/charge", handleMicroCharge)

// Benchmarks
mux.HandleFunc("/api/run/benchmark", handleBenchmark)
mux.HandleFunc("/api/run/stress", handleStress)
}

func registerStaticRoutes(mux *http.ServeMux, root string) {
htmlPath := filepath.Join(root, "firedancer.html")
if _, err := os.Stat(htmlPath); err == nil {
mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
if r.URL.Path != "/" {
http.NotFound(w, r)
return
}
http.ServeFile(w, r, htmlPath)
})
}

staticDir := filepath.Join(root, "static")
if fi, err := os.Stat(staticDir); err == nil && fi.IsDir() {
fs := http.FileServer(http.Dir(staticDir))
mux.Handle("/static/", http.StripPrefix("/static/", fs))
}
}

// ---------- Agents ----------

func handleAgentsList(w http.ResponseWriter, r *http.Request) {
writeJSON(w, http.StatusOK, jsonMap{
"agents": []jsonMap{
{"id": "arb-1", "type": "arbitrage", "status": "running"},
{"id": "liq-1", "type": "liquidation", "status": "idle"},
},
"ts": time.Now().UTC().Format(time.RFC3339),
})
}

func handleAgentsStartArb(w http.ResponseWriter, r *http.Request) {
writeJSON(w, http.StatusOK, jsonMap{
"bot":    "arbitrage",
"status": "started",
"ts":     time.Now().UTC().Format(time.RFC3339),
})
}

func handleAgentsStartLiq(w http.ResponseWriter, r *http.Request) {
writeJSON(w, http.StatusOK, jsonMap{
"bot":    "liquidation",
"status": "started",
"ts":     time.Now().UTC().Format(time.RFC3339),
})
}

// ---------- DePIN Nodes ----------

func handleDepinStatus(w http.ResponseWriter, r *http.Request) {
writeJSON(w, http.StatusOK, jsonMap{
"nodes": []jsonMap{
{"id": "render-1", "network": "Render", "status": "online"},
{"id": "acurast-1", "network": "Acurast", "status": "online"},
{"id": "helium-1", "network": "Helium", "status": "offline"},
},
"ts": time.Now().UTC().Format(time.RFC3339),
})
}

func handleDepinRegister(w http.ResponseWriter, r *http.Request) {
writeJSON(w, http.StatusOK, jsonMap{
"status": "registered",
"id":     "node-" + strconv.FormatInt(time.Now().Unix(), 10),
"ts":     time.Now().UTC().Format(time.RFC3339),
})
}

// ---------- Micro‑transactions ----------

type microReq struct {
Wallet string  `json:"wallet"`
Amount float64 `json:"amount,string"`
}

func handleMicroCharge(w http.ResponseWriter, r *http.Request) {
var req microReq
if !parseJSONBody(w, r, &req) {
return
}
if req.Wallet == "" || req.Amount <= 0 {
writeJSON(w, http.StatusBadRequest, jsonMap{"error": "wallet and positive amount required"})
return
}
writeJSON(w, http.StatusOK, jsonMap{
"status": "ok",
"wallet": req.Wallet,
"amount": req.Amount,
"txid":   "FAKE_TX_" + strconv.FormatInt(time.Now().UnixNano(), 10),
"note":   "Wire this into real Solana tx logic.",
})
}

// ---------- Benchmark / Stress ----------

func handleBenchmark(w http.ResponseWriter, r *http.Request) {
rand.Seed(time.Now().UnixNano())
tps := 5000 + rand.Intn(15000)
lat := 80 + rand.Intn(120)
writeJSON(w, http.StatusOK, jsonMap{
"kind":       "benchmark",
"tps":        tps,
"latency_ms": lat,
"log":        "Synthetic benchmark complete. Replace with real Firedancer/validator harness.",
"ts":         time.Now().UTC().Format(time.RFC3339),
})
}

func handleStress(w http.ResponseWriter, r *http.Request) {
rand.Seed(time.Now().UnixNano())
tps := 20000 + rand.Intn(50000)
lat := 120 + rand.Intn(250)
writeJSON(w, http.StatusOK, jsonMap{
"kind":       "stress",
"tps":        tps,
"latency_ms": lat,
"log":        "Synthetic stress test complete. Replace with real load generator.",
"ts":         time.Now().UTC().Format(time.RFC3339),
})
}

func main() {
root := projectRoot()
mux := http.NewServeMux()
registerRoutes(mux)
registerStaticRoutes(mux, root)

addr := normalizeURL(":8080")
log.Println("Firedancer demo server listening on", addr)
if err := http.ListenAndServe(addr, withRequestLogging(withCORS(mux))); err != nil {
log.Fatal(err)
}
}

// main.go
package main

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

type pairing struct {
	RID       string
	ExpiresAt time.Time
	Paired    bool
}

var (
	mu      sync.Mutex
	pending = map[string]*pairing{}
	apiKey  = "<PUT_YOUR_STASH_API_KEY_HERE>" // load from config/env
	ttl     = 180 * time.Second
)

func genRID() (string, error) {
	buf := make([]byte, 16) // 128 bits
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf), nil
}

func initHandler(w http.ResponseWriter, r *http.Request) {
	rid, err := genRID()
	if err != nil {
		http.Error(w, "rid", 500)
		return
	}
	p := &pairing{RID: rid, ExpiresAt: time.Now().Add(ttl)}
	mu.Lock()
	pending[rid] = p
	mu.Unlock()
	host := r.Host // assumes served behind Stash host:port
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	pairURL := scheme + "://" + host + "/plugin/roku-pair?rid=" + rid
	resp := map[string]any{
		"request_id": rid,
		"pair_url":   pairURL,
		"expires_at": p.ExpiresAt.UTC().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(resp)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	rid := r.URL.Query().Get("rid")
	mu.Lock()
	p, ok := pending[rid]
	mu.Unlock()
	if !ok {
		json.NewEncoder(w).Encode(map[string]string{"status": "invalid"})
		return
	}
	if time.Now().After(p.ExpiresAt) {
		json.NewEncoder(w).Encode(map[string]string{"status": "expired"})
		return
	}
	if p.Paired {
		json.NewEncoder(w).Encode(map[string]string{"status": "paired", "api_key": apiKey})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "pending"})
}

func confirmHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RID string `json:"rid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	mu.Lock()
	p, ok := pending[req.RID]
	if ok && time.Now().Before(p.ExpiresAt) {
		p.Paired = true
	}
	mu.Unlock()
	if !ok {
		http.Error(w, "invalid", 404)
		return
	}
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func sweeper() {
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		now := time.Now()
		mu.Lock()
		for rid, p := range pending {
			if now.After(p.ExpiresAt) {
				delete(pending, rid)
			}
		}
		mu.Unlock()
	}
}

func main() {
	go sweeper()
	http.HandleFunc("/roku/pair/init", initHandler)
	http.HandleFunc("/roku/pair/status", statusHandler)
	http.HandleFunc("/roku/pair/confirm", confirmHandler)
	log.Println("pairing server listening on :9998")
	log.Fatal(http.ListenAndServe(":9998", nil)) // same LAN; http is fine per your constraints
}

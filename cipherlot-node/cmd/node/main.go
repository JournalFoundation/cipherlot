package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cipherlot/node/internal/cid"
)

type server struct {
	DataRoot     string
	BlobsDir     string
	ManifestsDir string
	FeedsDir     string
}

func newServer(root string) *server {
	s := &server{
		DataRoot:     root,
		BlobsDir:     filepath.Join(root, "blobs"),
		ManifestsDir: filepath.Join(root, "manifests"),
		FeedsDir:     filepath.Join(root, "feeds"),
	}
	_ = os.MkdirAll(s.BlobsDir, 0o755)
	_ = os.MkdirAll(s.ManifestsDir, 0o755)
	_ = os.MkdirAll(s.FeedsDir, 0o755)
	return s
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "time": time.Now().Unix()})
}

var (
	startTime = time.Now()
	version   = "dev" // Set via ldflags during build
)

func (s *server) status(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	
	// Get directory info
	var blobCount, manifestCount, feedCount int
	if entries, err := os.ReadDir(s.BlobsDir); err == nil {
		blobCount = len(entries)
	}
	if entries, err := os.ReadDir(s.ManifestsDir); err == nil {
		manifestCount = len(entries)
	}
	if entries, err := os.ReadDir(s.FeedsDir); err == nil {
		feedCount = len(entries)
	}
	
	status := map[string]any{
		"version":    version,
		"hostname":   hostname,
		"uptime":     time.Since(startTime).String(),
		"data_root":  s.DataRoot,
		"storage": map[string]int{
			"blobs":     blobCount,
			"manifests": manifestCount,
			"feeds":     feedCount,
		},
		"endpoints": []string{
			"/health",
			"/healthz",
			"/status",
			"/blobs/{cid}",
			"/manifests/{cid}",
			"/feeds/{author}",
		},
	}
	writeJSON(w, http.StatusOK, status)
}

// /blobs/{cid}
func (s *server) blobs(w http.ResponseWriter, r *http.Request) {
	c := strings.TrimPrefix(r.URL.Path, "/blobs/")
	if c == "" || strings.Contains(c, "/") {
		http.Error(w, "missing cid", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body error", http.StatusBadRequest)
			return
		}
		want, err := cid.ToDigestSHA256(c)
		if err != nil {
			http.Error(w, "bad cid: "+err.Error(), http.StatusBadRequest)
			return
		}
		sum := sha256.Sum256(body)
		if !equalBytes(sum[:], want) {
			http.Error(w, "digest mismatch", http.StatusBadRequest)
			return
		}
		path := filepath.Join(s.BlobsDir, c)
		if err := os.WriteFile(path, body, 0o644); err != nil {
			http.Error(w, "store failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	case http.MethodGet:
		path := filepath.Join(s.BlobsDir, c)
		f, err := os.Open(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", "application/octet-stream")
		if _, err := io.Copy(w, f); err != nil {
			log.Printf("stream blob error: %v", err)
		}
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// /manifests/{cid}
func (s *server) manifests(w http.ResponseWriter, r *http.Request) {
	c := strings.TrimPrefix(r.URL.Path, "/manifests/")
	if c == "" || strings.Contains(c, "/") {
		http.Error(w, "missing cid", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body error", http.StatusBadRequest)
			return
		}
		// validate JSON parses, but hash raw bytes for CID equality
		var tmp any
		if err := json.Unmarshal(body, &tmp); err != nil {
			http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
			return
		}
		want, err := cid.ToDigestSHA256(c)
		if err != nil {
			http.Error(w, "bad cid: "+err.Error(), http.StatusBadRequest)
			return
		}
		sum := sha256.Sum256(body)
		if !equalBytes(sum[:], want) {
			http.Error(w, "digest mismatch", http.StatusBadRequest)
			return
		}
		path := filepath.Join(s.ManifestsDir, c)
		if err := os.WriteFile(path, body, 0o644); err != nil {
			http.Error(w, "store failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	case http.MethodGet:
		path := filepath.Join(s.ManifestsDir, c)
		data, err := os.ReadFile(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

type feedEntry struct {
	CID string `json:"cid"`
	TS  int64  `json:"ts"`
}

// /feeds/{author}
func (s *server) feeds(w http.ResponseWriter, r *http.Request) {
	author := strings.TrimPrefix(r.URL.Path, "/feeds/")
	if author == "" || strings.Contains(author, "/") {
		http.Error(w, "missing author", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPost:
		var in struct {
			CID string `json:"cid"`
			TS  *int64 `json:"ts,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.CID == "" {
			http.Error(w, `body must be {"cid":"...", "ts": optional}`, http.StatusBadRequest)
			return
		}
		t := time.Now().Unix()
		if in.TS != nil {
			t = *in.TS
		}
		entry := feedEntry{CID: in.CID, TS: t}
		line, _ := json.Marshal(entry)
		path := filepath.Join(s.FeedsDir, author)
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			http.Error(w, "feed append failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer f.Close()
		if _, err := f.Write(append(line, '\n')); err != nil {
			http.Error(w, "feed write failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	case http.MethodGet:
		sinceStr := r.URL.Query().Get("since")
		var since *int64
		if sinceStr != "" {
			if v, err := strconv.ParseInt(sinceStr, 10, 64); err == nil {
				since = &v
			} else {
				http.Error(w, "invalid since", http.StatusBadRequest)
				return
			}
		}
		path := filepath.Join(s.FeedsDir, author)
		var entries []feedEntry
		if f, err := os.Open(path); err == nil {
			defer f.Close()
			sc := bufio.NewScanner(f)
			for sc.Scan() {
				var e feedEntry
				if err := json.Unmarshal(sc.Bytes(), &e); err == nil {
					if since != nil && e.TS <= *since {
						continue
					}
					entries = append(entries, e)
				}
			}
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].TS < entries[j].TS })
		writeJSON(w, http.StatusOK, entries)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func main() {
	root := os.Getenv("DATA_ROOT")
	if root == "" {
		root = os.Getenv("DATA_DIR")
	}
	if root == "" {
		root = "./data"
	}
	s := newServer(root)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/status", s.status)
	mux.HandleFunc("/blobs/", s.blobs)
	mux.HandleFunc("/manifests/", s.manifests)
	mux.HandleFunc("/feeds/", s.feeds)

	addr := ":8080"
	log.Printf("cipherlot-node listening on %s, data root %s", addr, s.DataRoot)
	if err := http.ListenAndServe(addr, logReq(mux)); err != nil {
		log.Fatal(err)
	}
}

func logReq(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

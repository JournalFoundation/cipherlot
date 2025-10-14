package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cipherlot/client/internal/cid"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "publish":
		publishCmd(os.Args[2:])
	case "subscribe":
		subscribeCmd(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `cipherlot-client

Commands:
  publish   --author <id> --node <http://host:port> <file>
  subscribe --author <id> --node <http://host:port> [--out DIR] [--since UNIX] [--interval SECONDS]

Examples:
  clt publish --author bob --node http://127.0.0.1:30999 ./myfile.bin
  clt subscribe --author bob --node http://127.0.0.1:30999 --out ./downloads
`)
	os.Exit(2)
}

func publishCmd(args []string) {
	fs := flag.NewFlagSet("publish", flag.ExitOnError)
	author := fs.String("author", "", "author id")
	node := fs.String("node", "http://127.0.0.1:30999", "node base URL")
	_ = fs.Parse(args)
	if *author == "" || fs.NArg() != 1 {
		usage()
	}
	file := fs.Arg(0)
	data, err := os.ReadFile(file)
	must(err)

	blobCID := cid.FromBytesSHA256(data)
	put(*node, "/blobs/"+blobCID, "application/octet-stream", data)

	manifest := map[string]any{
		"v":       0,
		"author":  *author,
		"created": time.Now().UTC().Format(time.RFC3339),
		"mime":    "application/octet-stream",
		"chunks":  []string{blobCID},
		"enc":     map[string]any{"algo": "none"},
		"caps":    nil,
		"sig":     nil,
	}
	manBytes, _ := json.Marshal(manifest)
	manCID := cid.FromBytesSHA256(manBytes)
	put(*node, "/manifests/"+manCID, "application/json", manBytes)

	body := map[string]any{"cid": manCID, "ts": time.Now().Unix()}
	payload, _ := json.Marshal(body)
	post(*node, "/feeds/"+*author, "application/json", payload)

	fmt.Printf("Published. Deeplink: vault://cid/%s\n", manCID)
}

func subscribeCmd(args []string) {
	fs := flag.NewFlagSet("subscribe", flag.ExitOnError)
	author := fs.String("author", "", "author id")
	node := fs.String("node", "http://127.0.0.1:30999", "node base URL")
	out := fs.String("out", ".", "output directory")
	since := fs.Int64("since", 0, "only entries newer than this UNIX ts")
	interval := fs.Int("interval", 5, "poll interval seconds")
	_ = fs.Parse(args)
	if *author == "" {
		usage()
	}
	_ = os.MkdirAll(*out, 0o755)

	var sincePtr *int64
	if *since > 0 {
		s := *since
		sincePtr = &s
	}

	for {
		url := *node + "/feeds/" + *author
		if sincePtr != nil {
			url += fmt.Sprintf("?since=%d", *sincePtr)
		}
		resp, err := http.Get(url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "feed error: %v\n", err)
			time.Sleep(time.Duration(*interval) * time.Second)
			continue
		}
		if resp.StatusCode != 200 {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			fmt.Fprintf(os.Stderr, "feed %d: %s\n", resp.StatusCode, string(b))
			time.Sleep(time.Duration(*interval) * time.Second)
			continue
		}
		var entries []struct {
			CID string `json:"cid"`
			TS  int64  `json:"ts"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&entries)
		resp.Body.Close()

		// process in order
		for _, e := range entries {
			manURL := *node + "/manifests/" + e.CID
			mr, err := http.Get(manURL)
			if err != nil || mr.StatusCode != 200 {
				if mr != nil {
					mr.Body.Close()
				}
				continue
			}
			var manifest struct {
				Created string   `json:"created"`
				Chunks  []string `json:"chunks"`
			}
			_ = json.NewDecoder(mr.Body).Decode(&manifest)
			mr.Body.Close()

			if len(manifest.Chunks) == 0 {
				continue
					}
			blobCID := manifest.Chunks[0]
			br, err := http.Get(*node + "/blobs/" + blobCID)
			if err != nil || br.StatusCode != 200 {
				if br != nil {
					br.Body.Close()
				}
				continue
			}
			data, _ := io.ReadAll(br.Body)
			br.Body.Close()

			name := manifest.Created
			if name == "" {
				name = time.Unix(e.TS, 0).UTC().Format(time.RFC3339)
			}
			name = strings.ReplaceAll(name, ":", "_")
			name = strings.ReplaceAll(name, "/", "_")
			outfile := filepath.Join(*out, fmt.Sprintf("%s_%s.bin", name, blobCID[:8]))
			_ = os.WriteFile(outfile, data, 0o644)
			fmt.Printf("Downloaded %s\n", outfile)

			s := e.TS
			sincePtr = &s
		}

		time.Sleep(time.Duration(*interval) * time.Second)
	}
}

func put(node, path, contentType string, body []byte) {
	req, _ := http.NewRequest(http.MethodPut, node+path, bytes.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	resp, err := http.DefaultClient.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		b, _ := io.ReadAll(resp.Body)
		panic(fmt.Sprintf("PUT %s -> %d %s", path, resp.StatusCode, string(b)))
	}
}

func post(node, path, contentType string, body []byte) {
	resp, err := http.Post(node+path, contentType, bytes.NewReader(body))
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		b, _ := io.ReadAll(resp.Body)
		panic(fmt.Sprintf("POST %s -> %d %s", path, resp.StatusCode, string(b)))
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

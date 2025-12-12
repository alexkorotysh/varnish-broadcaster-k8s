package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

func resolveIPs(host string) ([]string, error) {
	addrs, err := net.LookupHost(host)
	if err != nil {
		return nil, err
	}
	return addrs, nil
}

// broadcast sents requests to Pod’s with retry and timeout
func broadcast(ctx context.Context, host string, port, retries int, timeout time.Duration, r *http.Request) error {
	ips, err := resolveIPs(host)
	if err != nil {
		return err
	}
	if len(ips) == 0 {
		return fmt.Errorf("no backend IPs resolved for %s", host)
	}

	body, _ := io.ReadAll(r.Body)
	r.Body.Close()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var failed []string

	for _, ip := range ips {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()

			var lastErr error
			for i := 0; i <= retries; i++ {
				url := fmt.Sprintf("http://%s:%d%s", ip, port, r.URL.Path)
				req, _ := http.NewRequestWithContext(ctx, r.Method, url, bytes.NewReader(body))
				req.Header = r.Header.Clone()

				client := http.Client{Timeout: timeout}
				resp, err := client.Do(req)
				if err != nil {
					lastErr = err
				} else {
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
					if resp.StatusCode < 400 {
						return // success
					}
					lastErr = fmt.Errorf("status %d", resp.StatusCode)
				}
			}

			mu.Lock()
			failed = append(failed, fmt.Sprintf("%s: %v", ip, lastErr))
			mu.Unlock()
		}(ip)
	}

	wg.Wait()
	if len(failed) > 0 {
		return fmt.Errorf("failed pods: %v", failed)
	}
	return nil
}

func main() {
	host := os.Getenv("BACKEND_HOST")
	if host == "" {
		log.Fatal("BACKEND_HOST is required")
	}

	port := mustEnvInt("BACKEND_PORT", 6081)
	retries := mustEnvInt("RETRIES", 2)
	timeout := mustEnvDuration("TIMEOUT", 3*time.Second)

	mux := http.NewServeMux()

	// Основной endpoint для fan-out
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err := broadcast(r.Context(), host, port, retries, timeout, r)
		if err != nil {
			log.Printf("broadcast error: %v", err)
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(200)
		w.Write([]byte("OK\n"))
	})

	// Health check endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		ips, err := resolveIPs(host)
		if err != nil || len(ips) == 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(fmt.Sprintf("unhealthy: %v\n", err)))
			return
		}

		// проверка хотя бы одного Pod
		success := false
		for _, ip := range ips {
			url := fmt.Sprintf("http://%s:%d/", ip, port)
			client := http.Client{Timeout: 1 * time.Second}
			resp, err := client.Get(url)
			if err == nil && resp.StatusCode < 500 {
				success = true
				resp.Body.Close()
				break
			}
		}
		if success {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok\n"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("unhealthy\n"))
		}
	})

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Listening on :8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func mustEnvInt(key string, def int) int {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	v, _ := strconv.Atoi(val)
	return v
}

func mustEnvDuration(key string, def time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return def
	}
	return d
}
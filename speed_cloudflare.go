package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// runCloudflareSpeedtest measures Ping/Jitter, Download (Mbps), and Upload (Mbps)
// against Cloudflare's speedtest endpoints (speed.cloudflare.com).
func runCloudflareSpeedtest(ctx context.Context, doPing, doDown, doUp bool) (pingMs, jitterMs, downMbps, upMbps float64, err error) {
	client := &http.Client{
		Timeout: 12 * time.Second,
	}

	// 1. Ping & Jitter via keep-alive warm-up + 4 RTT samples
	if doPing {
		var rtts []float64
		for i := 0; i < 5; i++ {
			if ctx.Err() != nil {
				return 0, 0, 0, 0, ctx.Err()
			}
			req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, "https://speed.cloudflare.com/cdn-cgi/trace", nil)
			if reqErr != nil {
				continue
			}
			t0 := time.Now()
			resp, respErr := client.Do(req)
			elapsed := float64(time.Since(t0).Microseconds()) / 1000.0
			if respErr == nil && resp != nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
				if i > 0 { // skip sample 0 (DNS + TCP/TLS handshake)
					rtts = append(rtts, elapsed)
				}
			}
		}
		if len(rtts) == 0 {
			return 0, 0, 0, 0, fmt.Errorf("cloudflare ping probe unreachable")
		}
		var sum float64
		for _, r := range rtts {
			sum += r
		}
		pingMs = sum / float64(len(rtts))
		if len(rtts) > 1 {
			var jitterSum float64
			for i := 1; i < len(rtts); i++ {
				jitterSum += math.Abs(rtts[i] - rtts[i-1])
			}
			jitterMs = jitterSum / float64(len(rtts)-1)
		}
	}

	// 2. Multi-threaded Download (3 workers, 10MB chunks, bounded to 8s)
	if doDown && ctx.Err() == nil {
		dlCtx, cancelDL := context.WithTimeout(ctx, 8*time.Second)
		var totalBytes int64
		var wg sync.WaitGroup
		t0 := time.Now()

		for w := 0; w < 3; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for dlCtx.Err() == nil {
					req, reqErr := http.NewRequestWithContext(dlCtx, http.MethodGet, "https://speed.cloudflare.com/__down?bytes=10000000", nil)
					if reqErr != nil {
						return
					}
					resp, respErr := client.Do(req)
					if respErr != nil || resp == nil {
						return
					}
					n, _ := io.Copy(io.Discard, resp.Body)
					_ = resp.Body.Close()
					atomic.AddInt64(&totalBytes, n)
				}
			}()
		}
		wg.Wait()
		elapsedSec := time.Since(t0).Seconds()
		cancelDL()
		if elapsedSec > 0 && totalBytes > 0 {
			downMbps = (float64(totalBytes) * 8.0) / (elapsedSec * 1_000_000.0)
		}
	}

	// 3. Multi-threaded Upload (3 workers, 2MB payload chunks, bounded to 6s)
	if doUp && ctx.Err() == nil {
		ulCtx, cancelUL := context.WithTimeout(ctx, 6*time.Second)
		payload := make([]byte, 2*1024*1024)
		var totalBytes int64
		var wg sync.WaitGroup
		t0 := time.Now()

		for w := 0; w < 3; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for ulCtx.Err() == nil {
					req, reqErr := http.NewRequestWithContext(ulCtx, http.MethodPost, "https://speed.cloudflare.com/__up", bytes.NewReader(payload))
					if reqErr != nil {
						return
					}
					req.Header.Set("Content-Type", "application/octet-stream")
					resp, respErr := client.Do(req)
					if respErr != nil || resp == nil {
						return
					}
					_, _ = io.Copy(io.Discard, resp.Body)
					_ = resp.Body.Close()
					atomic.AddInt64(&totalBytes, int64(len(payload)))
				}
			}()
		}
		wg.Wait()
		elapsedSec := time.Since(t0).Seconds()
		cancelUL()
		if elapsedSec > 0 && totalBytes > 0 {
			upMbps = (float64(totalBytes) * 8.0) / (elapsedSec * 1_000_000.0)
		}
	}

	return pingMs, jitterMs, downMbps, upMbps, nil
}

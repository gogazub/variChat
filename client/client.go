package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	addr     = flag.String("addr", "http://localhost:8080", "base URL of API (include http:// and port)")
	scenario = flag.String("scenario", "all", "scenario to run: one | idempotent | concurrency | bigpayload | merkle | all")
	conns    = flag.Int("conns", 20, "number of concurrent workers for concurrency scenario")
	reqs     = flag.Int("reqs", 100, "total requests to send in concurrency scenario")
	timeout  = flag.Duration("timeout", 10*time.Second, "request timeout per HTTP call")
)

type MessagePayload struct {
	ChatID      int64  `json:"chat_id"`
	UserID      int64  `json:"user_id"`
	Payload     string `json:"payload"`
	PayloadHash string `json:"payload_hash,omitempty"`
	BatchID     int64  `json:"batch_id,omitempty"`
}

type MerklePayload struct {
	ChatID int64  `json:"chat_id"`
	Root   string `json:"root"`
}

func main() {
	flag.Parse()

	switch *scenario {
	case "one":
		runOne()
	case "idempotent":
		runIdempotent()
	case "concurrency":
		runConcurrency(*conns, *reqs)
	case "bigpayload":
		runBigPayload()
	case "merkle":
		runMerkle()
	case "all":
		runAll(*conns, *reqs)
	default:
		fmt.Printf("unknown scenario %s\n", *scenario)
		os.Exit(2)
	}
}

func doRequest(ctx context.Context, method, url string, body []byte, headers map[string]string) (int, []byte, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if _, ok := headers["Content-Type"]; !ok {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{}
	start := time.Now()
	resp, err := client.Do(req)
	dur := time.Since(start)
	if err != nil {
		return 0, nil, dur, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, dur, err
	}
	return resp.StatusCode, b, dur, nil
}

func runOne() {
	fmt.Println("Scenario: one — single /messages request")
	msg := MessagePayload{
		ChatID:  1,
		UserID:  42,
		Payload: "Hello, this is a test message",
	}
	data, _ := json.Marshal(msg)
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	url := *addr + "/messages"
	status, body, dur, err := doRequest(ctx, "POST", url, data, nil)
	if err != nil {
		fmt.Printf("Request error: %v\n", err)
		return
	}
	fmt.Printf("Status: %d, time: %v, body: %s\n", status, dur, string(body))
}

func runIdempotent() {
	fmt.Println("Scenario: idempotent — send same Idempotency-Key twice")
	key := fmt.Sprintf("idem-%d", time.Now().UnixNano())
	msg := MessagePayload{
		ChatID:  1,
		UserID:  77,
		Payload: "Idempotent test payload",
	}
	data, _ := json.Marshal(msg)
	url := *addr + "/messages"
	for i := 0; i < 2; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		headers := map[string]string{"Idempotency-Key": key}
		status, body, dur, err := doRequest(ctx, "POST", url, data, headers)
		cancel()
		if err != nil {
			fmt.Printf("Attempt %d error: %v\n", i+1, err)
		} else {
			fmt.Printf("Attempt %d: status=%d time=%v body=%s\n", i+1, status, dur, string(body))
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func runConcurrency(workers, total int) {
	fmt.Printf("Scenario: concurrency — %d total requests with %d workers\n", total, workers)
	url := *addr + "/messages"

	var wg sync.WaitGroup
	jobs := make(chan int, total)
	results := make(chan error, total)

	// producer
	go func() {
		for i := 0; i < total; i++ {
			jobs <- i
		}
		close(jobs)
	}()

	// workers
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range jobs {
				msg := MessagePayload{
					ChatID:  int64(1 + (j % 5)),
					UserID:  int64(1000 + (j % 50)),
					Payload: fmt.Sprintf("concurrent payload #%d from worker %d", j, id),
				}
				data, _ := json.Marshal(msg)
				ctx, cancel := context.WithTimeout(context.Background(), *timeout)
				start := time.Now()
				status, _, dur, err := doRequest(ctx, "POST", url, data, nil)
				cancel()
				_ = dur
				if err != nil {
					results <- fmt.Errorf("job %d worker %d error: %w", j, id, err)
					continue
				}
				if status >= 400 {
					results <- fmt.Errorf("job %d worker %d bad status: %d", j, id, status)
					continue
				}
				// success
				if time.Since(start) > 500*time.Millisecond {
					// optionally log slow
					fmt.Printf("slow request: job %d worker %d took %v\n", j, id, time.Since(start))
				}
				results <- nil
			}
		}(w)
	}

	// wait and close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// collect
	var errs int
	var done int
	for r := range results {
		done++
		if r != nil {
			errs++
			fmt.Println("ERROR:", r)
		}
		if done%50 == 0 {
			fmt.Printf("progress: %d/%d\n", done, total)
		}
	}
	fmt.Printf("Finished concurrency run: total=%d errors=%d\n", total, errs)
}

func runBigPayload() {
	fmt.Println("Scenario: bigpayload — send large payload (~1MB)")
	// build ~1MB body
	var b bytes.Buffer
	for i := 0; i < 100000; i++ {
		b.WriteString("x")
		if b.Len() > 1024*1024 {
			break
		}
	}
	msg := MessagePayload{
		ChatID:  10,
		UserID:  999,
		Payload: b.String(),
	}
	data, _ := json.Marshal(msg)
	url := *addr + "/messages"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	status, body, dur, err := doRequest(ctx, "POST", url, data, map[string]string{"Content-Type": "application/json"})
	if err != nil {
		fmt.Printf("Request error: %v\n", err)
		return
	}
	fmt.Printf("Status: %d, time: %v, body len: %d\n", status, dur, len(body))
}

func runMerkle() {
	fmt.Println("Scenario: merkle — call /merkle endpoint")
	pl := MerklePayload{
		ChatID: 1,
		Root:   "deadbeefcafebabedeadbeef", // example root
	}
	data, _ := json.Marshal(pl)
	url := *addr + "/merkle"
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	status, body, dur, err := doRequest(ctx, "POST", url, data, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Status: %d time: %v body: %s\n", status, dur, string(body))
}

func runAll(workers, total int) {
	fmt.Println("Running all scenarios in sequence:")
	runOne()
	time.Sleep(200 * time.Millisecond)
	runIdempotent()
	time.Sleep(200 * time.Millisecond)
	runBigPayload()
	time.Sleep(200 * time.Millisecond)
	runMerkle()
	time.Sleep(200 * time.Millisecond)
	runConcurrency(workers, total)
}

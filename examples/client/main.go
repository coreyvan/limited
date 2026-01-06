package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"

	"github.com/coreyvan/limited"
)

var retryableStatusCodes = []int{
	http.StatusTooManyRequests,
	http.StatusServiceUnavailable,
	http.StatusGatewayTimeout,
}

func main() {
	l := limited.NewLimiter(limited.Config{
		MaxRetries: 5,
	})

	//if err := l.Call(func() error {
	//	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/toomany", nil)
	//	if err != nil {
	//		return fmt.Errorf("failed to create request: %w", err)
	//	}
	//
	//	resp, err := http.DefaultClient.Do(req)
	//	if err != nil {
	//		return fmt.Errorf("request failed: %w", err)
	//	}
	//	defer resp.Body.Close()
	//
	//	if slices.Contains(retryableStatusCodes, resp.StatusCode) {
	//		return limited.WrapRetryable(fmt.Errorf("received retryable status code: %d", resp.StatusCode))
	//	}
	//	fmt.Printf("Request succeeded with status code: %s\n", resp.Status)
	//	return nil
	//}); err != nil {
	//	panic(err)
	//}

	for i := 0; i < 10; i++ {
		if err := l.Call(func() error {
			req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/bucket", nil)
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response body: %w", err)
			}
			log.Printf("Got response body: %s", string(b))

			if slices.Contains(retryableStatusCodes, resp.StatusCode) {
				return limited.WrapRetryable(fmt.Errorf("received retryable status code: %d", resp.StatusCode))
			}
			return nil
		}); err != nil {
			panic(err)
		}
	}
}

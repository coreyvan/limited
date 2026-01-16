package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"

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

	err := setupOpenTelemetry()
	if err != nil {
		log.Fatalf("Failed to setup OpenTelemetry exporter: %v", err)
	}
	log.Println("otel setup successfully")

	ticker := time.NewTicker(300 * time.Millisecond)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	errChan := make(chan error, 1)

	for {
		select {
		case <-ticker.C:
			log.Println("tick... making request to server")
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
				errChan <- err
			}
		case sig := <-sigChan:
			log.Printf("Received signal, initiating shutdown... signal: %v", sig)
		case err := <-errChan:
			log.Fatalf("Received an error: %v", err)
		}
	}
}

func setupOpenTelemetry() error {
	ctx := context.Background()
	exp, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithInsecure(), otlpmetricgrpc.WithEndpointURL("http://localhost:4317"))
	if err != nil {
		return err
	}

	reader := sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(30*time.Second))

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.DeploymentEnvironmentName("limited.development.server"),
		),
	)

	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader), sdkmetric.WithResource(res))

	otel.SetMeterProvider(mp)

	go func() {
		if err := runtime.Start(); err != nil {
			log.Fatal(err)
		}
	}()

	return nil
}

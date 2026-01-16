package main

import (
	"context"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	otelchimetric "github.com/riandyrn/otelchi/metric"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"

	"github.com/coreyvan/limited"
)

// This is a toy server that demonstrates use of server side rate limiting
//
func main() {
	sl := limited.NewBucketLimiter(limited.BucketLimiterConfig{
		MaxTokens:  5,
		RefillRate: 1,
	})
	err := sl.Start()
	if err != nil {
		log.Fatalf("Failed to start bucket limiter: %v", err)
	}
	defer func() {
		_ = sl.Stop()
	}()
	err = setupOpenTelemetry()
	if err != nil {
		log.Fatalf("Failed to setup OpenTelemetry exporter: %v", err)
	}
	log.Println("otel setup successfully")

	baseCfg := otelchimetric.NewBaseConfig("limited-server", otelchimetric.WithMeterProvider(otel.GetMeterProvider()))
	r := chi.NewRouter()
	r.Use(
		otelchimetric.NewRequestDurationMillis(baseCfg),
		otelchimetric.NewRequestInFlight(baseCfg),
		otelchimetric.NewResponseSizeBytes(baseCfg),
	)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Get("/toomany", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	r.Get("/bucket", func(w http.ResponseWriter, r *http.Request) {
		jitter := time.Duration(50+rand.IntN(200)) * time.Millisecond
		time.Sleep(jitter)
		if !sl.Allow() {
			log.Printf("Too Many Requests")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("Too many requests - bucket is empty"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Request allowed - bucket has tokens"))
	})

	s := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	errChan := make(chan error, 1)

	go func() {
		log.Println("Listening on", s.Addr)
		errChan <- s.ListenAndServe()
	}()

	select {
	case sig := <-sigChan:
		log.Printf("Received signal, initiating shutdown... signal: %v", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.Shutdown(ctx); err != nil {
			log.Printf("Error during server shutdown: %v", err)
		} else {
			log.Println("Server shutdown gracefully")
		}
	case err := <-errChan:
		log.Fatalf("Server encountered an error: %v", err)
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

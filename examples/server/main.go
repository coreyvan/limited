package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

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
	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Get("/toomany", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	r.Get("/bucket", func(w http.ResponseWriter, r *http.Request) {
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

	log.Println("Listening on", s.Addr)
	err = s.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

package util

import (
	"context"
	"net/http"
	"sync"
	"time"
)

var (
	healthCheckServerShutdownTimeout = 1 * time.Second
)

func SetupHealthCheck(ctx context.Context, logger *Logger, wg *sync.WaitGroup, address string) error {
	var srv http.Server

	wg.Add(1)
	go func() {
		defer wg.Done()

		srv = http.Server{
			Addr: address,
		}
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.PrintError("error when running the healthcheck server: %s", err)
		}

	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		<-ctx.Done()
		ctxWithTimeout, _ := context.WithTimeout(context.Background(), healthCheckServerShutdownTimeout)
		err := srv.Shutdown(ctxWithTimeout)
		if err != nil {
			logger.PrintError("failed to shutdown the health check server: %s", err)
		}

	}()

	return nil
}

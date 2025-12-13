package util

import (
	"context"
	"net"
	"net/http"
	"os"
)

func GoServeHTTP(ctx context.Context, logger *Logger, addr string, serveMux *http.ServeMux) {
	s := &http.Server{
		BaseContext: func(net.Listener) context.Context { return ctx },
		Addr:        addr,
		Handler:     serveMux,
	}
	lc := net.ListenConfig{}
	l, err := lc.Listen(ctx, "tcp", s.Addr)
	if err != nil {
		logger.PrintError("Error starting HTTP server on %s: %v\n", addr, err)
		return
	}
	go func() {
		err := s.Serve(l)
		if err != http.ErrServerClosed {
			logger.PrintError("Error running HTTP server on %s: %v\n", addr, err)
		}
	}()
	go func() {
		<-ctx.Done()
		s.Close()
	}()
}

func SetupHttpHandlerDummy(ctx context.Context, logger *Logger) {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "5000"
	}
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", HttpRedirectToApp)
	GoServeHTTP(ctx, logger, ":"+port, serveMux)
}

// HttpRedirectToApp - Provides a HTTP redirect to the pganalyze app
func HttpRedirectToApp(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://app.pganalyze.com/", http.StatusFound)
}

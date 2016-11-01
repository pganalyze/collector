package config

import (
	"net/http"
	"os"
	"strings"
)

func dummyHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://app.pganalyze.com/", http.StatusFound)
}

func handleHeroku() (servers []ServerConfig) {
	// This is required so Heroku doesn't think the dyno crashed
	go func() {
		http.HandleFunc("/", dummyHandler)
		http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	}()

	for _, kv := range os.Environ() {
		parts := strings.Split(kv, "=")
		if strings.HasSuffix(parts[0], "_URL") {
			config := getDefaultConfig()
			config.SectionName = parts[0]
			config.SystemID = strings.Replace(parts[0], "_URL", "", 1)
			config.SystemType = "heroku"
			config.DbURL = parts[1]
			servers = append(servers, *config)
		}
	}

	return
}

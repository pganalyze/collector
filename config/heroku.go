package config

import (
	"bufio"
	"net/http"
	"os"
	"strings"

	"github.com/bmizerany/lpx"
)

const bufferLen = 500

func dummyHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://app.pganalyze.com/", http.StatusFound)
}

func (s *Config) logHandler(w http.ResponseWriter, r *http.Request) {
	var specifiedConfigName string
	if strings.HasPrefix(r.URL.Path, "/logs/") {
		specifiedConfigName = strings.Replace(r.URL.Path, "/logs/", "", 1)
	}
	lp := lpx.NewReader(bufio.NewReader(r.Body))
	for lp.Next() {
		procID := string(lp.Header().Procid)
		if procID == "heroku-postgres" || strings.HasPrefix(procID, "postgres.") {
			s.HerokuLogStream <- HerokuLogStreamItem{Header: *lp.Header(), Content: lp.Bytes(), SpecifiedConfigName: specifiedConfigName}
		}
	}
}

func handleHeroku() (conf Config) {
	conf.HerokuLogStream = make(chan HerokuLogStreamItem, bufferLen)

	// This is required to receive logs, as well as so Heroku doesn't think the dyno crashed
	go func() {
		defer close(conf.HerokuLogStream)
		http.HandleFunc("/", dummyHandler)
		http.HandleFunc("/logs", conf.logHandler)
		http.HandleFunc("/logs/", conf.logHandler)
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
			conf.Servers = append(conf.Servers, *config)
		}
	}

	return
}

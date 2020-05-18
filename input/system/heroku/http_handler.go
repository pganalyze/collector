package heroku

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/bmizerany/lpx"
)

func SetupHttpHandlerDummy() {
	go func() {
		http.HandleFunc("/", dummyHandler)
		http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	}()
}

func SetupHttpHandlerLogs(herokuLogStream chan<- HerokuLogStreamItem) {
	go func() {
		http.HandleFunc("/", dummyHandler)
		http.HandleFunc("/logs/", func(w http.ResponseWriter, r *http.Request) {
			var namespace string
			if strings.HasPrefix(r.URL.Path, "/logs/") {
				namespace = strings.Replace(r.URL.Path, "/logs/", "", 1)
			} else {
				namespace = "default"
			}
			lp := lpx.NewReader(bufio.NewReader(r.Body))
			for lp.Next() {
				procID := string(lp.Header().Procid)
				if procID == "heroku-postgres" || strings.HasPrefix(procID, "postgres.") {
					select {
					case herokuLogStream <- HerokuLogStreamItem{Header: *lp.Header(), Content: lp.Bytes(), Namespace: namespace}:
						// Handed over successfully
					default:
						fmt.Printf("WARNING: Channel buffer exceeded, skipping message\n")
					}
				}
			}
		})
		http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	}()
}

func dummyHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://app.pganalyze.com/", http.StatusFound)
}

package runner

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/util"
)

func HandlePsqlExplain(logger *util.Logger, configFilename string) {
	conf, err := config.Read(logger, configFilename)
	if err != nil {
		fmt.Printf("\nERROR: Failed reading pganalyze collector configuration (use '\\pset pager off' to disable): %s\n", err)
		return
	}

	if len(conf.Servers) == 0 {
		fmt.Println("\nERROR: Missing pganalyze collector configuration - please configure at least one server to upload EXPLAIN data for (use '\\pset pager off' to disable)")
		return
	}

	fileInfo, _ := os.Stdin.Stat()
	if !(fileInfo.Mode()&os.ModeCharDevice == 0) {
		fmt.Println("\nERROR: This mode can only be used as a pager for psql, like this: \\setenv PAGER 'pganalyze-collector --psql-explain'")
		return
	}

	server := conf.Servers[0]

	scanner := bufio.NewScanner(os.Stdin)
	planOpen := false
	planLines := ""
	alignedMode := false
	for scanner.Scan() {
		s := scanner.Text()
		if strings.TrimSpace(s) == "QUERY PLAN" {
			planOpen = true
			planLines = ""
			alignedMode = false
		} else if planOpen && regexp.MustCompile(`^\(\d+ rows?\)$`).MatchString(s) {
			planOpen = false
			requestURL := server.APIBaseURL + "/v2/snapshots/upload_explain"
			data := url.Values{
				"explain":     {planLines},
				"db_name":     {server.DbName},
				"db_username": {server.DbUsername},
			}

			req, err := http.NewRequest("POST", requestURL, strings.NewReader(data.Encode()))
			if err != nil {
				fmt.Printf("\nERROR: Could not upload EXPLAIN to pganalyze (use '\\pset pager off' to disable): %s\n", err)
				return
			}

			req.Header.Set("Pganalyze-Api-Key", server.APIKey)
			req.Header.Set("Pganalyze-System-Id", server.SystemID)
			req.Header.Set("Pganalyze-System-Type", server.SystemType)
			req.Header.Set("Pganalyze-System-Scope", server.SystemScope)
			req.Header.Set("User-Agent", util.CollectorNameAndVersion)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Add("Accept", "application/json,text/plain")

			resp, err := server.HTTPClient.Do(req)
			if err != nil {
				fmt.Printf("\nERROR: Could not upload EXPLAIN to pganalyze (use '\\pset pager off' to disable): %s\n", err)
				return
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("\nERROR: Could not upload EXPLAIN to pganalyze (use '\\pset pager off' to disable): %s\n", err)
				return
			}

			if resp.StatusCode != http.StatusOK {
				fmt.Printf("\nERROR: Could not upload EXPLAIN to pganalyze (use '\\pset pager off' to disable): %s\n", body)
			}

			if len(body) > 0 {
				fmt.Printf("\n🚀 Plan uploaded: %s\n", body)
			}
		} else if planOpen {
			if regexp.MustCompile(`^\-+$`).MatchString(s) {
				alignedMode = true
			} else {
				if alignedMode && s[len(s)-1:len(s)] == "+" {
					s = strings.TrimRight(s[0:len(s)-1], " ")
				}
				planLines += s + "\n"
			}
		} else {
			fmt.Println(s)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("\nERROR: pganalyze collector could not read result set (use '\\pset pager off' to disable): %s\n", err)
	}
	return
}

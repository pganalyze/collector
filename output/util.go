package output

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"

	"github.com/pganalyze/collector/state"
)

func parseSnapshotResponse(resp *http.Response, opts state.CollectionOpts) (msg string, err error) {
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Error when submitting: %s\n", body)
	}

	if !opts.TestRun {
		// we only care about the response body in test runs, so don't bother parsing otherwise
		return "", nil
	}

	contentType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		return "", fmt.Errorf("Error decoding response: %s\n", err)
	}

	if contentType != "application/json" {
		return string(body), nil
	}

	var jsonBody struct {
		Message string `json:"message"`
	}
	err = json.Unmarshal(body, &jsonBody)
	if err != nil {
		return "", fmt.Errorf("Error decoding response: %s\n", err)
	}
	return jsonBody.Message, nil
}

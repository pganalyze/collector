package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/pganalyze/collector/jobs"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type ErrorResult struct {
	Error string `json:"error"`
}

func errorJson(errMessage string) []byte {
	var res ErrorResult
	res.Error = errMessage
	resJson, _ := json.Marshal(res)
	return resJson
}

func runJob(ctx context.Context, server *state.Server, prefixedLogger *util.Logger, globalCollectionOpts state.CollectionOpts, jobKind string, jobParameters []byte) (string, []byte) {
	switch jobKind {
	case "reindex":
		err := jobs.CheckSupportForReindexJob(ctx, server, prefixedLogger, globalCollectionOpts, jobParameters)
		if err != nil {
			return "unsupported", errorJson(err.Error())
		}
		result, err := jobs.RunReindexJob(ctx, server, prefixedLogger, globalCollectionOpts, jobParameters)
		if err != nil {
			return "failed", errorJson(err.Error())
		}
		return "succeeded", result
	default:
		return "unsupported", errorJson(fmt.Sprintf("unsupported job kind: %s", jobKind))
	}
}

type Job struct {
	Id int `json:"id"`
	Kind string `json:"kind"`
	Parameters json.RawMessage `json:"parameters"`
}

type jobsApiResponse struct {
	Jobs []Job `json:"jobs"`
}

func getJobs(server *state.Server) ([]Job, error) {
	req, err := http.NewRequest("GET", server.Config.APIBaseURL+"/v2/jobs", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Pganalyze-Api-Key", server.Config.APIKey)
	req.Header.Set("Pganalyze-System-Id", server.Config.SystemID)
	req.Header.Set("Pganalyze-System-Type", server.Config.SystemType)
	req.Header.Set("Pganalyze-System-Scope", server.Config.SystemScope)
	req.Header.Set("Pganalyze-System-Id-Fallback", server.Config.SystemIDFallback)
	req.Header.Set("Pganalyze-System-Type-Fallback", server.Config.SystemTypeFallback)
	req.Header.Set("Pganalyze-System-Scope-Fallback", server.Config.SystemScopeFallback)
	req.Header.Set("User-Agent", util.CollectorNameAndVersion)
	req.Header.Add("Accept", "application/json")

	resp, err := server.Config.HTTPClientWithRetry.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK || len(body) == 0 {
		return nil, fmt.Errorf("error when getting jobs: %s", body)
	}

	parsedBody := jobsApiResponse{}
	err = json.Unmarshal(body, &parsedBody)
	if err != nil {
		return nil, err
	}

	return parsedBody.Jobs, nil
}

func updateJobState(server *state.Server, jobId int, jobState string, jobResult []byte) error {
	data := url.Values{"state": {jobState}, "result": {string(jobResult)}}
	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/v2/jobs/%d", server.Config.APIBaseURL, jobId), strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Pganalyze-Api-Key", server.Config.APIKey)
	req.Header.Set("Pganalyze-System-Id", server.Config.SystemID)
	req.Header.Set("Pganalyze-System-Type", server.Config.SystemType)
	req.Header.Set("Pganalyze-System-Scope", server.Config.SystemScope)
	req.Header.Set("Pganalyze-System-Id-Fallback", server.Config.SystemIDFallback)
	req.Header.Set("Pganalyze-System-Type-Fallback", server.Config.SystemTypeFallback)
	req.Header.Set("Pganalyze-System-Scope-Fallback", server.Config.SystemScopeFallback)
	req.Header.Set("User-Agent", util.CollectorNameAndVersion)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")

	resp, err := server.Config.HTTPClientWithRetry.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error when updating job state: %s", body)
	}

	return nil
}

// RunTestJob - Runs globalCollectionOpts.TestJob with globalCollectionOpts.TestJobParameters for all servers and outputs the result to stdout
func RunTestJob(ctx context.Context, servers []*state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	for _, server := range servers {
		prefixedLogger := logger.WithPrefix(server.Config.SectionName)

		jobState, jobResult := runJob(ctx, server, prefixedLogger, globalCollectionOpts, globalCollectionOpts.TestJob, []byte(globalCollectionOpts.TestJobParameters))

		var out bytes.Buffer
		json.Indent(&out, jobResult, "", "\t")
		fmt.Printf("%s:\n%s\n\n", jobState, out.String())
	}
}

// RunJobs - Retrieves current jobs from the API, runs them and submits status updates to the API
func RunJobs(ctx context.Context, servers []*state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	for _, server := range servers {
		prefixedLogger := logger.WithPrefix(server.Config.SectionName)

		jobs, err := getJobs(server)
		if err != nil {
			prefixedLogger.PrintError("Failed to get jobs: %s", err)
			continue
		}

		if len(jobs) == 0 {
			continue
		}

		for _, job := range jobs {
			prefixedLogger.PrintVerbose("Starting %s job (id = %d)", job.Kind, job.Id)

			err := updateJobState(server, job.Id, "started", []byte("{}"))
			if err != nil {
				prefixedLogger.PrintError("Failed to update job state: %s", err)
				continue
			}

			jobState, jobResult := runJob(ctx, server, prefixedLogger, globalCollectionOpts, job.Kind, job.Parameters)
			prefixedLogger.PrintInfo("Finished %s job (state = %s, id = %d)", job.Kind, jobState, job.Id)

			err = updateJobState(server, job.Id, jobState, jobResult)
			if err != nil {
				prefixedLogger.PrintError("Failed to update job state: %s", err)
				continue
			}
		}
	}
}

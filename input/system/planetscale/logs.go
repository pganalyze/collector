package planetscale

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const (
	defaultAPIURL  = "https://api.planetscale.com"
	defaultLogsURL = "https://logs.psdb.cloud"
)

// LogEntry represents a single log entry from the PlanetScale logs API
type LogEntry struct {
	Time      string `json:"_time"`
	StreamID  string `json:"_stream_id"`
	Msg       string `json:"_msg"`
	Component string `json:"planetscale.component"`
	Role      string `json:"planetscale.role"`
	BranchID  string `json:"planetscale.database_branch_id"`
	Pod       string `json:"planetscale.pod"`
}

// signatureResponse represents the response from the logs signature API
type signatureResponse struct {
	Data struct {
		Sig string `json:"sig"`
		Exp string `json:"exp"`
	} `json:"data"`
}

// branchResponse represents the response from the branch API
type branchResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// cachedAuth holds cached authentication data for the PlanetScale logs API
type cachedAuth struct {
	branchID  string
	signature string
	expiry    int64
}

// LogStreamReader implements logs.LineReader for streaming PlanetScale log entries
type LogStreamReader struct {
	body       io.ReadCloser
	decoder    *json.Decoder
	logger     *util.Logger
	newestTime time.Time
	err        error
}

// ReadString reads the next log entry and returns its message
func (r *LogStreamReader) ReadString(delim byte) (string, error) {
	entry, err := r.Read()
	if err != nil {
		return "", err
	}

	// Return the message (ensure it ends with newline)
	msg := entry.Msg
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	return msg, nil
}

// Read reads the next log entry and returns the full entry struct.
// This is useful for external iteration when you need access to all entry fields.
func (r *LogStreamReader) Read() (*LogEntry, error) {
	if r.err != nil {
		return nil, r.err
	}

	for {
		var entry LogEntry
		if err := r.decoder.Decode(&entry); err != nil {
			if err == io.EOF {
				r.err = io.EOF
				return nil, io.EOF
			}
			// Log parse errors but continue trying
			if r.logger != nil {
				r.logger.PrintVerbose("PlanetScale: failed to parse log entry: %v", err)
			}
			continue
		}

		// Track newest timestamp
		if entryTime, err := time.Parse(time.RFC3339Nano, entry.Time); err == nil {
			if r.newestTime.IsZero() || entryTime.After(r.newestTime) {
				r.newestTime = entryTime
			}
		}

		return &entry, nil
	}
}

// GetNewestTimestamp returns the newest timestamp seen during streaming
func (r *LogStreamReader) GetNewestTimestamp() time.Time {
	return r.newestTime
}

// Close closes the underlying HTTP response body
func (r *LogStreamReader) Close() error {
	return r.body.Close()
}

// HTTPError represents an HTTP error response with status code
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
	}
	return fmt.Sprintf("HTTP %d", e.StatusCode)
}

type authCacheKey = struct{ org, db, branch string }

// serverAuthCache maps server identifiers to their cached auth data
var serverAuthCache = map[authCacheKey]cachedAuth{}

// DownloadLogFiles fetches logs from PlanetScale's logs API.
// Called every 30 seconds by the log download scheduler.
func DownloadLogFiles(ctx context.Context, server *state.Server, logger *util.Logger) (
	state.PersistedLogState,
	[]state.LogFile, []state.PostgresQuerySample,
	error,
) {
	config := server.Config
	psl := server.LogPrevState

	apiURL := cmp.Or(config.PlanetScaleAPIURL, defaultAPIURL)
	logsURL := cmp.Or(config.PlanetScaleLogsURL, defaultLogsURL)

	cacheKey := authCacheKey{config.PlanetScaleOrg, config.PlanetScaleDatabase, config.PlanetScaleBranch}
	auth := serverAuthCache[cacheKey]
	defer func() { serverAuthCache[cacheKey] = auth }()

	// Get branch ID (cached for collector lifetime)
	if auth.branchID == "" {
		branchID, err := GetBranchID(ctx, config.HTTPClient, apiURL, config.PlanetScaleTokenID, config.PlanetScaleTokenSecret,
			config.PlanetScaleOrg, config.PlanetScaleDatabase, config.PlanetScaleBranch)
		if err != nil {
			return psl, nil, nil, fmt.Errorf("failed to get branch ID: %w", err)
		}
		auth.branchID = branchID
		logger.PrintVerbose("PlanetScale: resolved branch %s to ID %s", config.PlanetScaleBranch, branchID)
	}

	// Get signature, refreshing if we don't have one or if it's expired
	now := time.Now().UTC().Unix()
	if auth.signature == "" || auth.expiry <= now {
		sig, exp, err := GetSignature(ctx, config.HTTPClient, apiURL, config.PlanetScaleTokenID, config.PlanetScaleTokenSecret,
			config.PlanetScaleOrg, config.PlanetScaleDatabase, config.PlanetScaleBranch)
		if err != nil {
			return psl, nil, nil, fmt.Errorf("failed to get signature: %w", err)
		}
		auth.signature = sig
		auth.expiry = exp
		logger.PrintVerbose("PlanetScale: obtained new signature, expires at %d", exp)
	}

	logReader, err := QueryLogs(
		ctx, config.HTTPClient, logsURL, auth.branchID,
		auth.signature, auth.expiry, psl.PlanetScaleLastTimestamp, 1000)
	if err != nil {
		// If we get a 403, clear the cached signature and return error to retry next cycle
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusForbidden {
			auth.signature = ""
			auth.expiry = 0
			serverAuthCache[cacheKey] = auth
		}
		return psl, nil, nil, fmt.Errorf("failed to query logs: %w", err)
	}
	// Add logger to the reader for verbose output
	logReader.logger = logger
	defer logReader.Close()

	// Only process lines newer than 2 minutes ago (similar to RDS)
	linesNewerThan := time.Now().UTC().Add(-2 * time.Minute)

	// Parse the log content using the standard parser, streaming directly
	logLines, samples := logs.ParseAndAnalyzeBuffer(logReader, linesNewerThan, server)

	// Create log file
	logFile, err := state.NewLogFile("planetscale-logs")
	if err != nil {
		return psl, nil, nil, fmt.Errorf("error initializing log file: %w", err)
	}
	logFile.LogLines = logLines

	// Update persisted state with the newest timestamp
	if newestTime := logReader.GetNewestTimestamp(); !newestTime.IsZero() {
		psl.PlanetScaleLastTimestamp = newestTime
	}

	var logFiles []state.LogFile
	if len(logLines) > 0 {
		logFiles = append(logFiles, logFile)
	}

	return psl, logFiles, samples, nil
}

// GetBranchID fetches the branch ID from the PlanetScale API
func GetBranchID(ctx context.Context, httpClient *http.Client, apiURL, tokenID, tokenSecret, org, database, branch string) (string, error) {
	reqURL := fmt.Sprintf("%s/v1/organizations/%s/databases/%s/branches/%s",
		apiURL, org, database, branch)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", tokenID+":"+tokenSecret)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}
	}

	var branchResp branchResponse
	if err := json.NewDecoder(resp.Body).Decode(&branchResp); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	return branchResp.ID, nil
}

// GetSignature fetches a log access signature from the PlanetScale API
func GetSignature(ctx context.Context, httpClient *http.Client, apiURL, tokenID, tokenSecret, org, database, branch string) (string, int64, error) {
	reqURL := fmt.Sprintf("%s/internal/organizations/%s/databases/%s/branches/%s/logs/signatures",
		apiURL, org, database, branch)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", 0, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", tokenID+":"+tokenSecret)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", 0, &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}
	}

	var sigResp signatureResponse
	if err := json.NewDecoder(resp.Body).Decode(&sigResp); err != nil {
		return "", 0, fmt.Errorf("decoding response: %w", err)
	}

	expiry, err := strconv.ParseInt(sigResp.Data.Exp, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("parsing expiry: %w", err)
	}

	return sigResp.Data.Sig, expiry, nil
}

// QueryLogs fetches logs from the PlanetScale logs API and returns a streaming reader.
// If since is non-zero, only logs after that time will be returned.
func QueryLogs(ctx context.Context, httpClient *http.Client, logsURL, branchID, sig string, expiry int64, since time.Time, limit int) (*LogStreamReader, error) {
	params := url.Values{}
	params.Set("sig", sig)
	params.Set("exp", strconv.FormatInt(expiry, 10))

	// Build query with optional time filter
	var query strings.Builder
	query.WriteString("planetscale.component:postgres planetscale.role:primary")
	if !since.IsZero() {
		fmt.Fprintf(&query, " _time:>%s", since.Format(time.RFC3339Nano))
	}
	// Add sorting to ensure chronological order
	query.WriteString(" | sort by (_time)")
	params.Set("query", query.String())
	params.Set("limit", strconv.Itoa(limit))

	reqURL := fmt.Sprintf("%s/logs/branch/%s/query?%s",
		logsURL, branchID, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}
	}

	// Response is newline-delimited JSON, stream it through a custom reader
	return &LogStreamReader{
		body:    resp.Body,
		decoder: json.NewDecoder(resp.Body),
	}, nil
}

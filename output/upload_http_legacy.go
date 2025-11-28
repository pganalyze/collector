package output

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func uploadSnapshot(ctx context.Context, httpClient *http.Client, grant *state.Grant, logger *util.Logger, data []byte, filename string) (string, error) {
	var err error

	if !grant.ValidForS3Until.After(time.Now()) {
		return "", fmt.Errorf("Error - can't upload without valid S3 grant")
	}

	if grant.S3URL == "" && grant.LocalDir != "" {
		location := grant.LocalDir + filename
		err = os.MkdirAll(filepath.Dir(location), 0755)
		if err != nil {
			logger.PrintError("Error creating target directory: %s", err)
			return "", err
		}

		err = os.WriteFile(location, data, 0644)
		if err != nil {
			logger.PrintError("Error writing local file: %s", err)
			return "", err
		}
		return location, nil
	}

	logger.PrintVerbose("Successfully prepared S3 request - size of request body: %.4f MB", float64(len(data))/1024.0/1024.0)

	return uploadToS3(ctx, httpClient, grant.S3URL, grant.S3Fields, data, filename)
}

type s3UploadResponse struct {
	Location string
	Bucket   string
	Key      string
}

func uploadToS3(ctx context.Context, httpClient *http.Client, S3URL string, S3Fields map[string]string, data []byte, filename string) (string, error) {
	var err error
	var formBytes bytes.Buffer

	writer := multipart.NewWriter(&formBytes)

	for key, val := range S3Fields {
		err = writer.WriteField(key, val)
		if err != nil {
			return "", err
		}
	}

	part, _ := writer.CreateFormFile("file", filename)
	_, err = part.Write(data)
	if err != nil {
		return "", err
	}

	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", S3URL, &formBytes)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("Bad S3 upload return code %s (expected 201 Created), body: %s", resp.Status, body)
	}

	var s3Resp s3UploadResponse
	err = xml.Unmarshal(body, &s3Resp)
	if err != nil {
		return "", err
	}

	return s3Resp.Key, nil
}

func submitSnapshot(ctx context.Context, server *state.Server, testRun bool, logger *util.Logger, s3Location string, collectedAt time.Time, compact bool) error {
	requestURL := server.Config.APIBaseURL + "/v2/snapshots"

	if testRun {
		requestURL = server.Config.APIBaseURL + "/v2/snapshots/test"
	} else if compact {
		requestURL = server.Config.APIBaseURL + "/v2/snapshots/compact"
	}

	data := url.Values{
		"s3_location":  {s3Location},
		"collected_at": {fmt.Sprintf("%d", collectedAt.Unix())},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header = config.APIHeaders(server.Config, testRun)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json,text/plain")

	resp, err := server.Config.HTTPClientWithRetry.Do(req)
	if err != nil {
		return util.CleanHTTPError(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Error when submitting: %s\n", body)
	}

	if testRun {
		contentType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
		if err != nil {
			return fmt.Errorf("Error decoding response: %s\n", err)
		}

		var msg string

		if contentType == "application/json" {
			var jsonBody struct {
				Message string `json:"message"`
			}
			err = json.Unmarshal(body, &jsonBody)
			if err != nil {
				return fmt.Errorf("Error decoding response: %s\n", err)
			}
			msg = jsonBody.Message
		} else {
			msg = string(body)
		}

		if len(msg) > 0 {
			logger.PrintInfo("  %s", msg)
		}
	}

	return nil
}

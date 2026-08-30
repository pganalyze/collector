package output

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
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

func markTestRun(ctx context.Context, server *state.Server, opts state.CollectionOpts) error {
	req, err := http.NewRequestWithContext(ctx, "POST", server.Config.APIBaseURL+"/v2/snapshots/test", nil)
	if err != nil {
		return err
	}

	req.Header = config.APIHeaders(server.Config, opts.TestRun, opts.StartedAt)

	resp, err := server.Config.HTTPClientWithRetry.Do(req)
	if err != nil {
		return util.CleanHTTPError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("Error when marking test run: %s", body)
	}

	return nil
}

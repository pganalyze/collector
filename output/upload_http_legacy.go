package output

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func uploadSnapshot(ctx context.Context, httpClient *http.Client, grant *state.Grant, logger *util.Logger, data []byte) error {
	if !grant.ValidForS3Until.After(time.Now()) {
		return fmt.Errorf("Error - can't upload without valid S3 grant")
	}

	logger.PrintVerbose("Successfully prepared S3 request - size of request body: %.4f MB", float64(len(data))/1024.0/1024.0)

	return uploadToS3(ctx, httpClient, grant.S3URL, grant.S3Fields, data)
}

func uploadToS3(ctx context.Context, httpClient *http.Client, S3URL string, S3Fields map[string]string, data []byte) error {
	var err error
	var formBytes bytes.Buffer

	writer := multipart.NewWriter(&formBytes)

	for key, val := range S3Fields {
		err = writer.WriteField(key, val)
		if err != nil {
			return err
		}
	}

	part, _ := writer.CreateFormFile("file", "snapshot")
	_, err = part.Write(data)
	if err != nil {
		return err
	}

	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", S3URL, &formBytes)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Bad S3 upload return code %s (expected 201 Created), body: %s", resp.Status, body)
	}

	return nil
}

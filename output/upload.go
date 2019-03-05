package output

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type s3UploadResponse struct {
	Location string
	Bucket   string
	Key      string
}

func uploadCompactSnapshot(httpClient *http.Client, s3 state.GrantS3, logger *util.Logger, data bytes.Buffer, filename string) (string, error) {
	if s3.S3URL == "" {
		return "", fmt.Errorf("Error - can't upload without valid S3 URL")
	}

	logger.PrintVerbose("Successfully prepared S3 request - size of request body: %.4f MB", float64(data.Len())/1024.0/1024.0)

	return uploadToS3(httpClient, s3.S3URL, s3.S3Fields, logger, data.Bytes(), filename)
}

func uploadSnapshot(httpClient *http.Client, grant state.Grant, logger *util.Logger, data bytes.Buffer, filename string) (string, error) {
	var err error

	if !grant.Valid {
		return "", fmt.Errorf("Error - can't upload without valid S3 grant")
	}

	if grant.S3URL == "" && grant.LocalDir != "" {
		location := grant.LocalDir + filename
		err = os.MkdirAll(filepath.Dir(location), 0755)
		if err != nil {
			logger.PrintError("Error creating target directory: %s", err)
			return "", err
		}

		err = ioutil.WriteFile(location, data.Bytes(), 0644)
		if err != nil {
			logger.PrintError("Error writing local file: %s", err)
			return "", err
		}
		return location, nil
	}

	logger.PrintVerbose("Successfully prepared S3 request - size of request body: %.4f MB", float64(data.Len())/1024.0/1024.0)

	return uploadToS3(httpClient, grant.S3URL, grant.S3Fields, logger, data.Bytes(), filename)
}

func uploadToS3(httpClient *http.Client, S3URL string, S3Fields map[string]string, logger *util.Logger, data []byte, filename string) (string, error) {
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

	req, err := http.NewRequest("POST", S3URL, &formBytes)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("Bad S3 upload return code %s (should be 201 Created), body: %s", resp.Status, body)
	}

	var s3Resp s3UploadResponse
	err = xml.Unmarshal(body, &s3Resp)
	if err != nil {
		return "", err
	}

	return s3Resp.Key, nil
}

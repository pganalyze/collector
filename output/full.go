package output

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/output/transform"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/satori/go.uuid"
)

func SendFull(db state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, newState state.State, diffState state.DiffState, collectedIntervalSecs uint32) error {
	var err error
	var data []byte

	snapshotUUID := uuid.NewV4()
	collectedAt := time.Now()
	s := transform.StateToSnapshot(newState, diffState)

	s.SnapshotVersionMajor = 1
	s.SnapshotVersionMinor = 0
	s.CollectorVersion = util.CollectorNameAndVersion

	s.SnapshotUuid = snapshotUUID.String()
	s.CollectedAt, _ = ptypes.TimestampProto(collectedAt)
	s.CollectedIntervalSecs = collectedIntervalSecs

	data, err = proto.Marshal(&s)
	if err != nil {
		logger.PrintError("Error marshaling protocol buffers")
		return err
	}

	var compressedData bytes.Buffer
	w := zlib.NewWriter(&compressedData)
	w.Write(data)
	w.Close()

	if !collectionOpts.SubmitCollectedData {
		debugOutputAsJSON(logger, compressedData)
		return nil
	}

	s3Location, err := uploadSnapshot(db, collectionOpts, logger, compressedData, snapshotUUID)
	if err != nil {
		logger.PrintError("Error uploading to S3: %s", err)
		return err
	}

	return submitSnapshot(db, collectionOpts, logger, s3Location, collectedAt)
}

func debugOutputAsJSON(logger *util.Logger, compressedData bytes.Buffer) {
	var err error
	var data bytes.Buffer

	r, err := zlib.NewReader(&compressedData)
	if err != nil {
		logger.PrintError("Failed to decompress protocol buffers: %s", err)
		return
	}
	defer r.Close()

	io.Copy(&data, r)

	s := &pganalyze_collector.FullSnapshot{}
	if err = proto.Unmarshal(data.Bytes(), s); err != nil {
		logger.PrintError("Failed to re-read protocol buffers: %s", err)
		return
	}

	var out bytes.Buffer
	dataJSON, _ := json.Marshal(s)
	json.Indent(&out, dataJSON, "", "\t")
	logger.PrintInfo("Dry run - data that would have been sent will be output on stdout:\n")
	logger.PrintInfo(out.String())
}

type s3UploadResponse struct {
	Location string
	Bucket   string
	Key      string
}

func uploadSnapshot(server state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, compressedData bytes.Buffer, snapshotUUID uuid.UUID) (string, error) {
	logger.PrintVerbose("Successfully prepared request - size of request body: %.4f MB", float64(compressedData.Len())/1024.0/1024.0)

	var formBytes bytes.Buffer
	var err error

	writer := multipart.NewWriter(&formBytes)

	for key, val := range server.Grant.S3Fields {
		err = writer.WriteField(key, val)
		if err != nil {
			return "", err
		}
	}

	part, _ := writer.CreateFormFile("file", snapshotUUID.String())
	_, err = part.Write(compressedData.Bytes())
	if err != nil {
		return "", err
	}

	writer.Close()

	req, err := http.NewRequest("POST", server.Grant.S3URL, &formBytes)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
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

func submitSnapshot(server state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, s3Location string, collectedAt time.Time) error {
	requestURL := server.Config.APIBaseURL + "/v2/snapshots"

	if collectionOpts.TestRun {
		requestURL = server.Config.APIBaseURL + "/v2/snapshots/test"
	}

	data := url.Values{
		"s3_location":  {s3Location},
		"collected_at": {fmt.Sprintf("%d", collectedAt.Unix())},
	}

	req, err := http.NewRequest("POST", requestURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Pganalyze-Api-Key", server.Config.APIKey)
	req.Header.Set("User-Agent", util.CollectorNameAndVersion)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json,text/plain")

	resp, err := http.DefaultClient.Do(req)
	// TODO: We could consider re-running on error (e.g. if it was a temporary server issue)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Error when submitting: %s\n", body)
	}

	if len(body) > 0 {
		logger.PrintInfo("%s", body)
	} else {
		logger.PrintInfo("Submitted snapshot successfully")
	}

	return nil
}

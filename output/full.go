package output

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/pganalyze/collector/output/snapshot"
	"github.com/pganalyze/collector/output/transform"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func SendFull(db state.Database, collectionOpts state.CollectionOpts, logger *util.Logger, newState state.State, diffState state.DiffState) error {
	var err error
	var data []byte

	collectedAt := time.Now()
	s := transform.StateToSnapshot(newState, diffState)

	s.CollectorVersion = "pganalyze-collector 0.9.0rc8"
	s.CollectedAt, err = ptypes.TimestampProto(collectedAt)
	if err != nil {
		logger.PrintError("Error initializating snapshot timestamp")
		return err
	}

	data, err = proto.Marshal(&s)
	if err != nil {
		logger.PrintError("Error marshaling protocol buffers")
		return err
	}

	var compressedData bytes.Buffer
	w := zlib.NewWriter(&compressedData)
	w.Write(data)
	w.Close()

	if true { //!collectionOpts.SubmitCollectedData {
		debugOutputAsJSON(logger, compressedData)
		return nil
	}

	return submitSnapshot(db, collectionOpts, logger, compressedData, collectedAt)
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

	s := &snapshot.Snapshot{}
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

func submitSnapshot(db state.Database, collectionOpts state.CollectionOpts, logger *util.Logger, compressedData bytes.Buffer, collectedAt time.Time) error {
	logger.PrintVerbose("Successfully prepared request - size of request body: %.4f MB", float64(compressedData.Len())/1024.0/1024.0)

	requestURL := db.Config.APIBaseURL + "/v2/snapshots"

	if collectionOpts.TestRun {
		requestURL = db.Config.APIBaseURL + "/v2/snapshots/test"
	}

	req, err := http.NewRequest("POST", requestURL, &compressedData)
	if err != nil {
		return err
	}

	req.Header.Set("Pganalyze-Api-Key", db.Config.APIKey)
	req.Header.Set("Pganalyze-Collected-At", fmt.Sprintf("%d", collectedAt.Unix()))
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Add("Accept", "text/plain")

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

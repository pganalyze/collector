package output

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/pganalyze/collector/output/snapshot"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func transformState(newState state.State, diffState state.DiffState) (s snapshot.Snapshot, err error) {
	// Go through all statements, creating references as needed
	for _, statement := range diffState.Statements {
		// FIXME: This is (very!) incomplete code

		var queryInformation snapshot.QueryInformation

		queryInformation.NormalizedQuery = statement.NormalizedQuery

		s.QueryInformations = append(s.QueryInformations, &queryInformation)
	}

	for _, relation := range newState.Relations {
		ref := snapshot.RelationReference{
			DatabaseIdx:  0,
			SchemaName:   relation.SchemaName,
			RelationName: relation.RelationName,
		}
		idx := int32(len(s.RelationReferences))
		s.RelationReferences = append(s.RelationReferences, &ref)

		// Information
		info := snapshot.RelationInformation{
			RelationRef:  idx,
			RelationType: relation.RelationType,
		}
		if relation.ViewDefinition != "" {
			info.ViewDefinition = &snapshot.NullString{Valid: true, Value: relation.ViewDefinition}
		}
		// TODO: Add columns and constraints here
		s.RelationInformations = append(s.RelationInformations, &info)

		// Statistic
		stats, exists := diffState.RelationStats[relation.Oid]
		if exists {
			statistic := snapshot.RelationStatistic{
				RelationRef: idx,
				SizeBytes:   stats.SizeBytes,
				SeqScan:     stats.SeqScan,
				NTupUpd:     stats.NTupUpd,
			}
			// TODO: Complete set of stats
			s.RelationStatistics = append(s.RelationStatistics, &statistic)
		}
	}

	return
}

func SendFull(db state.Database, collectionOpts state.CollectionOpts, logger *util.Logger, newState state.State, diffState state.DiffState) (err error) {
	s, err := transformState(newState, diffState)
	if err != nil {
		logger.PrintError("Error transforming state into snapshot: %s", err)
		return
	}

	// FIXME: Need to transform state into snapshot
	statsProto, err := proto.Marshal(&s)
	if err != nil {
		logger.PrintError("Error marshaling statistics: %s", err)
		return
	}

	if true { //!collectionOpts.SubmitCollectedData {
		statsReRead := &snapshot.Snapshot{}
		if err = proto.Unmarshal(statsProto, statsReRead); err != nil {
			log.Fatalln("Failed to re-read stats:", err)
		}

		var out bytes.Buffer
		statsJSON, _ := json.Marshal(statsReRead)
		json.Indent(&out, statsJSON, "", "\t")
		logger.PrintInfo("Dry run - data that would have been sent will be output on stdout:\n")
		fmt.Print(out.String())
		return
	}

	var compressedJSON bytes.Buffer
	w := zlib.NewWriter(&compressedJSON)
	w.Write(statsProto)
	w.Close()

	requestURL := db.Config.APIBaseURL + "/v1/snapshots"

	if collectionOpts.TestRun {
		requestURL = db.Config.APIBaseURL + "/v1/snapshots/test"
	}

	data := url.Values{
		"data":            {compressedJSON.String()},
		"data_compressor": {"zlib"},
		"api_key":         {db.Config.APIKey},
		"submitter":       {"pganalyze-collector 0.9.0rc7"},
		"no_reset":        {"true"},
		"query_source":    {"pg_stat_statements"},
		"collected_at":    {fmt.Sprintf("%d", time.Now().Unix())},
	}

	encodedData := data.Encode()

	req, err := http.NewRequest("POST", requestURL, strings.NewReader(encodedData))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json,text/plain")

	logger.PrintVerbose("Successfully prepared request - size of request body: %.4f MB", float64(len(encodedData))/1024.0/1024.0)

	resp, err := http.DefaultClient.Do(req)
	// TODO: We could consider re-running on error (e.g. if it was a temporary server issue)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("Error when submitting: %s\n", body)
		return
	}

	if len(body) > 0 {
		logger.PrintInfo("%s", body)
	} else {
		logger.PrintInfo("Submitted snapshot successfully")
	}

	return
}

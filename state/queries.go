package state

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"github.com/klauspost/compress/zstd"
	"github.com/pganalyze/collector/util"
	pg_query "github.com/pganalyze/pg_query_go/v6"
	"sync"
	"unicode/utf8"
)

type Queries struct {
	fingerprints      *Fingerprints
	queries           []QuerySummary
	compressedQueries [][]byte
	newQueries        chan NewQuery
	queryLock         sync.RWMutex
}

type QuerySummary struct {
	Fingerprint     int64
	TruncatedQuery  string
	NormalizedQuery string
	StatementTypes  []string
	TableNames      []string
}

type NewQuery struct {
	fingerprint            int64
	text                   string
	filterQueryText        string
	trackActivityQuerySize int
}

func NewQueries(ctx context.Context, wg *sync.WaitGroup) *Queries {
	c := &Queries{
		fingerprints: NewFingerprints(),
		queryLock:    sync.RWMutex{},
		newQueries:   make(chan NewQuery, 10),
	}
	wg.Add(1)
	go func(ctx context.Context, wg *sync.WaitGroup, c *Queries) {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case q := <-c.newQueries:
				c.process(q.fingerprint, q.text, q.filterQueryText, q.trackActivityQuerySize)
				// TODO: report errors? Note the code needs to skip things like <truncated query>
			}
		}
	}(ctx, wg, c)
	return c
}

// Marks the query ID as new, so future pg_stat_statements(showtext=true) calls try to capture the query details
func (c *Queries) Track(queryID int64) {
	c.fingerprints.Get(queryID, false)
}

func (c *Queries) Add(queryID int64, text string, filterQueryText string, trackActivityQuerySize int) int64 {
	fingerprint, new := c.fingerprints.Add(queryID, text, filterQueryText, trackActivityQuerySize)
	if new {
		c.newQueries <- NewQuery{fingerprint, text, filterQueryText, trackActivityQuerySize}
	}
	return fingerprint
}

// Skips pg_query calls if query text is over 200k characters,
// and uses simple string truncation if query text is over 10k characters
func (c *Queries) process(fingerprint int64, text string, filterQueryText string, trackActivityQuerySize int) error {
	var query QuerySummary
	query.Fingerprint = fingerprint
	// Hmm, should we truncate query text as part of the initial SELECT to avoid huge query texts?
	query.TruncatedQuery = truncate(text, 100)
	if len(text) < 200_000 {
		query.NormalizedQuery = util.NormalizeQuery(text, filterQueryText, trackActivityQuerySize)
		truncateSize := 100
		if len(text) > 10_000 {
			truncateSize = -1 // Disable pg_query truncation
		}
		summary, err := pg_query.Summary(text, truncateSize)
		if err != nil {
			return err
		}
		if truncateSize != -1 {
			query.TruncatedQuery = summary.TruncatedQuery
		}
		query.StatementTypes = summary.StatementTypes
		for _, table := range summary.Tables {
			query.TableNames = append(query.TableNames, table.Name)
		}
	}
	c.queryLock.Lock()
	defer c.queryLock.Unlock()
	c.queries = append(c.queries, query)
	if len(c.queries) >= 500 {
		bytes, err := compressQueries(c.queries)
		c.queries = nil
		if err != nil {
			return err
		}
		c.compressedQueries = append(c.compressedQueries, bytes)
	}
	return nil
}

func (c *Queries) TakeQueries(fn func(q QuerySummary)) error {
	c.queryLock.Lock()
	defer c.queryLock.Unlock()
	for _, query := range c.queries {
		fn(query)
	}
	c.queries = nil
	for _, bytes := range c.compressedQueries {
		var queries *[]QuerySummary
		err := decompressQueries(bytes, queries)
		if err != nil {
			return err
		}
		for _, query := range *queries {
			fn(query)
		}
	}
	c.compressedQueries = nil
	return nil
}

func (c *Queries) Cleanup() {
	c.fingerprints.Cleanup()
}

func (c *Queries) ProcessNewQueryIDs() []int64 {
	return c.fingerprints.ProcessNewQueryIDs()
}

// A string truncate function that avoids execessive allocations for large strings
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	// Start checking the byte slice from the end of the desired length
	// and move backward until we find a valid rune start position.
	for !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

func compressQueries(data []QuerySummary) ([]byte, error) {
	var buf bytes.Buffer
	zw, err := zstd.NewWriter(&buf)
	if err != nil {
		return nil, err
	}
	enc := gob.NewEncoder(zw)
	if err := enc.Encode(data); err != nil {
		zw.Close()
		return nil, fmt.Errorf("gob encoding error: %w", err)
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("zstd writer close error: %w", err)
	}
	return buf.Bytes(), nil
}

func decompressQueries(compressedBytes []byte, target *[]QuerySummary) error {
	zr, err := zstd.NewReader(bytes.NewReader(compressedBytes))
	if err != nil {
		return err
	}
	defer zr.Close()
	dec := gob.NewDecoder(zr)
	if err := dec.Decode(target); err != nil {
		return fmt.Errorf("gob decoding error: %w", err)
	}
	return nil
}

// TODO: include this normalize logic
// Maybe parts of it can be dropped? The Rust code notes this handles old collector versions
// pub fn normalize(received_query: &str, benchmark: &Benchmark) -> String {
//     let _bench = benchmark.measure("queries:normalize");
//     let mut query = match pg_query::normalize(received_query) {
//         Ok(query) => query,
//         Err(pg_query::Error::Parse(err)) if err == "UNENCRYPTED PASSWORD is no longer supported" => {
//             match pg_query::normalize(&received_query.replace("UNENCRYPTED PASSWORD", "PASSWORD")) {
//                 Ok(query) => query,
//                 Err(_) => received_query.to_string(),
//             }
//         }
//         _ => received_query.to_string(),
//     };
//     query = query.trim().trim_end_matches(";").to_string();
//     lazy_static! {
//         static ref R0_CHECK: Regex = Regex::new(r"\A\s*INSERT").unwrap();
//         static ref R0: Regex = Regex::new(
//             r"(?im)(VALUES\s+)(\s*\((?P<last_complete_row>[^\(\)]+)\),?)+(\s*\(([^\(\)]+)(\n?\z|\)),?)?"
//         )
//         .unwrap();
//         static ref R1: Regex = Regex::new(r"(?i)\ADEALLOCATE [A-z0-9_]+").unwrap();
//         static ref R2: Regex = Regex::new(r"(?i)\ADECLARE [A-z0-9_]+").unwrap();
//         static ref R3: Regex = Regex::new(r"(?i)\AFETCH (\d+) FROM [A-z0-9_]+").unwrap();
//         static ref R4: Regex = Regex::new(r"(?i)\ACLOSE [A-z0-9_]+").unwrap();
//     }
//     // Normalize VALUES lists by keeping only the last non-truncated row,
//     // to avoid storing a lot of repeated param refs for batch inserts.
//     if R0_CHECK.is_match(&query) {
//         query = R0.replace_all(&query, "$1(${last_complete_row})").to_string();
//     }
//     // Make it clear when fingerprinting grouped certain DDL together
//     query = R1.replace_all(&query, "DEALLOCATE prepared_statement").to_string();
//     query = R2.replace_all(&query, "DECLARE cursor").to_string();
//     query = R3.replace_all(&query, "FETCH $1 FROM cursor").to_string();
//     query = R4.replace_all(&query, "CLOSE cursor").to_string();
//     query
// }

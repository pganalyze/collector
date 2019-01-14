package logs

import (
	"sort"

	"github.com/pganalyze/collector/state"
)

type logRange struct {
	start int64
	end   int64
}

const replacementChar = 'X'

// ReplaceSecrets - Replaces the secrets of the specified kind with the replacement character in the text
func ReplaceSecrets(input string, logLines []state.LogLine, filterLogSecret []state.LogSecretKind) string {
	var goodRanges []logRange

	filterUnidentified := false
	for _, k := range filterLogSecret {
		if k == state.UnidentifiedLogSecret {
			filterUnidentified = true
		}
	}

	for _, logLine := range logLines {
		goodRanges = append(goodRanges, logRange{start: logLine.ByteStart, end: logLine.ByteContentStart})
		if logLine.ReviewedForSecrets {
			sort.Slice(logLine.SecretMarkers, func(i, j int) bool {
				return logLine.SecretMarkers[i].ByteStart < logLine.SecretMarkers[j].ByteEnd
			})
			var lastGood int64
			for _, m := range logLine.SecretMarkers {
				filter := false
				for _, k := range filterLogSecret {
					if m.Kind == k {
						filter = true
					}
				}
				if filter {
					goodRanges = append(goodRanges, logRange{start: logLine.ByteContentStart + lastGood, end: logLine.ByteContentStart + int64(m.ByteStart)})
					lastGood = int64(m.ByteEnd)
				}
			}
			if lastGood < (logLine.ByteEnd - logLine.ByteContentStart) {
				goodRanges = append(goodRanges, logRange{start: logLine.ByteContentStart + lastGood, end: logLine.ByteEnd})
			}
		} else if !filterUnidentified {
			goodRanges = append(goodRanges, logRange{start: logLine.ByteContentStart, end: logLine.ByteEnd})
		}
		goodRanges = append(goodRanges, logRange{start: logLine.ByteEnd, end: logLine.ByteEnd + 1}) // newline character
	}
	sort.Slice(goodRanges, func(i, j int) bool {
		return goodRanges[i].start < goodRanges[j].start
	})

	var lastGood int64
	output := []rune(input)
	for _, r := range goodRanges {
		for i := lastGood; i < r.start; i++ {
			output[i] = replacementChar
		}
		lastGood = r.end
	}
	if filterUnidentified {
		for i := lastGood; i < int64(len(input)); i++ {
			output[i] = replacementChar
		}
	}
	return string(output)
}

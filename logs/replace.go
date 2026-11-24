package logs

import (
	"slices"
	"sort"

	"github.com/google/uuid"
	"github.com/pganalyze/collector/state"
)

const replacement = "[redacted]"

func ReplaceSecrets(logLines []state.LogLine, filterLogSecret []state.LogSecretKind) {
	filterUnidentified := false
	for _, k := range filterLogSecret {
		if k == state.UnidentifiedLogSecret {
			filterUnidentified = true
		}
	}
	for idx, logLine := range logLines {
		if filterUnidentified && (!logLine.ReviewedForSecrets || (logLine.Classification == 0 && logLines[idx].ParentUUID == uuid.Nil)) {
			logLines[idx].Content = replacement + "\n"
		} else {
			sort.Slice(logLine.SecretMarkers, func(i, j int) bool {
				return logLine.SecretMarkers[i].ByteStart < logLine.SecretMarkers[j].ByteStart
			})
			content := []byte(logLine.Content)
			bytesChecked := -1
			offset := 0
			for _, m := range logLine.SecretMarkers {
				for _, k := range filterLogSecret {
					if m.Kind == k && m.ByteStart >= bytesChecked {
						r := []byte(replacement)
						if m.ByteStart == bytesChecked {
							// Directly follows the previously redacted marker - extend
							// that redaction instead of adding another one
							r = nil
						}
						content = slices.Replace(content, m.ByteStart-offset, m.ByteEnd-offset, r...)
						bytesChecked = m.ByteEnd
						offset += (m.ByteEnd - m.ByteStart) - len(r)
					}
				}
			}
			logLines[idx].Content = string(content)
		}
	}
}

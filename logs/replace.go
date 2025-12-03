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
				return logLine.SecretMarkers[i].ByteStart < logLine.SecretMarkers[j].ByteEnd
			})
			content := []byte(logLine.Content)
			bytesChecked := -1
			offset := 0
			for _, m := range logLine.SecretMarkers {
				for _, k := range filterLogSecret {
					if m.Kind == k && m.ByteStart > bytesChecked {
						content = slices.Replace(content, m.ByteStart-offset, m.ByteEnd-offset, []byte(replacement)...)
						bytesChecked = m.ByteEnd
						offset += (m.ByteEnd - m.ByteStart) - len(replacement)
					}
				}
			}
			logLines[idx].Content = string(content)
		}
	}
}

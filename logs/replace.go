package logs

import (
	"slices"
	"sort"

	"github.com/pganalyze/collector/state"
)

const replacement = "[redacted]"

func ReplaceSecrets(input []byte, logLines []state.LogLine, filterLogSecret []state.LogSecretKind) {
	filterUnidentified := false
	for _, k := range filterLogSecret {
		if k == state.UnidentifiedLogSecret {
			filterUnidentified = true
		}
	}
	for idx, logLine := range logLines {
		if filterUnidentified && logLines[idx].Classification == 0 {
			logLines[idx].Content = replacement + "\n"
		} else {
			content := input[logLines[idx].ByteContentStart:logLines[idx].ByteEnd]
			sort.Slice(logLine.SecretMarkers, func(i, j int) bool {
				return logLine.SecretMarkers[i].ByteStart < logLine.SecretMarkers[j].ByteEnd
			})
			bytesChecked := 0
			offset := 0
			for _, m := range logLine.SecretMarkers {
				for _, k := range filterLogSecret {
					if m.Kind == k && m.ByteStart > bytesChecked {
						content = slices.Replace(content, m.ByteStart-offset, m.ByteEnd-offset, []byte(replacement)...)
						bytesChecked = m.ByteEnd
						offset += max(1, len(replacement)-m.ByteEnd-m.ByteStart)
					}
				}
			}
			logLines[idx].Content = string(content)
		}
	}
}

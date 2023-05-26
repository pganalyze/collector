package heroku

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

type HttpSyslogMessage struct {
	HeaderTimestamp string
	HeaderProcID    string
	Content         []byte
	Path            string
}

// Reads log messages in the Syslog TCP protocol octet counting framing method
//
// See format documentation here:
// - https://devcenter.heroku.com/articles/log-drains#https-drains
// - https://datatracker.ietf.org/doc/html/rfc6587#section-3.4.1
//
// Discards all syslog messages not related to Heroku Postgres
func ReadHerokuPostgresSyslogMessages(r io.Reader) []HttpSyslogMessage {
	var out []HttpSyslogMessage

	reader := bufio.NewReader(r)
	for {
		lenStr, err := reader.ReadString(' ')
		if err != nil {
			break
		}
		lenStr = lenStr[:len(lenStr)-1]

		remainingBytes, err := strconv.ParseInt(lenStr, 10, 64)
		if err != nil {
			break
		}

		// PRI/VERSION (Skip)
		buf, err := reader.ReadSlice(' ')
		remainingBytes -= int64(len(buf))
		if err != nil {
			break
		}

		// TIMESTAMP
		headerTimestamp, err := reader.ReadString(' ')
		if err != nil {
			break
		}
		remainingBytes -= int64(len(headerTimestamp))
		headerTimestamp = headerTimestamp[:len(headerTimestamp)-1]

		// HOSTNAME (Skip)
		buf, err = reader.ReadSlice(' ')
		if err != nil {
			break
		}
		remainingBytes -= int64(len(buf))

		// APP-NAME
		headerAppName, err := reader.ReadString(' ')
		if err != nil {
			break
		}
		remainingBytes -= int64(len(headerAppName))
		headerAppName = headerAppName[:len(headerAppName)-1]

		// PROCID
		headerProcID, err := reader.ReadString(' ')
		if err != nil {
			break
		}
		remainingBytes -= int64(len(headerProcID))
		headerProcID = headerProcID[:len(headerProcID)-1]

		// MSGID (Skip)
		buf, err = reader.ReadSlice(' ')
		if err != nil {
			break
		}
		remainingBytes -= int64(len(buf))

		// Unexpected for Postgres log output, so skip this data
		if remainingBytes <= 0 {
			continue
		}

		bytes := make([]byte, remainingBytes)
		_, err = io.ReadFull(reader, bytes)
		if err != nil {
			break
		}

		if headerAppName == "app" && (headerProcID == "heroku-postgres" || strings.HasPrefix(headerProcID, "postgres.")) {
			item := HttpSyslogMessage{
				HeaderTimestamp: headerTimestamp,
				HeaderProcID:    headerProcID,
				Content:         bytes,
				Path:            "", // To be added later by caller
			}
			out = append(out, item)
		}
	}

	return out
}

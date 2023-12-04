package querysample

import (
	"net/url"
	"regexp"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v4"
)

var singleQuotedRegex = regexp.MustCompile(`^'(.*)'$`)
var metaCharacterRegex = regexp.MustCompile(`\\(.)`)

func parseTags(query string) map[string]string {
	tokens, err := pg_query.Scan(query)
	if err != nil {
		return nil
	}

	tags := make(map[string]string)
	for _, token := range tokens.GetTokens() {
		var comment string
		if token.Token == pg_query.Token_SQL_COMMENT {
			// Turn "--comment" into "comment"
			comment = query[token.Start+2 : token.End]
		} else if token.Token == pg_query.Token_C_COMMENT {
			// Turn "/*comment*/" into "comment"
			comment = query[token.Start+2 : token.End-2]
		} else {
			continue
		}
		for _, part := range strings.Split(strings.TrimSpace(comment), ",") {
			if sqlcommenterFormat(part) {
				// Parse sqlcommenter format (key='value')
				keyAndValue := strings.SplitN(part, "=", 2)
				// Remove surrounding single quotes (if present)
				value := strings.TrimSpace(keyAndValue[1])
				if match := singleQuotedRegex.FindStringSubmatch(value); match != nil {
					value = match[1]
				}
				// Decode key and value
				key := decodeString(strings.TrimSpace(keyAndValue[0]))
				value = decodeString(value)

				tags[key] = value
			} else if strings.Contains(part, ":") {
				// Parse marginalia format (key:value)
				keyAndValue := strings.SplitN(part, ":", 2)
				tags[strings.TrimSpace(keyAndValue[0])] = strings.TrimSpace(keyAndValue[1])
			} // TODO: Support other formats
		}
	}
	return tags
}

func sqlcommenterFormat(str string) bool {
	keyAndValue := strings.SplitN(str, "=", 2)
	// With sqlcommenter format (key='value'), key shouldn't include ":"
	return len(keyAndValue) == 2 && !strings.Contains(keyAndValue[0], ":")
}

func decodeString(str string) string {
	// From https://google.github.io/sqlcommenter/spec/#parsing
	// 1. Unescape the meta characters
	unescapedStr := metaCharacterRegex.ReplaceAllStringFunc(str, func(matched string) string {
		// This is simply returning the single char after the backslash "\"
		// Things like \n or \t are not supported
		return string(matched[1])
	})
	// 2. URL Decode
	decodedStr, err := url.QueryUnescape(unescapedStr)
	if err != nil {
		// Give up decode and use the original str
		decodedStr = unescapedStr
	}
	return decodedStr
}

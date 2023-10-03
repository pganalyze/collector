package querysample

import (
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v4"
)

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
			if strings.Contains(part, "=") {
				keyAndValue := strings.SplitN(part, "=", 2)
				tags[keyAndValue[0]] = keyAndValue[1]
				// TODO: Handle sqlcommenter's URL encoding logic, and quoting with single quotes
			} else if strings.Contains(part, ":") {
				keyAndValue := strings.SplitN(part, ":", 2)
				tags[keyAndValue[0]] = keyAndValue[1]
			} // TODO: Support other formats
		}
	}
	return tags
}

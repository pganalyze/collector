package postgres

const queryMarker string = "/* pganalyze-collector */"

const QueryMarkerSQL string = queryMarker + " "

func WithQueryMarker(sql string) string {
	return sql + "\n" + queryMarker
}

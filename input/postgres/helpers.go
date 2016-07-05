package postgres

import (
	"strconv"
	"strings"

	"github.com/pganalyze/collector/state"

	"gopkg.in/guregu/null.v3"
)

func unpackPostgresInt32Array(input null.String) (result []int32) {
	if !input.Valid {
		return
	}

	for _, cstr := range strings.Split(strings.Trim(input.String, "{}"), ",") {
		cint, _ := strconv.Atoi(cstr)
		result = append(result, int32(cint))
	}

	return
}

func unpackPostgresOidArray(input null.String) (result []state.Oid) {
	if !input.Valid {
		return
	}

	for _, cstr := range strings.Split(strings.Trim(input.String, "{}"), ",") {
		cint, _ := strconv.Atoi(cstr)
		result = append(result, state.Oid(cint))
	}

	return
}

func unpackPostgresStringArray(input null.String) (result []string) {
	if !input.Valid {
		return
	}

	result = strings.Split(strings.Trim(input.String, "{}"), ",")

	return
}

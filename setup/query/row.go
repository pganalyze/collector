package query

import (
	"errors"
	"fmt"
	"strconv"
)

var ErrNoRows = errors.New("no rows returned")

type Row []string

func (r Row) GetString(idx int) string {
	return r[idx]
}

func (r Row) GetBool(idx int) bool {
	return r[idx] == "t"
}

func (r Row) GetInt(idx int) int {
	num, err := strconv.Atoi(r[idx])
	if err != nil {
		panic(fmt.Sprintf("expected int in column %d; found %s", idx, r[idx]))
	}
	return num
}

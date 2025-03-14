package postgres_test

import (
	"context"
	"testing"

	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type collectionHelperTestpair struct {
	Name               string
	InputTypes         []string
	ExpectedExists     bool
	ExpectedReturnType string
}

var collectionHelperTests = []collectionHelperTestpair{
	{
		"explain_analyze",
		[]string{"text", "text[]", "text[]", "text[]"},
		true,
		"text",
	},
	{
		"get_stat_statements",
		[]string{"boolean"},
		true,
		"SETOF pg_stat_statements",
	},
	// Verify we don't match on a shorter set of arguments
	{
		"explain_analyze",
		[]string{"text", "text[]", "text[]"},
		false,
		"",
	},
	// Verify we don't match on a shorter set of arguments
	{
		"explain_analyze",
		[]string{"text", "text[]", "text[]", "text[]", "text[]"},
		false,
		"",
	},
	// Verify we don't match on different argument types
	{
		"explain_analyze",
		[]string{"text", "text[]", "text[]", "float"},
		false,
		"",
	},
	// Verify we allow default arguments to be ommitted
	{
		"get_stat_statements",
		[]string{},
		true,
		"SETOF pg_stat_statements",
	},
}

func TestCollectionFindHelper(t *testing.T) {
	db := setupTest(t)
	defer db.Close()

	_, err := db.Exec("CREATE EXTENSION IF NOT EXISTS pg_stat_statements")
	if err != nil {
		t.Fatalf("Could not create pg_stat_statements extension: %s", err)
	}

	_, err = db.Exec(util.GetStatStatementsHelper)
	if err != nil {
		t.Fatalf("Could not load get_stat_statements helper: %s", err)
	}

	ctx := context.Background()
	logger := &util.Logger{}
	server := &state.Server{}
	opts := state.CollectionOpts{}

	collection, err := postgres.NewCollection(ctx, logger, server, opts, db)
	if err != nil {
		t.Fatalf("Could not initialize collection struct: %s", err)
	}

	for _, pair := range collectionHelperTests {
		helperExists := collection.HelperExists(pair.Name, pair.InputTypes)
		if helperExists != pair.ExpectedExists {
			t.Errorf("Incorrect exists state for helper %s(%+v):\n got: %+v\n expected: %+v", pair.Name, pair.InputTypes, helperExists, pair.ExpectedExists)
		}

		helperReturnType := collection.HelperReturnType(pair.Name, pair.InputTypes)
		if helperReturnType != pair.ExpectedReturnType {
			t.Errorf("Incorrect return type for helper %s(%+v):\n got: %s\n expected: %s", pair.Name, pair.InputTypes, helperReturnType, pair.ExpectedReturnType)
		}
	}
}

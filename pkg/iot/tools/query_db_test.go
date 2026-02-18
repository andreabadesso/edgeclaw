package tools

import (
	"context"
	"strings"
	"testing"
)

func TestQueryTimescaleDB_Name(t *testing.T) {
	tool := NewQueryTimescaleDB(nil)
	if name := tool.Name(); name == "" {
		t.Error("Name() returned empty string")
	}
	if name := tool.Name(); name != "query_timescaledb" {
		t.Errorf("Name() = %q, want %q", name, "query_timescaledb")
	}
}

func TestQueryTimescaleDB_Description(t *testing.T) {
	tool := NewQueryTimescaleDB(nil)
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
	if !strings.Contains(desc, "sensor") {
		t.Error("Description() should mention sensors")
	}
}

func TestQueryTimescaleDB_Execute_EmptyQuery(t *testing.T) {
	tool := NewQueryTimescaleDB(nil)

	// No query key at all
	_, err := tool.Execute(context.Background(), map[string]string{})
	if err == nil {
		t.Error("expected error for missing query, got nil")
	}
	if !strings.Contains(err.Error(), "missing required argument") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "missing required argument")
	}

	// Empty string query
	_, err = tool.Execute(context.Background(), map[string]string{"query": ""})
	if err == nil {
		t.Error("expected error for empty query, got nil")
	}
}

func TestQueryTimescaleDB_Execute_RejectsNonSELECT(t *testing.T) {
	tool := NewQueryTimescaleDB(nil)

	nonSelectQueries := []string{
		"INSERT INTO sensors VALUES (now(), 'temp', 42.0, 'dev1')",
		"UPDATE sensors SET value = 0",
		"DELETE FROM sensors WHERE sensor_type = 'temp'",
		"DROP TABLE sensors",
		"  insert into sensors VALUES (now(), 'temp', 42.0, 'dev1')",
		"  drop table sensors",
	}

	for _, q := range nonSelectQueries {
		_, err := tool.Execute(context.Background(), map[string]string{"query": q})
		if err == nil {
			t.Errorf("expected error for query %q, got nil", q)
		}
		if err != nil && !strings.Contains(err.Error(), "only SELECT queries") {
			t.Errorf("query %q: error = %q, want it to contain %q", q, err.Error(), "only SELECT queries")
		}
	}
}

func TestQueryTimescaleDB_Execute_AcceptsSELECT(t *testing.T) {
	// With a nil db client, a valid SELECT should pass validation and then
	// panic when trying to use the nil client. We catch the panic to confirm
	// the validation logic accepted the query.
	tool := NewQueryTimescaleDB(nil)

	selectQueries := []string{
		"SELECT * FROM sensors",
		"  SELECT * FROM sensors",
		"select avg(value) from sensors",
		"  select count(*) from sensors_hourly",
	}

	for _, q := range selectQueries {
		func() {
			defer func() {
				if r := recover(); r == nil {
					// If no panic, Execute either succeeded or returned an error.
					// With nil db, we expect a panic, but this is fine either way.
				}
			}()
			_, err := tool.Execute(context.Background(), map[string]string{"query": q})
			// If we get here without panic, the error should be about the db, not validation
			if err != nil && strings.Contains(err.Error(), "only SELECT queries") {
				t.Errorf("query %q should have been accepted but was rejected", q)
			}
		}()
	}
}

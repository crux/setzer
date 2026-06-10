package main

import "testing"

func TestPrettyJSON(t *testing.T) {
	// Indentation, trailing newline, and — importantly — original key order
	// preserved (not alphabetized), so diffs stay stable.
	out, err := prettyJSON([]byte(`{"b":1,"a":{"y":2,"x":1}}`))
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"b\": 1,\n  \"a\": {\n    \"y\": 2,\n    \"x\": 1\n  }\n}\n"
	if string(out) != want {
		t.Fatalf("got:\n%q\nwant:\n%q", out, want)
	}

	if _, err := prettyJSON([]byte(`{not json`)); err == nil {
		t.Fatal("expected an error on malformed JSON")
	}
}

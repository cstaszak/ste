package main

import (
	"io"
	"strings"
	"testing"
)

// clearCredentials removes every credential name the check looks at, so one
// test does not see another one's environment.
func clearCredentials(t *testing.T) {
	t.Helper()
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	for _, name := range nearMisses {
		t.Setenv(name, "")
	}
}

// The correct name passes with no output.
func TestCheckCredentialsAcceptsTheCorrectName(t *testing.T) {
	clearCredentials(t)
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")

	var sb strings.Builder
	if err := checkCredentials(&sb); err != nil {
		t.Fatalf("error = %v, want none", err)
	}
	if sb.String() != "" {
		t.Errorf("output = %q, want none", sb.String())
	}
}

// A name close to the correct one is a typing mistake. Every request would
// fail on authentication, so the command stops and names the fix.
func TestCheckCredentialsRejectsANearMiss(t *testing.T) {
	for _, name := range nearMisses {
		t.Run(name, func(t *testing.T) {
			clearCredentials(t)
			t.Setenv(name, "sk-ant-test")

			err := checkCredentials(io.Discard)
			if err == nil {
				t.Fatalf("%s was accepted, want an error", name)
			}
			for _, want := range []string{name, "ANTHROPIC_API_KEY", "export"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("the message does not contain %q: %v", want, err)
				}
			}
		})
	}
}

// The correct name wins, even when a near miss is also set.
func TestCheckCredentialsPrefersTheCorrectName(t *testing.T) {
	clearCredentials(t)
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")
	t.Setenv("ANTHROPIC_KEY", "sk-ant-other")

	if err := checkCredentials(io.Discard); err != nil {
		t.Fatalf("error = %v, want none", err)
	}
}

// With no credential in the environment, the SDK can still find a profile, so
// the command writes a note and continues.
func TestCheckCredentialsNotesAnEmptyEnvironment(t *testing.T) {
	clearCredentials(t)

	var sb strings.Builder
	if err := checkCredentials(&sb); err != nil {
		t.Fatalf("error = %v, want none", err)
	}
	if !strings.Contains(sb.String(), "ANTHROPIC_API_KEY is not set") {
		t.Errorf("output = %q, want a note", sb.String())
	}
}

// An auth token is a valid credential, so it draws no note.
func TestCheckCredentialsAcceptsAnAuthToken(t *testing.T) {
	clearCredentials(t)
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "sk-ant-oat01-test")

	var sb strings.Builder
	if err := checkCredentials(&sb); err != nil {
		t.Fatalf("error = %v, want none", err)
	}
	if sb.String() != "" {
		t.Errorf("output = %q, want none", sb.String())
	}
}

func TestSplitList(t *testing.T) {
	cases := map[string]int{
		"a,b,c":                         3,
		" a , b ":                       2,
		"a":                             1,
		"":                              0,
		",,":                            0,
		"a,,b":                          2,
		"claude-opus-5,claude-sonnet-5": 2,
	}
	for in, want := range cases {
		if got := len(splitList(in)); got != want {
			t.Errorf("splitList(%q) gave %d items, want %d", in, got, want)
		}
	}
}

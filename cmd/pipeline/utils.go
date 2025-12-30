package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

func sortStrings(in []string) []string {
	out := append([]string{}, in...)
	sort.Strings(out)
	return out
}

func uniqueHeadlinesByURL(in []Headline) []Headline {
	seen := map[string]bool{}
	out := make([]Headline, 0, len(in))
	for _, h := range in {
		if h.URL == "" {
			continue
		}
		if seen[h.URL] {
			continue
		}
		seen[h.URL] = true
		out = append(out, h)
	}
	return out
}

func writeJSONToStdout(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeJSONFile(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func readJSONFile(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

func warnf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "WARN: "+format+"\n", args...)
}

func infof(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "INFO: "+format+"\n", args...)
}

func normalizeWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func uniqStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

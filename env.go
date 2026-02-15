package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type envPair struct {
	Key   string
	Value string
}

func parseEnvContent(content string) ([]envPair, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	out := make([]envPair, 0)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		idx := strings.IndexRune(line, '=')
		if idx <= 0 {
			return nil, fmt.Errorf("invalid .env line %d", lineNum)
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		if key == "" {
			return nil, fmt.Errorf("empty key at line %d", lineNum)
		}
		if unquoted, ok := unquoteEnv(value); ok {
			value = unquoted
		}
		out = append(out, envPair{Key: key, Value: value})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func unquoteEnv(value string) (string, bool) {
	if len(value) < 2 {
		return value, false
	}
	if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) || (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
		q := value[0]
		inner := value[1 : len(value)-1]
		if q == '\'' {
			return inner, true
		}
		replacer := strings.NewReplacer(
			`\n`, "\n",
			`\r`, "\r",
			`\t`, "\t",
			`\\`, `\`,
			`\"`, `\"`,
		)
		return replacer.Replace(inner), true
	}
	return value, false
}

func renderEnv(bundle *ProjectBundle) string {
	var b bytes.Buffer
	sorted := make([]Secret, len(bundle.Secrets))
	copy(sorted, bundle.Secrets)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Key < sorted[j].Key })
	for _, secret := range sorted {
		value := secret.Value
		if strings.ContainsAny(value, " \t\n\r#") {
			encoded, _ := json.Marshal(value)
			value = string(encoded)
		}
		_, _ = b.WriteString(secret.Key + "=" + value + "\n")
	}
	return b.String()
}

func renderProjectJSON(bundle *ProjectBundle) (string, error) {
	b, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

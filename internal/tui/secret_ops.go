package tui

import (
	"sort"
	"strings"
	"time"
)

var prefixGroups = map[string]string{
	"OPENAI_":      "API Keys",
	"ANTHROPIC_":   "API Keys",
	"STRIPE_":      "Payments",
	"SUPABASE_":    "Database",
	"DATABASE_":    "Database",
	"AWS_":         "AWS",
	"GITHUB_":      "GitHub",
	"NEXT_PUBLIC_": "Frontend",
	"REDIS_":       "Database",
	"POSTGRES_":    "Database",
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func detectGroup(key string) string {
	upper := strings.ToUpper(strings.TrimSpace(key))
	for prefix, group := range prefixGroups {
		if strings.HasPrefix(upper, prefix) {
			return group
		}
	}
	return "General"
}

func upsertSecret(bundle *ProjectBundle, key, value, group string) (created bool) {
	now := nowRFC3339()
	for i := range bundle.Secrets {
		if bundle.Secrets[i].Key == key {
			bundle.Secrets[i].Value = value
			if group != "" {
				bundle.Secrets[i].Group = group
			}
			bundle.Secrets[i].UpdatedAt = now
			return false
		}
	}
	if group == "" {
		group = detectGroup(key)
	}
	bundle.Secrets = append(bundle.Secrets, Secret{
		Key:       key,
		Value:     value,
		Group:     group,
		CreatedAt: now,
		UpdatedAt: now,
	})
	return true
}

func removeSecret(bundle *ProjectBundle, key string) bool {
	for i := range bundle.Secrets {
		if bundle.Secrets[i].Key == key {
			bundle.Secrets = append(bundle.Secrets[:i], bundle.Secrets[i+1:]...)
			return true
		}
	}
	return false
}

func getSecret(bundle *ProjectBundle, key string) (Secret, bool) {
	for _, secret := range bundle.Secrets {
		if secret.Key == key {
			return secret, true
		}
	}
	return Secret{}, false
}

func maskValue(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 6 {
		return strings.Repeat("*", len(value))
	}
	return value[:6] + strings.Repeat("*", len(value)-6)
}

func sortSecrets(secrets []Secret) {
	sort.Slice(secrets, func(i, j int) bool {
		if secrets[i].Group == secrets[j].Group {
			return secrets[i].Key < secrets[j].Key
		}
		return secrets[i].Group < secrets[j].Group
	})
}

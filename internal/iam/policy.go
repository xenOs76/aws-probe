package iam

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
)

// decodePolicyDocument URL-decodes an IAM policy document and unmarshals it to a
// generic JSON value. IAM returns documents URL-encoded per RFC 3986.
func decodePolicyDocument(encoded string) (any, error) {
	if encoded == "" {
		return nil, errors.New("empty policy document")
	}

	decoded, err := url.QueryUnescape(encoded)
	if err != nil {
		return nil, fmt.Errorf("url-decoding policy document: %w", err)
	}

	var doc any
	if err := json.Unmarshal([]byte(decoded), &doc); err != nil {
		return nil, fmt.Errorf("parsing policy document JSON: %w", err)
	}

	return doc, nil
}

// extractServices returns sorted unique AWS service prefixes from Action/NotAction
// values in a policy document (e.g. "s3:GetObject" → "s3", "*" → "*").
func extractServices(doc any) []string {
	seen := make(map[string]struct{})
	collectServices(doc, seen)

	services := make([]string, 0, len(seen))
	for s := range seen {
		services = append(services, s)
	}

	slices.Sort(services)

	return services
}

func collectServices(node any, seen map[string]struct{}) {
	switch v := node.(type) {
	case map[string]any:
		for key, val := range v {
			switch strings.ToLower(key) {
			case "action", "notaction":
				addActions(val, seen)
			default:
				collectServices(val, seen)
			}
		}
	case []any:
		for _, item := range v {
			collectServices(item, seen)
		}
	default:
		return
	}
}

func addActions(val any, seen map[string]struct{}) {
	switch a := val.(type) {
	case string:
		if svc := serviceFromAction(a); svc != "" {
			seen[svc] = struct{}{}
		}
	case []any:
		for _, item := range a {
			s, ok := item.(string)
			if !ok {
				continue
			}

			if svc := serviceFromAction(s); svc != "" {
				seen[svc] = struct{}{}
			}
		}
	default:
		return
	}
}

func serviceFromAction(action string) string {
	action = strings.TrimSpace(action)
	if action == "" {
		return ""
	}

	if action == "*" {
		return "*"
	}

	service, _, ok := strings.Cut(action, ":")
	if !ok || service == "" {
		return ""
	}

	return service
}

// formatPrincipal flattens a trust-policy Principal field for table output.
func formatPrincipal(principal any) string {
	if principal == nil {
		return ""
	}

	switch p := principal.(type) {
	case string:
		return p
	case map[string]any:
		parts := make([]string, 0, len(p))
		keys := make([]string, 0, len(p))

		for k := range p {
			keys = append(keys, k)
		}

		slices.Sort(keys)

		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s=%s", k, formatPrincipal(p[k])))
		}

		return strings.Join(parts, "; ")
	case []any:
		parts := make([]string, 0, len(p))
		for _, item := range p {
			parts = append(parts, formatPrincipal(item))
		}

		return strings.Join(parts, ", ")
	default:
		b, err := json.Marshal(p)
		if err != nil {
			return fmt.Sprintf("%v", p)
		}

		return string(b)
	}
}

// formatActions flattens Action/NotAction for table output.
func formatActions(actions any) string {
	if actions == nil {
		return ""
	}

	switch a := actions.(type) {
	case string:
		return a
	case []any:
		parts := make([]string, 0, len(a))
		for _, item := range a {
			parts = append(parts, fmt.Sprintf("%v", item))
		}

		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", a)
	}
}

// trustStatementRows flattens trust policy statements into table rows.
func trustStatementRows(doc any) [][]string {
	m, ok := doc.(map[string]any)
	if !ok {
		return nil
	}

	rawStatements, ok := m["Statement"]
	if !ok {
		return nil
	}

	var statements []any

	switch s := rawStatements.(type) {
	case []any:
		statements = s
	case map[string]any:
		statements = []any{s}
	default:
		return nil
	}

	rows := make([][]string, 0, len(statements))

	for _, stmt := range statements {
		sm, ok := stmt.(map[string]any)
		if !ok {
			continue
		}

		effect := ""
		if e, ok := sm["Effect"].(string); ok {
			effect = e
		}

		rows = append(rows, []string{
			effect,
			formatPrincipal(sm["Principal"]),
			formatActions(sm["Action"]),
			formatActions(sm["NotAction"]),
		})
	}

	return rows
}

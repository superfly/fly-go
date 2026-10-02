package fly

import (
	"encoding/json"
	"strings"
)

// redactedValue replaces credential material in logged request and response bodies.
const redactedValue = "[REDACTED]"

// sensitiveKeys are JSON object keys whose values carry credentials: app secret
// values, tokens, passwords, and the environment blocks that extension
// provisioning returns. Matching is case-insensitive. Keys are kept so
// debug logs still show which secrets or tokens a request touched.
var sensitiveKeys = map[string]bool{
	"value":         true,
	"values":        true,
	"secret":        true,
	"password":      true,
	"token":         true,
	"tokenheader":   true,
	"token_header":  true,
	"access_token":  true,
	"accesstoken":   true,
	"authorization": true,
	"environment":   true,
}

// redactJSON returns data with every string nested under a sensitive key
// replaced by redactedValue. Data that is not a JSON document is returned as is.
func redactJSON(data []byte) []byte {
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		return data
	}

	out, err := json.Marshal(redactValue(doc, false))
	if err != nil {
		return data
	}

	return out
}

func redactValue(v any, sensitive bool) any {
	switch v := v.(type) {
	case map[string]any:
		for k, child := range v {
			v[k] = redactValue(child, sensitive || sensitiveKeys[strings.ToLower(k)])
		}

		return v
	case []any:
		for i, child := range v {
			v[i] = redactValue(child, sensitive)
		}

		return v
	case string:
		if sensitive {
			return redactedValue
		}

		return v
	default:
		return v
	}
}

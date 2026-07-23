package views

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (r HostRule) AppliesToJob(job Job) bool {
	if !r.Enabled {
		return false
	}

	if !jobTypeMatches(r.AppliesTo, job.JobType()) {
		return false
	}

	meta := job.Metadata()

	if !pathMatchesAny(r.Match.Paths, meta.Path) {
		return false
	}

	if !methodMatches(r.Match.Methods, meta.Method) {
		return false
	}

	switch job.JobType() {
	case RequestJobType:
		req, ok := job.(*RequestJob)
		if !ok {
			return false
		}

		if !headerMatches(r.Match.Headers, req.Headers) {
			return false
		}
	case ResponseJobType:
		res, ok := job.(*ResponseJob)
		if !ok {
			return false
		}

		if !headerMatches(r.Match.Headers, res.Headers) {
			return false
		}
	default:
		return false
	}

	if r.Type != RuleTypeQuery && !queryParamsMatches(r.Match.QueryParams, meta.Query) {
		return false
	}

	return true
}

func jobTypeMatches(supported []JobType, actual JobType) bool {
	if len(supported) == 0 {
		return true
	}

	for _, jobType := range supported {
		if jobType == actual {
			return true
		}
	}

	return false
}

func pathMatchesAny(patterns []string, actual string) bool {
	if len(patterns) == 0 {
		return true
	}

	for _, pattern := range patterns {
		if pathMatches(pattern, actual) {
			return true
		}
	}

	return false
}

func pathMatches(pattern, actual string) bool {
	if pattern == "*" {
		return true
	}

	if pattern == actual {
		return true
	}

	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return actual == prefix || strings.HasPrefix(actual, prefix+"/")
	}

	if matchOpenAPIPath(pattern, actual) {
		return true
	}

	return false
}

func methodMatches(methods []string, actual string) bool {
	if len(methods) == 0 {
		return true
	}

	for _, method := range methods {
		if strings.EqualFold(method, actual) {
			return true
		}
	}

	return false
}

func headerMatches(headers map[string][]string, actual http.Header) bool {
	if len(headers) == 0 {
		return true
	}

	if _, ok := headers["*"]; ok {
		return len(actual) > 0
	}

	for key := range headers {
		if actual.Get(key) != "" {
			return true
		}
	}

	return false
}

func queryParamsMatches(expected map[string][]string, rawQuery string) bool {
	if len(expected) == 0 {
		return true
	}

	vals, err := url.ParseQuery(rawQuery)
	if err != nil {
		return false
	}

	if _, ok := expected["*"]; ok {
		return len(vals) > 0
	}

	for key := range expected {
		if _, ok := vals[key]; ok {
			return true
		}
	}

	return false
}

func (r HostRule) CheckHostRule(job Job, jobID, ruleID string) ([]Finding, error) {
	var findings []Finding

	if !r.AppliesToJob(job) {
		return nil, nil
	}

	switch r.Type {
	case RuleTypePath:
		findings = append(findings, r.EvaluatePath(job.Metadata(), jobID, ruleID)...)
	case RuleTypeHeader:
		switch job.JobType() {
		case RequestJobType:
			req, ok := job.(*RequestJob)
			if ok {
				findings = append(findings, r.EvaluateHeader(req.Headers, jobID, ruleID)...)
			}
		case ResponseJobType:
			res, ok := job.(*ResponseJob)
			if ok {
				findings = append(findings, r.EvaluateHeader(res.Headers, jobID, ruleID)...)
			}
		}
	case RuleTypeQuery:
		findings = append(findings, r.EvaluateQueryParams(job.Metadata(), jobID, ruleID)...)
	case RuleTypeBodyField:
		switch job.JobType() {
		case RequestJobType:
			req, ok := job.(*RequestJob)
			if ok {
				body, parseFindings := unmarshalJSONBody(req.Body, jobID, ruleID)
				if parseFindings != nil {
					return parseFindings, nil
				}

				findings = append(findings, r.EvaluateBody(body, jobID, ruleID)...)
			}
		case ResponseJobType:
			res, ok := job.(*ResponseJob)
			if ok {
				body, parseFindings := unmarshalJSONBody(res.Body, jobID, ruleID)
				if parseFindings != nil {
					return parseFindings, nil
				}

				findings = append(findings, r.EvaluateBody(body, jobID, ruleID)...)
			}
		}
	default:
		return nil, nil
	}

	return findings, nil
}

func unmarshalJSONBody(data []byte, jobID, ruleID string) (any, []Finding) {
	var body any

	if err := json.Unmarshal(data, &body); err != nil {
		return nil, []Finding{
			{
				ID:        uuid.NewString(),
				JobID:     jobID,
				RuleID:    ruleID,
				Title:     "JSON body parsing failed",
				Message:   fmt.Sprintf("Observer could not parse JSON body while evaluating custom rule %q: %v", ruleID, err),
				CreatedAt: time.Now().UTC(),
			},
		}
	}

	return body, nil
}

func (r HostRule) newCustomFinding(jobID, ruleID string) Finding {
	return Finding{
		ID:        uuid.NewString(),
		JobID:     jobID,
		RuleID:    ruleID,
		Title:     r.Finding.Title,
		Message:   r.Finding.Message,
		CreatedAt: time.Now().UTC(),
	}
}

func (r HostRule) EvaluatePath(meta Metadata, jobID, ruleID string) []Finding {
	if meta.Path == "" {
		return nil
	}

	for _, pattern := range r.Match.Patterns {
		re := pattern.Regex

		if pattern.Target != TargetTypePath {
			continue
		}

		if re == nil {
			continue
		}

		if re.MatchString(meta.Path) {
			return []Finding{
				r.newCustomFinding(jobID, ruleID),
			}
		}
	}

	return nil
}

func (r HostRule) EvaluateHeader(header http.Header, jobID, ruleID string) []Finding {
	if len(header) == 0 {
		return nil
	}

	if len(r.Match.Headers) != 0 && headerExactMatch(r.Match.Headers, header) {
		return []Finding{
			r.newCustomFinding(jobID, ruleID),
		}
	}

	if len(r.Match.Patterns) != 0 && headerPatternMatch(r.Match.Patterns, header) {
		return []Finding{
			r.newCustomFinding(jobID, ruleID),
		}
	}

	return nil
}

func headerExactMatch(expected map[string][]string, actual http.Header) bool {
	if len(expected) == 0 {
		return false
	}

	for name, expectedValues := range expected {
		if name == "*" {
			return len(actual) > 0
		}

		actualValues := actual.Values(name)
		if len(actualValues) == 0 {
			continue
		}

		// If no expected values are configured, treat header presence as a match.
		if len(expectedValues) == 0 {
			return true
		}

		for _, expectedValue := range expectedValues {
			for _, actualValue := range actualValues {
				if expectedValue == "*" {
					return true
				}

				if strings.EqualFold(
					strings.TrimSpace(actualValue),
					strings.TrimSpace(expectedValue),
				) {
					return true
				}
			}
		}
	}

	return false
}

func headerPatternMatch(patterns []RulePattern, actual http.Header) bool {
	for _, pattern := range patterns {
		if pattern.Target != TargetTypeHeader {
			continue
		}

		if pattern.Regex == nil {
			continue
		}

		if pattern.Name == "*" || pattern.Name == "" {
			for _, values := range actual {
				for _, value := range values {
					if pattern.Regex.MatchString(value) {
						return true
					}
				}
			}

			continue
		}

		values := actual.Values(pattern.Name)
		for _, value := range values {
			if pattern.Regex.MatchString(value) {
				return true
			}
		}
	}

	return false
}

func (r HostRule) EvaluateQueryParams(meta Metadata, jobID, ruleID string) []Finding {
	if meta.Query == "" {
		return nil
	}

	vals, err := url.ParseQuery(meta.Query)
	if err != nil {
		return []Finding{
			{
				ID:     uuid.NewString(),
				JobID:  jobID,
				RuleID: ruleID,
				Title:  "URL query parsing failed",
				Message: fmt.Sprintf(
					"Observer could not parse the query string for %s %s while evaluating custom rule %q.",
					meta.Method,
					meta.Path,
					ruleID,
				),
				CreatedAt: time.Now().UTC(),
			},
		}
	}

	if queryParamExactMatch(r.Match.QueryParams, vals) {
		return []Finding{
			r.newCustomFinding(jobID, ruleID),
		}
	}

	if queryParamPatternMatch(r.Match.Patterns, vals) {
		return []Finding{
			r.newCustomFinding(jobID, ruleID),
		}
	}

	return nil
}

func queryParamExactMatch(expected map[string][]string, actual url.Values) bool {
	if len(expected) == 0 {
		return false
	}

	for expectedName, expectedValues := range expected {
		if expectedName == "*" {
			return len(actual) > 0
		}

		actualValues, ok := actual[expectedName]
		if !ok {
			continue
		}

		// If no values are configured, treat presence of the query param as a match.
		if len(expectedValues) == 0 {
			return true
		}

		for _, expectedValue := range expectedValues {
			for _, actualValue := range actualValues {
				if expectedValue == "*" {
					return true
				}

				if actualValue == expectedValue {
					return true
				}
			}
		}
	}

	return false
}

func queryParamPatternMatch(patterns []RulePattern, actual url.Values) bool {
	for _, pattern := range patterns {
		if pattern.Target != TargetTypeQuery {
			continue
		}

		if pattern.Regex == nil {
			continue
		}

		if pattern.Name == "*" || pattern.Name == "" {
			for _, values := range actual {
				for _, value := range values {
					if pattern.Regex.MatchString(value) {
						return true
					}
				}
			}

			continue
		}

		values, ok := actual[pattern.Name]
		if !ok {
			continue
		}

		for _, value := range values {
			if pattern.Regex.MatchString(value) {
				return true
			}
		}
	}

	return false
}

func (r HostRule) EvaluateBody(body any, jobID, ruleID string) []Finding {
	if body == nil {
		return nil
	}

	if len(r.Match.Fields) != 0 && bodyExactMatch(r.Match.Fields, body) {
		return []Finding{
			r.newCustomFinding(jobID, ruleID),
		}
	}

	if len(r.Match.Patterns) != 0 && bodyPatternMatch(r.Match.Patterns, body) {
		return []Finding{
			r.newCustomFinding(jobID, ruleID),
		}
	}

	return nil
}

func bodyExactMatch(fields []string, body any) bool {
	if len(fields) == 0 {
		return false
	}

	for _, field := range fields {
		parts := splitFieldPath(field)
		if len(parts) == 0 {
			continue
		}

		if traverseBodyPath(body, parts) {
			return true
		}
	}

	return false
}

func splitFieldPath(field string) []string {
	rawParts := strings.Split(field, ".")
	parts := make([]string, 0, len(rawParts))

	for _, part := range rawParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		parts = append(parts, part)
	}

	return parts
}

func traverseBodyPath(val any, fields []string) bool {
	if len(fields) == 0 {
		return true
	}

	field := fields[0]

	switch v := val.(type) {
	case map[string]any:
		child, ok := v[field]
		if !ok {
			return false
		}

		return traverseBodyPath(child, fields[1:])

	case []any:
		for _, item := range v {
			if traverseBodyPath(item, fields) {
				return true
			}
		}

		return false

	default:
		return false
	}
}

func bodyPatternMatch(patterns []RulePattern, body any) bool {
	if len(patterns) == 0 {
		return false
	}

	for _, pattern := range patterns {
		if pattern.Target != TargetTypeField {
			continue
		}

		if pattern.Regex == nil {
			continue
		}

		name := strings.TrimSpace(pattern.Name)

		// No name or "*" means search all field names and primitive values.
		if name == "" || name == "*" {
			if matchBodyPattern(body, name, pattern.Regex) {
				return true
			}

			continue
		}

		// Dot path support, such as "tokens.accessToken".
		if strings.Contains(name, ".") {
			parts := splitFieldPath(name)
			if len(parts) == 0 {
				continue
			}

			value, ok := getBodyPathValue(body, parts)
			if !ok {
				continue
			}

			if primitiveValueMatchesRegex(value, pattern.Regex) {
				return true
			}

			continue
		}

		// Single field name support, such as "accessToken".
		if matchBodyPattern(body, name, pattern.Regex) {
			return true
		}
	}

	return false
}

func matchBodyPattern(val any, fieldName string, re *regexp.Regexp) bool {
	switch v := val.(type) {
	case map[string]any:
		for key, child := range v {
			if fieldName == "" || fieldName == "*" {
				if re.MatchString(key) {
					return true
				}

				if primitiveValueMatchesRegex(child, re) {
					return true
				}
			} else if key == fieldName {
				if primitiveValueMatchesRegex(child, re) {
					return true
				}
			}

			if matchBodyPattern(child, fieldName, re) {
				return true
			}
		}

	case []any:
		for _, item := range v {
			if matchBodyPattern(item, fieldName, re) {
				return true
			}
		}
	}

	return false
}

func primitiveValueMatchesRegex(val any, re *regexp.Regexp) bool {
	switch v := val.(type) {
	case string:
		return re.MatchString(v)
	case float64:
		return re.MatchString(strconv.FormatFloat(v, 'f', -1, 64))
	case bool:
		return re.MatchString(strconv.FormatBool(v))
	case nil:
		return re.MatchString("null")
	default:
		return false
	}
}

func getBodyPathValue(val any, fields []string) (any, bool) {
	if len(fields) == 0 {
		return val, true
	}

	field := fields[0]

	switch v := val.(type) {
	case map[string]any:
		child, ok := v[field]
		if !ok {
			return nil, false
		}

		return getBodyPathValue(child, fields[1:])

	case []any:
		for _, item := range v {
			value, ok := getBodyPathValue(item, fields)
			if ok {
				return value, true
			}
		}

		return nil, false

	default:
		return nil, false
	}
}

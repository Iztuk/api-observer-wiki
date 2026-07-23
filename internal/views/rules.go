package views

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type HostRulesDoc struct {
	Rules map[string]HostRule `json:"rules" yaml:"rules"`
}

type HostRule struct {
	Enabled     bool      `json:"enabled" yaml:"enabled"`
	AppliesTo   []JobType `json:"applies_to" yaml:"applies_to"`
	Type        RuleType  `json:"type" yaml:"type"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`

	Match   RuleMatch   `json:"match" yaml:"match"`
	Finding RuleFinding `json:"finding" yaml:"finding"`
}

type RuleType string

const (
	RuleTypePath      RuleType = "path"
	RuleTypeQuery     RuleType = "query"
	RuleTypeHeader    RuleType = "header"
	RuleTypeBodyField RuleType = "body_field"
)

type RuleMatch struct {
	Paths       []string            `json:"paths,omitempty" yaml:"paths,omitempty"`
	Methods     []string            `json:"methods,omitempty" yaml:"methods,omitempty"`
	Headers     map[string][]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	QueryParams map[string][]string `json:"query_params,omitempty" yaml:"query_params,omitempty"`
	Fields      []string            `json:"fields,omitempty" yaml:"fields,omitempty"`
	Patterns    []RulePattern       `json:"patterns,omitempty" yaml:"patterns,omitempty"`
}

type RulePattern struct {
	Target  TargetType     `json:"target" yaml:"target"`                 // query, header, path, field
	Name    string         `json:"name,omitempty" yaml:"name,omitempty"` // param/header/field name
	Pattern string         `json:"pattern" yaml:"pattern"`               // regex
	Regex   *regexp.Regexp `json:"-" yaml:"-"`
}

type TargetType string

const (
	TargetTypeQuery  TargetType = "query"
	TargetTypeHeader TargetType = "header"
	TargetTypePath   TargetType = "path"
	TargetTypeField  TargetType = "field"
)

type RuleFinding struct {
	Title   string `json:"title" yaml:"title"`
	Message string `json:"message" yaml:"message"`
}

func LoadRulesDocument(path string) (HostRulesDoc, error) {
	var doc HostRulesDoc

	data, err := os.ReadFile(path)
	if err != nil {
		return HostRulesDoc{}, err
	}

	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &doc); err != nil {
			return HostRulesDoc{}, fmt.Errorf("parse Host Rules JSON document: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return HostRulesDoc{}, fmt.Errorf("parse Host Rules YAML document: %w", err)
		}
	default:
		return HostRulesDoc{}, fmt.Errorf("unsupported Host Rules document type: %s", path)
	}

	if err := validateRulesDocument(doc); err != nil {
		return HostRulesDoc{}, err
	}

	// Precompile regex patterns after validation
	for ruleID, rule := range doc.Rules {
		for i := range rule.Match.Patterns {
			pattern := rule.Match.Patterns[i]

			re, err := regexp.Compile(pattern.Pattern)
			if err != nil {
				return HostRulesDoc{}, fmt.Errorf(
					"host rule %q has invalid regex pattern %q: %w",
					ruleID,
					pattern.Pattern,
					err,
				)
			}

			rule.Match.Patterns[i].Regex = re
		}

		doc.Rules[ruleID] = rule
	}

	return doc, nil
}

func validateRulesDocument(doc HostRulesDoc) error {
	if len(doc.Rules) == 0 {
		return nil
	}

	for ruleID, rule := range doc.Rules {
		if strings.TrimSpace(ruleID) == "" {
			return fmt.Errorf("host rule id cannot be empty")
		}

		// if len(rule.AppliesTo) == 0 {
		// 	return fmt.Errorf("host rule %q missing applies_to", ruleID)
		// }

		if rule.Type == "" {
			return fmt.Errorf("host rule %q missing type", ruleID)
		}

		if !isValidRuleType(rule.Type) {
			return fmt.Errorf("host rule %q has unsupported type %q", ruleID, rule.Type)
		}

		if strings.TrimSpace(rule.Finding.Title) == "" {
			return fmt.Errorf("host rule %q missing finding.title", ruleID)
		}

		if strings.TrimSpace(rule.Finding.Message) == "" {
			return fmt.Errorf("host rule %q missing finding.message", ruleID)
		}

		for _, pattern := range rule.Match.Patterns {
			if strings.TrimSpace(string(pattern.Target)) == "" {
				return fmt.Errorf("host rule %q has pattern with missing target", ruleID)
			}

			if !isValidTargetType(pattern.Target) {
				return fmt.Errorf(
					"host rule %q has unsupported pattern target %q",
					ruleID,
					pattern.Target,
				)
			}

			if !targetAllowedForRuleType(rule.Type, pattern.Target) {
				return fmt.Errorf(
					"host rule %q has pattern target %q that is not allowed for rule type %q",
					ruleID,
					pattern.Target,
					rule.Type,
				)
			}

			if strings.TrimSpace(pattern.Pattern) == "" {
				return fmt.Errorf("host rule %q has pattern with missing regex", ruleID)
			}

			if _, err := regexp.Compile(pattern.Pattern); err != nil {
				return fmt.Errorf("host rule %q has invalid regex pattern %q: %w", ruleID, pattern.Pattern, err)
			}
		}
	}

	return nil
}

func isValidTargetType(target TargetType) bool {
	switch target {
	case TargetTypeQuery,
		TargetTypeHeader,
		TargetTypePath,
		TargetTypeField:
		return true
	default:
		return false
	}
}

func isValidRuleType(ruleType RuleType) bool {
	switch ruleType {
	case RuleTypePath, RuleTypeQuery, RuleTypeHeader, RuleTypeBodyField:
		return true
	default:
		return false
	}
}

func targetAllowedForRuleType(ruleType RuleType, target TargetType) bool {
	switch ruleType {
	case RuleTypeQuery:
		return target == TargetTypeQuery

	case RuleTypeHeader:
		return target == TargetTypeHeader

	case RuleTypePath:
		return target == TargetTypePath

	case RuleTypeBodyField:
		return target == TargetTypeField

	default:
		return false
	}
}

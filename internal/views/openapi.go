package views

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type OpenAPIDoc struct {
	OpenAPI    string                     `json:"openapi" yaml:"openapi"`
	Info       OpenAPIInfo                `json:"info" yaml:"info"`
	Servers    []OpenAPIServer            `json:"servers,omitempty" yaml:"servers,omitempty"`
	Paths      map[string]OpenAPIPathItem `json:"paths" yaml:"paths"`
	Components *OpenAPIComponents         `json:"components,omitempty" yaml:"components,omitempty"`
}

type OpenAPIInfo struct {
	Title          string `json:"title" yaml:"title"`
	Version        string `json:"version" yaml:"version"`
	Description    string `json:"description,omitempty" yaml:"description,omitempty"`
	TermsOfService string `json:"termsOfService,omitempty" yaml:"termsOfService,omitempty"`
}

type OpenAPIServer struct {
	URL         string                    `json:"url" yaml:"url"`
	Description string                    `json:"description,omitempty" yaml:"description,omitempty"`
	Variables   map[string]ServerVariable `json:"variables,omitempty" yaml:"variables,omitempty"`
}

type ServerVariable struct {
	Enum        []string `json:"enum,omitempty" yaml:"enum,omitempty"`
	Default     string   `json:"default" yaml:"default"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
}

type OpenAPIPathItem struct {
	Summary     string `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	GET     *OpenAPIOperation `json:"get,omitempty" yaml:"get,omitempty"`
	POST    *OpenAPIOperation `json:"post,omitempty" yaml:"post,omitempty"`
	PUT     *OpenAPIOperation `json:"put,omitempty" yaml:"put,omitempty"`
	PATCH   *OpenAPIOperation `json:"patch,omitempty" yaml:"patch,omitempty"`
	DELETE  *OpenAPIOperation `json:"delete,omitempty" yaml:"delete,omitempty"`
	HEAD    *OpenAPIOperation `json:"head,omitempty" yaml:"head,omitempty"`
	OPTIONS *OpenAPIOperation `json:"options,omitempty" yaml:"options,omitempty"`
	TRACE   *OpenAPIOperation `json:"trace,omitempty" yaml:"trace,omitempty"`

	Parameters []OpenAPIParameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type OpenAPIOperation struct {
	Tags        []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	OperationID string   `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Summary     string   `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Deprecated  bool     `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`

	Parameters  []OpenAPIParameter         `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *OpenAPIRequestBody        `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   map[string]OpenAPIResponse `json:"responses" yaml:"responses"`

	Security []map[string][]string `json:"security,omitempty" yaml:"security,omitempty"`
}

type OpenAPIParameter struct {
	Name        string         `json:"name" yaml:"name"`
	In          string         `json:"in" yaml:"in"`
	Description string         `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool           `json:"required,omitempty" yaml:"required,omitempty"`
	Deprecated  bool           `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Schema      *OpenAPISchema `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example     any            `json:"example,omitempty" yaml:"example,omitempty"`
}

type OpenAPIRequestBody struct {
	Description string                      `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool                        `json:"required,omitempty" yaml:"required,omitempty"`
	Content     map[string]OpenAPIMediaType `json:"content,omitempty" yaml:"content,omitempty"`
}

type OpenAPIResponse struct {
	Description string                      `json:"description" yaml:"description"`
	Headers     map[string]OpenAPIHeader    `json:"headers,omitempty" yaml:"headers,omitempty"`
	Content     map[string]OpenAPIMediaType `json:"content,omitempty" yaml:"content,omitempty"`
}

type OpenAPIHeader struct {
	Description string         `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool           `json:"required,omitempty" yaml:"required,omitempty"`
	Deprecated  bool           `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Schema      *OpenAPISchema `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example     any            `json:"example,omitempty" yaml:"example,omitempty"`
}

type OpenAPIMediaType struct {
	Schema   *OpenAPISchema `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example  any            `json:"example,omitempty" yaml:"example,omitempty"`
	Examples map[string]any `json:"examples,omitempty" yaml:"examples,omitempty"`
	Encoding map[string]any `json:"encoding,omitempty" yaml:"encoding,omitempty"`
}

type OpenAPIComponents struct {
	Schemas         map[string]OpenAPISchema         `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	Responses       map[string]OpenAPIResponse       `json:"responses,omitempty" yaml:"responses,omitempty"`
	Parameters      map[string]OpenAPIParameter      `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBodies   map[string]OpenAPIRequestBody    `json:"requestBodies,omitempty" yaml:"requestBodies,omitempty"`
	Headers         map[string]OpenAPIHeader         `json:"headers,omitempty" yaml:"headers,omitempty"`
	SecuritySchemes map[string]OpenAPISecurityScheme `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
}

type OpenAPISecurityScheme struct {
	Type             string         `json:"type" yaml:"type"`
	Description      string         `json:"description,omitempty" yaml:"description,omitempty"`
	Name             string         `json:"name,omitempty" yaml:"name,omitempty"`
	In               string         `json:"in,omitempty" yaml:"in,omitempty"`
	Scheme           string         `json:"scheme,omitempty" yaml:"scheme,omitempty"`
	BearerFormat     string         `json:"bearerFormat,omitempty" yaml:"bearerFormat,omitempty"`
	Flows            map[string]any `json:"flows,omitempty" yaml:"flows,omitempty"`
	OpenIDConnectURL string         `json:"openIdConnectUrl,omitempty" yaml:"openIdConnectUrl,omitempty"`
}

type OpenAPISchema struct {
	Ref string `json:"$ref,omitempty" yaml:"$ref,omitempty"`

	Type        string `json:"type,omitempty" yaml:"type,omitempty"`
	Format      string `json:"format,omitempty" yaml:"format,omitempty"`
	Title       string `json:"title,omitempty" yaml:"title,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Default     any    `json:"default,omitempty" yaml:"default,omitempty"`
	Example     any    `json:"example,omitempty" yaml:"example,omitempty"`

	Properties map[string]OpenAPISchema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Required   []string                 `json:"required,omitempty" yaml:"required,omitempty"`
	Items      *OpenAPISchema           `json:"items,omitempty" yaml:"items,omitempty"`

	Enum []any `json:"enum,omitempty" yaml:"enum,omitempty"`

	Nullable  bool `json:"nullable,omitempty" yaml:"nullable,omitempty"`
	ReadOnly  bool `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
	WriteOnly bool `json:"writeOnly,omitempty" yaml:"writeOnly,omitempty"`

	MinLength *int     `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	MaxLength *int     `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	Minimum   *float64 `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	Maximum   *float64 `json:"maximum,omitempty" yaml:"maximum,omitempty"`

	Pattern string `json:"pattern,omitempty" yaml:"pattern,omitempty"`

	AdditionalProperties any `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`

	AllOf []OpenAPISchema `json:"allOf,omitempty" yaml:"allOf,omitempty"`
	OneOf []OpenAPISchema `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	AnyOf []OpenAPISchema `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
	Not   *OpenAPISchema  `json:"not,omitempty" yaml:"not,omitempty"`
}

func LoadOpenAPIDocument(path string) (OpenAPIDoc, error) {
	var doc OpenAPIDoc

	data, err := os.ReadFile(path)
	if err != nil {
		return OpenAPIDoc{}, err
	}

	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &doc); err != nil {
			return OpenAPIDoc{}, fmt.Errorf("parse OpenAPI JSON document: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return OpenAPIDoc{}, fmt.Errorf("parse OpenAPI YAML document: %w", err)
		}
	default:
		return OpenAPIDoc{}, fmt.Errorf("unsupported OpenAPI document type: %s", path)
	}

	return validateOpenAPIContractStructure(doc)
}

func validateOpenAPIContractStructure(doc OpenAPIDoc) (OpenAPIDoc, error) {
	if strings.TrimSpace(doc.OpenAPI) == "" {
		return OpenAPIDoc{}, fmt.Errorf("invalid OpenAPI contract: missing openapi field")
	}

	if strings.TrimSpace(doc.Info.Title) == "" {
		return OpenAPIDoc{}, fmt.Errorf("invalid OpenAPI contract: missing info.title")
	}

	if strings.TrimSpace(doc.Info.Version) == "" {
		return OpenAPIDoc{}, fmt.Errorf("invalid OpenAPI contract: missing info.version")
	}

	if len(doc.Paths) == 0 {
		return OpenAPIDoc{}, fmt.Errorf("invalid OpenAPI contract: missing paths")
	}

	for path, pathItem := range doc.Paths {
		if strings.TrimSpace(path) == "" {
			return OpenAPIDoc{}, fmt.Errorf("invalid OpenAPI contract: empty path")
		}

		if !strings.HasPrefix(path, "/") {
			return OpenAPIDoc{}, fmt.Errorf("invalid OpenAPI contract: path %q must start with '/'", path)
		}

		if !pathItem.hasOperation() {
			return OpenAPIDoc{}, fmt.Errorf("invalid OpenAPI contract: path %q has no operations", path)
		}
	}

	return doc, nil
}

func (p OpenAPIPathItem) hasOperation() bool {
	return p.GET != nil ||
		p.POST != nil ||
		p.PUT != nil ||
		p.PATCH != nil ||
		p.DELETE != nil ||
		p.HEAD != nil ||
		p.OPTIONS != nil ||
		p.TRACE != nil
}

func (d OpenAPIDoc) FindOpenAPIOperation(method, path string) (*OpenAPIOperation, bool) {
	if pathItem, ok := d.Paths[path]; ok {
		op := pathItem.OperationForMethod(method)
		if op != nil {
			return op, true
		}
	}

	type candidate struct {
		path string
		item OpenAPIPathItem
	}

	candidates := make([]candidate, 0, len(d.Paths))

	for contractPath, pathItem := range d.Paths {
		if !matchOpenAPIPath(contractPath, path) {
			continue
		}

		candidates = append(candidates, candidate{
			path: contractPath,
			item: pathItem,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return pathSpecificity(candidates[i].path) > pathSpecificity(candidates[j].path)
	})

	for _, c := range candidates {
		op := c.item.OperationForMethod(method)
		if op != nil {
			return op, true
		}
	}

	return nil, false
}

func pathSpecificity(path string) int {
	score := 0

	for _, segment := range splitPath(path) {
		if !isPathParam(segment) {
			score++
		}
	}

	return score
}

func (p OpenAPIPathItem) OperationForMethod(method string) *OpenAPIOperation {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case http.MethodGet:
		return p.GET

	case http.MethodPost:
		return p.POST

	case http.MethodPut:
		return p.PUT

	case http.MethodPatch:
		return p.PATCH

	case http.MethodDelete:
		return p.DELETE

	case http.MethodHead:
		return p.HEAD

	case http.MethodOptions:
		return p.OPTIONS

	case http.MethodTrace:
		return p.TRACE

	default:
		return nil
	}
}

func matchOpenAPIPath(contractPath, requestPath string) bool {
	contractParts := splitPath(contractPath)
	requestParts := splitPath(requestPath)

	if len(contractParts) != len(requestParts) {
		return false
	}

	for i := range contractParts {
		contractPart := contractParts[i]
		requestPart := requestParts[i]

		if isPathParam(contractPart) {
			continue
		}

		if contractPart != requestPart {
			return false
		}
	}

	return true
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return []string{}
	}

	return strings.Split(path, "/")
}

func isPathParam(segment string) bool {
	return strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") && len(segment) > 2
}

func (d OpenAPIDoc) ResolveSchemaRef(ref string) (*OpenAPISchema, bool) {
	const prefix = "#/components/schemas/"

	if !strings.HasPrefix(ref, prefix) {
		return nil, false
	}

	if d.Components == nil {
		return nil, false
	}

	name := strings.TrimPrefix(ref, prefix)

	schema, ok := d.Components.Schemas[name]
	if !ok {
		return nil, false
	}

	return &schema, true
}

package views

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

type UpstreamFailureRule struct{}

func (r UpstreamFailureRule) ID() RuleID {
	return RuleProxyUpstreamFailure
}

func (r UpstreamFailureRule) Title() string {
	return "Upstream request failed"
}

func (r UpstreamFailureRule) AppliesTo() []JobType {
	return []JobType{FailureJobType}
}

func (r UpstreamFailureRule) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	failureJob, ok := job.(*FailureJob)
	if !ok {
		return nil, nil
	}

	if failureJob.Meta.Status == 504 {
		return nil, nil
	}

	return []Finding{
		{
			ID:        uuid.NewString(),
			JobID:     jobID,
			RuleID:    string(r.ID()),
			Title:     r.Title(),
			Message:   failureJob.Error,
			CreatedAt: time.Now().UTC(),
		}}, nil
}

type UpstreamTimeoutRule struct{}

func (r UpstreamTimeoutRule) ID() RuleID {
	return RuleProxyUpstreamTimeout
}

func (r UpstreamTimeoutRule) Title() string {
	return "Upstream request timed out"
}

func (r UpstreamTimeoutRule) AppliesTo() []JobType {
	return []JobType{FailureJobType}
}

func (r UpstreamTimeoutRule) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	failureJob, ok := job.(*FailureJob)
	if !ok {
		return nil, nil
	}

	if failureJob.Meta.Status != 504 {
		return nil, nil
	}

	return []Finding{
		{
			ID:        uuid.NewString(),
			JobID:     jobID,
			RuleID:    string(r.ID()),
			Title:     r.Title(),
			Message:   "The upstream service did not respond before the configured timeout.",
			CreatedAt: time.Now().UTC(),
		},
	}, nil
}

type RequestPathDoesNotExistRule struct{}

func (r RequestPathDoesNotExistRule) ID() RuleID {
	return RuleRequestPathDoesNotExist
}

func (r RequestPathDoesNotExistRule) Title() string {
	return "Request path does not exist"
}

func (r RequestPathDoesNotExistRule) AppliesTo() []JobType {
	return []JobType{RequestJobType}
}

func (r RequestPathDoesNotExistRule) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	requestJob, ok := job.(*RequestJob)
	if !ok {
		return nil, nil
	}

	_, found := ctx.Contracts.FindPathItem(
		requestJob.Meta.Host,
		requestJob.Meta.Path,
	)

	if found {
		return nil, nil
	}

	return []Finding{
		{
			ID:        uuid.NewString(),
			JobID:     jobID,
			RuleID:    string(r.ID()),
			Title:     r.Title(),
			Message:   fmt.Sprintf("Request path %q is not defined in the API contract.", requestJob.Meta.Path),
			CreatedAt: time.Now().UTC(),
		},
	}, nil
}

type RequestMethodNotAllowedRule struct{}

func (r RequestMethodNotAllowedRule) ID() RuleID {
	return RuleRequestMethodNotAllowed
}

func (r RequestMethodNotAllowedRule) Title() string {
	return "Request method not allowed"
}

func (r RequestMethodNotAllowedRule) AppliesTo() []JobType {
	return []JobType{RequestJobType}
}

func (r RequestMethodNotAllowedRule) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	requestJob, ok := job.(*RequestJob)
	if !ok {
		return nil, nil
	}

	_, found := ctx.Contracts.FindMethod(
		requestJob.Meta.Host,
		requestJob.Meta.Method,
		requestJob.Meta.Path,
	)

	if found {
		return nil, nil
	}

	return []Finding{
		{
			ID:        uuid.NewString(),
			JobID:     jobID,
			RuleID:    string(r.ID()),
			Title:     r.Title(),
			Message:   fmt.Sprintf("Request method %q is not defined in the API contract.", requestJob.Meta.Method),
			CreatedAt: time.Now().UTC(),
		},
	}, nil
}

type RequestContentTypeNotAllowed struct{}

func (r RequestContentTypeNotAllowed) ID() RuleID {
	return RuleRequestContentTypeNotAllowed
}

func (r RequestContentTypeNotAllowed) Title() string {
	return "Request content type not allowed"
}

func (r RequestContentTypeNotAllowed) AppliesTo() []JobType {
	return []JobType{RequestJobType}
}

func (r RequestContentTypeNotAllowed) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	requestJob, ok := job.(*RequestJob)
	if !ok {
		return nil, nil
	}

	contentType := requestJob.Headers.Get("Content-Type")

	_, applies, found := ctx.Contracts.FindRequestContentType(
		requestJob.Meta.Host,
		requestJob.Meta.Method,
		requestJob.Meta.Path,
		contentType,
	)

	if (applies && found) || !applies {
		return nil, nil
	}

	return []Finding{
		{
			ID:     uuid.NewString(),
			JobID:  jobID,
			RuleID: string(r.ID()),
			Title:  r.Title(),
			Message: fmt.Sprintf(
				"Request content type %q is not allowed for %s %s according to the API contract.",
				requestJob.Headers.Get("Content-Type"),
				requestJob.Meta.Method,
				requestJob.Meta.Path,
			),
			CreatedAt: time.Now().UTC(),
		},
	}, nil
}

type RequestBodyMissing struct{}

func (r RequestBodyMissing) ID() RuleID {
	return RuleRequestBodyMissing
}

func (r RequestBodyMissing) Title() string {
	return "Request body missing"
}

func (r RequestBodyMissing) AppliesTo() []JobType {
	return []JobType{RequestJobType}
}

func (r RequestBodyMissing) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	requestJob, ok := job.(*RequestJob)
	if !ok {
		return nil, nil
	}

	body, found := ctx.Contracts.FindRequestBody(
		requestJob.Meta.Host,
		requestJob.Meta.Method,
		requestJob.Meta.Path,
	)

	if !found {
		return nil, nil
	}

	if body != nil && body.Required && len(requestJob.Body) == 0 {
		return []Finding{
			{
				ID:     uuid.NewString(),
				JobID:  jobID,
				RuleID: string(r.ID()),
				Title:  r.Title(),
				Message: fmt.Sprintf(
					"Request body is required for %s %s according to the API contract, but the request body was empty.",
					requestJob.Meta.Method,
					requestJob.Meta.Path,
				),
				CreatedAt: time.Now().UTC(),
			},
		}, nil
	}

	return nil, nil
}

type RequestBodyNotAllowed struct{}

func (r RequestBodyNotAllowed) ID() RuleID {
	return RuleRequestBodyNotAllowed
}

func (r RequestBodyNotAllowed) Title() string {
	return "Request body not allowed"
}

func (r RequestBodyNotAllowed) AppliesTo() []JobType {
	return []JobType{RequestJobType}
}

func (r RequestBodyNotAllowed) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	requestJob, ok := job.(*RequestJob)
	if !ok {
		return nil, nil
	}

	body, found := ctx.Contracts.FindRequestBody(
		requestJob.Meta.Host,
		requestJob.Meta.Method,
		requestJob.Meta.Path,
	)

	if !found {
		return nil, nil
	}

	if body != nil {
		return nil, nil
	}

	if len(requestJob.Body) == 0 {
		return nil, nil
	}

	return []Finding{
		{
			ID:     uuid.NewString(),
			JobID:  jobID,
			RuleID: string(r.ID()),
			Title:  r.Title(),
			Message: fmt.Sprintf(
				"Request body not allowed for %s %s according to the API contract, but the request body was not empty.",
				requestJob.Meta.Method,
				requestJob.Meta.Path,
			),
			CreatedAt: time.Now().UTC(),
		},
	}, nil
}

type RequestBodyInvalidFormat struct{}

func (r RequestBodyInvalidFormat) ID() RuleID {
	return RuleRequestInvalidBodyFormat
}

func (r RequestBodyInvalidFormat) Title() string {
	return "Request body has invalid format"
}

func (r RequestBodyInvalidFormat) AppliesTo() []JobType {
	return []JobType{RequestJobType}
}

func (r RequestBodyInvalidFormat) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	requestJob, ok := job.(*RequestJob)
	if !ok {
		return nil, nil
	}

	body, found := ctx.Contracts.FindRequestBody(
		requestJob.Meta.Host,
		requestJob.Meta.Method,
		requestJob.Meta.Path,
	)

	if !found {
		return nil, nil
	}

	if body == nil {
		return nil, nil
	}

	if len(requestJob.Body) == 0 {
		return nil, nil
	}

	contentType := requestJob.Headers.Get("Content-Type")
	if contentType == "" {
		return nil, nil
	}

	ct, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return []Finding{
			{
				ID:     uuid.NewString(),
				JobID:  jobID,
				RuleID: string(r.ID()),
				Title:  r.Title(),
				Message: fmt.Sprintf(
					"Request Content-Type header %q could not be parsed for %s %s.",
					contentType,
					requestJob.Meta.Method,
					requestJob.Meta.Path,
				),
				CreatedAt: time.Now().UTC(),
			},
		}, nil
	}

	ct = strings.ToLower(ct)

	if !mediaTypeAllowed(body.Content, ct) {
		return nil, nil
	}

	if err := validateBodyForMediaType(ct, params, requestJob.Body); err != nil {
		return []Finding{
			{
				ID:     uuid.NewString(),
				JobID:  jobID,
				RuleID: string(r.ID()),
				Title:  r.Title(),
				Message: fmt.Sprintf(
					"Request body is not valid for Content-Type %q on %s %s: %v.",
					ct,
					requestJob.Meta.Method,
					requestJob.Meta.Path,
					err,
				),
				CreatedAt: time.Now().UTC(),
			},
		}, nil
	}

	return nil, nil
}

func mediaTypeAllowed(content map[string]OpenAPIMediaType, ct string) bool {
	for allowed := range content {
		allowed = strings.ToLower(allowed)

		if allowed == ct {
			return true
		}
	}

	return false
}

func validateBodyForMediaType(ct string, params map[string]string, body []byte) error {
	switch {
	case isJSONMediaType(ct):
		if !json.Valid(body) {
			return fmt.Errorf("body is not valid JSON")
		}

	case ct == "application/x-www-form-urlencoded":
		if _, err := url.ParseQuery(string(body)); err != nil {
			return fmt.Errorf("body is not valid form-urlencoded data: %w", err)
		}

	case isXMLMediaType(ct):
		if err := validateXML(body); err != nil {
			return fmt.Errorf("body is not valid XML: %w", err)
		}

	case ct == "multipart/form-data":
		boundary := params["boundary"]
		if boundary == "" {
			return fmt.Errorf("multipart/form-data is missing boundary parameter")
		}

		if err := validateMultipart(body, boundary); err != nil {
			return fmt.Errorf("body is not valid multipart/form-data: %w", err)
		}

	default:
		return nil
	}

	return nil
}

func isJSONMediaType(ct string) bool {
	return ct == "application/json" || strings.HasSuffix(ct, "+json")
}

func isXMLMediaType(ct string) bool {
	return ct == "application/xml" ||
		ct == "text/xml" ||
		strings.HasSuffix(ct, "+xml")
}

func validateXML(body []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(body))

	for {
		_, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return err
		}
	}
}

func validateMultipart(body []byte, boundary string) error {
	reader := multipart.NewReader(bytes.NewReader(body), boundary)

	const maxMemory = 10 << 20 // 10 MB

	form, err := reader.ReadForm(maxMemory)
	if err != nil {
		return err
	}

	if form != nil {
		_ = form.RemoveAll()
	}

	return nil
}

type RequestBodySchemaInvalid struct{}

func (r RequestBodySchemaInvalid) ID() RuleID {
	return RuleRequestBodySchemaInvalid
}

func (r RequestBodySchemaInvalid) Title() string {
	return "Request body schema invalid"
}

func (r RequestBodySchemaInvalid) AppliesTo() []JobType {
	return []JobType{RequestJobType}
}

func (r RequestBodySchemaInvalid) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	requestJob, ok := job.(*RequestJob)
	if !ok {
		return nil, nil
	}

	body, found := ctx.Contracts.FindRequestBody(
		requestJob.Meta.Host,
		requestJob.Meta.Method,
		requestJob.Meta.Path,
	)

	// Operation could not be resolved. Let path/method rules handle it.
	if !found {
		return nil, nil
	}

	// Operation does not define a requestBody.
	if body == nil {
		return nil, nil
	}

	// Empty body is handled by request.body_missing.
	if len(requestJob.Body) == 0 {
		return nil, nil
	}

	contentType := requestJob.Headers.Get("Content-Type")
	if contentType == "" {
		return nil, nil
	}

	ct, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, nil
	}

	ct = strings.ToLower(ct)

	// Content-Type not allowed is handled by request.content_type_not_allowed.
	if !mediaTypeAllowed(body.Content, ct) {
		return nil, nil
	}

	// Initial schema validation only supports JSON bodies.
	if !isJSONMediaType(ct) {
		return nil, nil
	}

	// Invalid JSON syntax is handled by request.invalid_body_format.
	if err := validateBodyForMediaType(ct, params, requestJob.Body); err != nil {
		return nil, nil
	}

	media, ok := findMatchingMediaType(body.Content, ct)
	if !ok {
		return nil, nil
	}

	if media.Schema == nil {
		return nil, nil
	}

	var requestBodyJSON any
	if err := json.Unmarshal(requestJob.Body, &requestBodyJSON); err != nil {
		return nil, nil
	}

	errs := validateJSONBodySchema(
		requestBodyJSON,
		*media.Schema,
		func(ref string) (*OpenAPISchema, bool) {
			return ctx.Contracts.ResolveSchemaRef(requestJob.Meta.Host, ref)
		},
		"$",
		map[string]bool{},
	)

	if len(errs) == 0 {
		return nil, nil
	}

	return []Finding{
		{
			ID:     uuid.NewString(),
			JobID:  jobID,
			RuleID: string(r.ID()),
			Title:  r.Title(),
			Message: fmt.Sprintf(
				"Request body does not match the OpenAPI schema for %s %s: %s.",
				requestJob.Meta.Method,
				requestJob.Meta.Path,
				joinSchemaErrors(errs),
			),
			CreatedAt: time.Now().UTC(),
		},
	}, nil
}
func validateJSONBodySchema(
	value any,
	schema OpenAPISchema,
	resolveRef func(ref string) (*OpenAPISchema, bool),
	path string,
	seenRefs map[string]bool,
) []error {
	var errs []error

	if schema.Ref != "" {
		if seenRefs[schema.Ref] {
			return []error{
				fmt.Errorf("%s has circular schema reference %q", path, schema.Ref),
			}
		}

		resolved, ok := resolveRef(schema.Ref)
		if !ok {
			return []error{
				fmt.Errorf("%s references unknown schema %q", path, schema.Ref),
			}
		}

		seenRefs[schema.Ref] = true
		errs = append(errs, validateJSONBodySchema(value, *resolved, resolveRef, path, seenRefs)...)
		delete(seenRefs, schema.Ref)

		return errs
	}

	switch schema.Type {
	case "object":
		obj, ok := value.(map[string]any)
		if !ok {
			return []error{
				fmt.Errorf("%s expected object but got %s", path, jsonTypeName(value)),
			}
		}

		required := make(map[string]bool, len(schema.Required))
		for _, field := range schema.Required {
			required[field] = true
		}

		for field := range required {
			if _, ok := obj[field]; !ok {
				errs = append(errs, fmt.Errorf("%s missing required field %q", path, field))
			}
		}

		for name, propSchema := range schema.Properties {
			propValue, exists := obj[name]
			if !exists {
				continue
			}

			childPath := path + "." + name
			errs = append(errs, validateJSONBodySchema(
				propValue,
				propSchema,
				resolveRef,
				childPath,
				seenRefs,
			)...)
		}

	case "array":
		arr, ok := value.([]any)
		if !ok {
			return []error{
				fmt.Errorf("%s expected array but got %s", path, jsonTypeName(value)),
			}
		}

		if schema.Items == nil {
			return errs
		}

		for i, item := range arr {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			errs = append(errs, validateJSONBodySchema(
				item,
				*schema.Items,
				resolveRef,
				itemPath,
				seenRefs,
			)...)
		}

	case "string":
		if _, ok := value.(string); !ok {
			errs = append(errs, fmt.Errorf("%s expected string but got %s", path, jsonTypeName(value)))
		}

	case "integer":
		number, ok := value.(float64)
		if !ok {
			errs = append(errs, fmt.Errorf("%s expected integer but got %s", path, jsonTypeName(value)))
			break
		}

		if number != float64(int64(number)) {
			errs = append(errs, fmt.Errorf("%s expected integer but got number", path))
		}

	case "number":
		if _, ok := value.(float64); !ok {
			errs = append(errs, fmt.Errorf("%s expected number but got %s", path, jsonTypeName(value)))
		}

	case "boolean":
		if _, ok := value.(bool); !ok {
			errs = append(errs, fmt.Errorf("%s expected boolean but got %s", path, jsonTypeName(value)))
		}

	case "":
		// Some OpenAPI schemas omit type and only use properties/items/$ref.
		// For MVP, you can skip, or infer object if properties exist.
		if len(schema.Properties) > 0 {
			schema.Type = "object"
			errs = append(errs, validateJSONBodySchema(value, schema, resolveRef, path, seenRefs)...)
		}

	default:
		// Unsupported schema type. Do not fail the request for MVP.
		return errs
	}

	return errs
}

func findMatchingMediaType(content map[string]OpenAPIMediaType, ct string) (*OpenAPIMediaType, bool) {
	for allowed, media := range content {
		if strings.EqualFold(allowed, ct) {
			return &media, true
		}
	}

	return nil, false
}

func jsonTypeName(value any) string {
	switch value.(type) {
	case nil:
		return "null"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	default:
		return fmt.Sprintf("%T", value)
	}
}

func joinSchemaErrors(errs []error) string {
	if len(errs) == 0 {
		return ""
	}

	limit := len(errs)
	if limit > 3 {
		limit = 3
	}

	parts := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		parts = append(parts, errs[i].Error())
	}

	if len(errs) > limit {
		parts = append(parts, fmt.Sprintf("and %d more", len(errs)-limit))
	}

	return strings.Join(parts, "; ")
}

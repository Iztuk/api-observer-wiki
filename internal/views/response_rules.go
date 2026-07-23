package views

import (
	"encoding/json"
	"fmt"
	"mime"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ResponseStatusCodeRule struct{}

func (r ResponseStatusCodeRule) ID() RuleID {
	return RuleResponseStatusCodeNotDefined
}

func (r ResponseStatusCodeRule) Title() string {
	return "Response status code not defined"
}

func (r ResponseStatusCodeRule) AppliesTo() []JobType {
	return []JobType{ResponseJobType}
}

func (r ResponseStatusCodeRule) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	responseJob, ok := job.(*ResponseJob)
	if !ok {
		return nil, nil
	}

	op, found := ctx.Contracts.FindMethod(
		responseJob.Meta.Host,
		responseJob.Meta.Method,
		responseJob.Meta.Path,
	)
	if !found {
		return nil, nil
	}

	status := strconv.Itoa(responseJob.Meta.Status)
	_, found = op.Responses[status]
	if found {
		return nil, nil
	}

	return []Finding{
		{
			ID:     uuid.NewString(),
			JobID:  jobID,
			RuleID: string(r.ID()),
			Title:  r.Title(),
			Message: fmt.Sprintf(
				"Response status code %d for %s %s is not defined in the API contract.",
				responseJob.Meta.Status,
				responseJob.Meta.Method,
				responseJob.Meta.Path,
			),
			CreatedAt: time.Now().UTC(),
		},
	}, nil
}

type ResponseContentTypeNotAllowed struct{}

func (r ResponseContentTypeNotAllowed) ID() RuleID {
	return RuleResponseContentTypeNotAllowed
}

func (r ResponseContentTypeNotAllowed) Title() string {
	return "Response content type not allowed"
}

func (r ResponseContentTypeNotAllowed) AppliesTo() []JobType {
	return []JobType{ResponseJobType}
}

func (r ResponseContentTypeNotAllowed) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	responseJob, ok := job.(*ResponseJob)
	if !ok {
		return nil, nil
	}

	ct := responseJob.Headers.Get("Content-Type")
	status := strconv.Itoa(responseJob.Meta.Status)

	_, applies, found := ctx.Contracts.FindResponseContentType(
		responseJob.Meta.Host,
		responseJob.Meta.Method,
		responseJob.Meta.Path,
		status,
		ct,
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
				"Response content type %q is not allowed for %s %s according to the API contract.",
				responseJob.Headers.Get("Content-Type"),
				responseJob.Meta.Method,
				responseJob.Meta.Path,
			),
			CreatedAt: time.Now().UTC(),
		},
	}, nil
}

type ResponseBodyMissing struct{}

func (r ResponseBodyMissing) ID() RuleID {
	return RuleResponseBodyMissing
}

func (r ResponseBodyMissing) Title() string {
	return "Response body missing"
}

func (r ResponseBodyMissing) AppliesTo() []JobType {
	return []JobType{ResponseJobType}
}

func (r ResponseBodyMissing) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	responseJob, ok := job.(*ResponseJob)
	if !ok {
		return nil, nil
	}

	status := strconv.Itoa(responseJob.Meta.Status)
	body, found := ctx.Contracts.FindResponseBody(
		responseJob.Meta.Host,
		responseJob.Meta.Method,
		responseJob.Meta.Path,
		status,
	)

	if !found {
		return nil, nil
	}

	if len(body.Content) == 0 {
		return nil, nil
	}

	if len(responseJob.Body) == 0 {
		return []Finding{
			{
				ID:     uuid.NewString(),
				JobID:  jobID,
				RuleID: string(r.ID()),
				Title:  r.Title(),
				Message: fmt.Sprintf(
					"Response body is required for %s %s according to the API contract, but the response body was empty.",
					responseJob.Meta.Method,
					responseJob.Meta.Path,
				),
				CreatedAt: time.Now().UTC(),
			},
		}, nil
	}

	return nil, nil
}

type ResponseBodyNotAllowed struct{}

func (r ResponseBodyNotAllowed) ID() RuleID {
	return RuleResponseBodyNotAllowed
}

func (r ResponseBodyNotAllowed) Title() string {
	return "Response body not allowed"
}

func (r ResponseBodyNotAllowed) AppliesTo() []JobType {
	return []JobType{ResponseJobType}
}

func (r ResponseBodyNotAllowed) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	responseJob, ok := job.(*ResponseJob)
	if !ok {
		return nil, nil
	}

	status := strconv.Itoa(responseJob.Meta.Status)
	body, found := ctx.Contracts.FindResponseBody(
		responseJob.Meta.Host,
		responseJob.Meta.Method,
		responseJob.Meta.Path,
		status,
	)

	if !found {
		return nil, nil
	}

	if len(body.Content) != 0 {
		return nil, nil
	}

	if len(responseJob.Body) == 0 {
		return nil, nil
	}

	return []Finding{
		{
			ID:     uuid.NewString(),
			JobID:  jobID,
			RuleID: string(r.ID()),
			Title:  r.Title(),
			Message: fmt.Sprintf(
				"Response body not allowed for %s %s according to the API contract, but the response body was not empty.",
				responseJob.Meta.Method,
				responseJob.Meta.Path,
			),
			CreatedAt: time.Now().UTC(),
		},
	}, nil
}

type ResponseBodyInvalidFormat struct{}

func (r ResponseBodyInvalidFormat) ID() RuleID {
	return RuleResponseInvalidBodyFormat
}

func (r ResponseBodyInvalidFormat) Title() string {
	return "Response body has invalid format"
}

func (r ResponseBodyInvalidFormat) AppliesTo() []JobType {
	return []JobType{ResponseJobType}
}

func (r ResponseBodyInvalidFormat) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	res, ok := job.(*ResponseJob)
	if !ok {
		return nil, nil
	}

	status := strconv.Itoa(res.Meta.Status)
	body, found := ctx.Contracts.FindResponseBody(
		res.Meta.Host,
		res.Meta.Method,
		res.Meta.Path,
		status,
	)
	if !found {
		return nil, nil
	}

	if body == nil {
		return nil, nil
	}

	if len(res.Body) == 0 {
		return nil, nil
	}

	contentType := res.Headers.Get("Content-Type")
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
					"Response Content-Type header %q could not be parsed for %s %s.",
					contentType,
					res.Meta.Method,
					res.Meta.Path,
				),
				CreatedAt: time.Now().UTC(),
			},
		}, nil
	}

	ct = strings.ToLower(ct)

	if !mediaTypeAllowed(body.Content, ct) {
		return nil, nil
	}

	if err := validateBodyForMediaType(ct, params, res.Body); err != nil {
		return []Finding{
			{
				ID:     uuid.NewString(),
				JobID:  jobID,
				RuleID: string(r.ID()),
				Title:  r.Title(),
				Message: fmt.Sprintf(
					"Response body is not valid for Content-Type %q on %s %s: %v.",
					ct,
					res.Meta.Method,
					res.Meta.Path,
					err,
				),
				CreatedAt: time.Now().UTC(),
			},
		}, nil
	}

	return nil, nil
}

type ResponseBodySchemaInvalid struct{}

func (r ResponseBodySchemaInvalid) ID() RuleID {
	return RuleResponseBodySchemaInvalid
}

func (r ResponseBodySchemaInvalid) Title() string {
	return "Response body schema invalid"
}

func (r ResponseBodySchemaInvalid) AppliesTo() []JobType {
	return []JobType{ResponseJobType}
}

func (r ResponseBodySchemaInvalid) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	res, ok := job.(*ResponseJob)
	if !ok {
		return nil, nil
	}

	status := strconv.Itoa(res.Meta.Status)

	body, found := ctx.Contracts.FindResponseBody(
		res.Meta.Host,
		res.Meta.Method,
		res.Meta.Path,
		status,
	)

	// Operation or response status could not be resolved.
	// Let path/method/status-code rules handle it.
	if !found {
		return nil, nil
	}

	// Response does not define a response body/content.
	if body == nil {
		return nil, nil
	}

	// Empty body is handled by response.body_missing.
	if len(res.Body) == 0 {
		return nil, nil
	}

	contentType := res.Headers.Get("Content-Type")
	if contentType == "" {
		return nil, nil
	}

	ct, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, nil
	}

	ct = strings.ToLower(ct)

	// Content-Type not allowed is handled by response.content_type_not_allowed.
	if !mediaTypeAllowed(body.Content, ct) {
		return nil, nil
	}

	// Initial schema validation only supports JSON bodies.
	if !isJSONMediaType(ct) {
		return nil, nil
	}

	// Invalid JSON syntax is handled by response.invalid_body_format.
	if err := validateBodyForMediaType(ct, params, res.Body); err != nil {
		return nil, nil
	}

	media, ok := findMatchingMediaType(body.Content, ct)
	if !ok {
		return nil, nil
	}

	if media.Schema == nil {
		return nil, nil
	}

	var responseBodyJSON any
	if err := json.Unmarshal(res.Body, &responseBodyJSON); err != nil {
		return nil, nil
	}

	errs := validateJSONBodySchema(
		responseBodyJSON,
		*media.Schema,
		func(ref string) (*OpenAPISchema, bool) {
			return ctx.Contracts.ResolveSchemaRef(res.Meta.Host, ref)
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
				"Response body does not match the OpenAPI schema for %s %s with status %d: %s.",
				res.Meta.Method,
				res.Meta.Path,
				res.Meta.Status,
				joinSchemaErrors(errs),
			),
			CreatedAt: time.Now().UTC(),
		},
	}, nil
}

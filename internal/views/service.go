package views

import (
	"api-observer-website/internal/utils"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"gopkg.in/yaml.v3"
)

func validatePlaygroundRequest(req PlaygroundRequest) error {
	method := strings.ToUpper(strings.TrimSpace(req.Request.Method))

	switch method {
	case
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete:

	default:
		return fmt.Errorf(
			"unsupported request method %q",
			req.Request.Method,
		)
	}

	if strings.TrimSpace(req.Request.Path) == "" {
		return errors.New("request path is required")
	}

	if strings.TrimSpace(req.OpenAPI) == "" {
		return errors.New("OpenAPI document is required")
	}

	if strings.TrimSpace(req.HostRules) == "" {
		return errors.New("host rules document is required")
	}

	return nil
}

func parseHeaders(raw string) (http.Header, error) {
	headers := make(http.Header)

	if strings.TrimSpace(raw) == "" {
		return headers, nil
	}

	var input map[string]json.RawMessage

	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		return nil, fmt.Errorf(
			"headers must be a valid JSON object: %w",
			err,
		)
	}

	for name, rawValue := range input {
		name = strings.TrimSpace(name)

		if name == "" {
			return nil, errors.New(
				"header names cannot be empty",
			)
		}

		var singleValue string

		if err := json.Unmarshal(rawValue, &singleValue); err == nil {
			headers.Add(name, singleValue)
			continue
		}

		var multipleValues []string

		if err := json.Unmarshal(
			rawValue,
			&multipleValues,
		); err == nil {
			for _, value := range multipleValues {
				headers.Add(name, value)
			}

			continue
		}

		return nil, fmt.Errorf(
			"header %q must be a string or array of strings",
			name,
		)
	}

	return headers, nil
}

func parseMockRequestPath(rawPath string) (*url.URL, error) {
	rawPath = strings.TrimSpace(rawPath)

	if rawPath == "" {
		return nil, errors.New("request path is required")
	}

	if strings.Contains(rawPath, "://") {
		return nil, errors.New(
			"request must contain a relative path, not a complete URL",
		)
	}

	if !strings.HasPrefix(rawPath, "/") {
		rawPath = "/" + rawPath
	}

	parsed, err := url.Parse(rawPath)
	if err != nil {
		return nil, fmt.Errorf(
			"parse request path: %w",
			err,
		)
	}

	if parsed.Scheme != "" || parsed.Host != "" {
		return nil, errors.New(
			"request must contain a relative path",
		)
	}

	return parsed, nil
}

func ensureSingleJSONValue(decoder *json.Decoder) error {
	var extra any

	err := decoder.Decode(&extra)

	switch {
	case errors.Is(err, io.EOF):
		return nil

	case err == nil:
		return errors.New(
			"request body must contain exactly one JSON object",
		)

	default:
		return fmt.Errorf(
			"invalid data after JSON request: %w",
			err,
		)
	}
}

func writeDecodeError(w http.ResponseWriter, err error) {
	var syntaxError *json.SyntaxError
	var typeError *json.UnmarshalTypeError
	var maxBytesError *http.MaxBytesError

	switch {
	case errors.As(err, &syntaxError):
		utils.WriteJSONError(
			w,
			http.StatusBadRequest,
			"invalid_json",
			fmt.Sprintf(
				"Malformed JSON near byte %d",
				syntaxError.Offset,
			),
		)

	case errors.As(err, &typeError):
		utils.WriteJSONError(
			w,
			http.StatusBadRequest,
			"invalid_field_type",
			fmt.Sprintf(
				"Field %q contains an invalid value",
				typeError.Field,
			),
		)

	case errors.As(err, &maxBytesError):
		utils.WriteJSONError(
			w,
			http.StatusRequestEntityTooLarge,
			"request_too_large",
			"Playground request is too large",
		)

	case errors.Is(err, io.EOF):
		utils.WriteJSONError(
			w,
			http.StatusBadRequest,
			"empty_request",
			"Request body is required",
		)

	default:
		utils.WriteJSONError(
			w,
			http.StatusBadRequest,
			"invalid_request",
			"Unable to decode playground request",
		)
	}
}

func parseYamlOpenAPIDoc(raw string) (OpenAPIDoc, error) {
	if strings.TrimSpace(raw) == "" {
		return OpenAPIDoc{}, fmt.Errorf("OpenAPI document is required")
	}

	var doc OpenAPIDoc

	if err := yaml.Unmarshal([]byte(raw), &doc); err != nil {
		return OpenAPIDoc{}, fmt.Errorf("parse OpenAPI document: %w", err)
	}

	return doc, nil
}

func parseYAMLHostRules(raw string) (HostRulesDoc, error) {
	raw = strings.TrimSpace(raw)

	if raw == "" {
		return HostRulesDoc{}, errors.New("host rules YAML is required")
	}

	var hostRules HostRulesDoc

	if err := yaml.Unmarshal([]byte(raw), &hostRules); err != nil {
		return HostRulesDoc{}, fmt.Errorf("unmarshal host rules YAML: %w", err)
	}

	return hostRules, nil
}

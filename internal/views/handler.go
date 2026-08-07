package views

import (
	"api-observer-wiki/internal/utils"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

type Handler struct {
	templates *template.Template
}

func NewHandler() (*Handler, error) {
	templates, err := template.ParseGlob("internal/views/templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse view templates: %w", err)
	}

	return &Handler{
		templates: templates,
	}, nil
}

func (h *Handler) renderTemplate(w http.ResponseWriter, status int, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Template rendering failed", http.StatusInternalServerError)
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	staticHandler := http.StripPrefix(
		"/static/",
		http.FileServer(
			http.Dir("./internal/views/templates/static"),
		),
	)

	mux.Handle("GET /static/", staticHandler)
	mux.HandleFunc("POST /playground", Playground)
	mux.HandleFunc("GET /{$}", h.HomePage)
}

func (h *Handler) HomePage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	h.renderTemplate(w, http.StatusOK, "index.html", nil)
}

const maxPlaygroundRequestSize = 2 << 20 // 2 MiB

type PlaygroundRequest struct {
	Request   MockRequest `json:"request"`
	OpenAPI   string      `json:"openapi,omitempty"`
	HostRules string      `json:"rules,omitempty"`
}

type MockRequest struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	Headers string `json:"headers,omitempty"`
	Body    string `json:"body,omitempty"`
}

type PlaygroundResponse struct {
	Jobs     []AuditJob `json:"audit_jobs"`
	Findings []Finding  `json:"findings"`
}

type AuditJob struct {
	ID   string `json:"id"`
	Type string `json:"type"`

	RequestID string `json:"request_id"`
	Host      string `json:"host"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Query     string `json:"query,omitempty"`
	Status    int    `json:"status,omitempty"`
	Timestamp string `json:"timestamp"`

	Headers http.Header `json:"headers,omitempty"`
	Body    string      `json:"body,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type Finding struct {
	ID        string    `json:"id"`
	JobID     string    `json:"job_id"`
	RuleID    string    `json:"rule_id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

func Playground(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)

		utils.WriteJSONError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
			"Method not allowed",
		)
		return
	}

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxPlaygroundRequestSize,
	)

	var req PlaygroundRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeDecodeError(w, err)
		return
	}

	// Reject a request containing more than one JSON object.
	if err := ensureSingleJSONValue(decoder); err != nil {
		utils.WriteJSONError(
			w,
			http.StatusBadRequest,
			"invalid_json",
			err.Error(),
		)
		return
	}

	if err := validatePlaygroundRequest(req); err != nil {
		utils.WriteJSONError(
			w,
			http.StatusBadRequest,
			"invalid_playground_request",
			err.Error(),
		)
		return
	}

	headers, err := parseHeaders(req.Request.Headers)
	if err != nil {
		utils.WriteJSONError(
			w,
			http.StatusBadRequest,
			"invalid_headers",
			err.Error(),
		)
		return
	}

	requestURL, err := parseMockRequestPath(req.Request.Path)
	if err != nil {
		utils.WriteJSONError(
			w,
			http.StatusBadRequest,
			"invalid_path",
			err.Error(),
		)
		return
	}

	fakeReq := &http.Request{
		Method: req.Request.Method,
		URL:    requestURL,
		Header: headers,
	}
	reqJob := NewRequestJob(fakeReq, "api-observer-playground", time.Now())

	reqJob.Body = []byte(req.Request.Body)

	var openapi OpenAPIDoc
	if req.OpenAPI != "" {
		openapi, err = parseYamlOpenAPIDoc(req.OpenAPI)
		if err != nil {
			utils.WriteJSONError(w, http.StatusBadRequest, "failed_to_parse_yaml_openapi", fmt.Sprintf("Failed to parse OpenAPI YAML: %s", err.Error()))
			return
		}

		openapi, err = validateOpenAPIContractStructure(openapi)
		if err != nil {
			utils.WriteJSONError(w, http.StatusBadRequest, "invalid_openapi_document", fmt.Sprintf("Invalid OpenAPI document: %s", err.Error()))
			return
		}
	}

	var hostRules HostRulesDoc
	if req.HostRules != "" {
		hostRules, err = parseYAMLHostRules(req.HostRules)
		if err != nil {
			utils.WriteJSONError(w, http.StatusBadRequest, "failed_to_parse_hostrules_yaml", fmt.Sprintf("Failed to parse Host Rules YAML: %s", err.Error()))
			return
		}

		err = validateRulesDocument(hostRules)
		if err != nil {
			utils.WriteJSONError(w, http.StatusBadRequest, "invalid_hostrules_document", fmt.Sprintf("Invalid Host Rules document: %s", err.Error()))
			return
		}
	}

	contractRegistry := NewContractRegistry()
	contractRegistry.contracts["api-observer-playground"] = openapi
	contractRegistry.rules["api-observer-playground"] = hostRules

	engine := NewRuleEngine(contractRegistry)

	jobs := make([]AuditJob, 0)
	findings := make([]Finding, 0)

	a, f, err := reqJob.Process(engine)
	if err != nil {
		utils.WriteJSONError(
			w,
			http.StatusInternalServerError,
			"request_audit_failed",
			fmt.Sprintf("Failed to audit request: %s", err),
		)
		return
	}

	jobs = append(jobs, a)
	findings = append(findings, f...)

	responseBody := `{
  "id": "INC-2026-042",
  "title": "Unexpected outbound media activity",
  "status": "requires_review",
  "confidence": 0.99,
  "severity": "high",
  "summary": "A previously undocumented external resource was identified during request analysis.",
  "evidence": [
    {
      "type": "external_reference",
      "description": "Open the captured external resource for manual verification.",
      "url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
    }
  ],
  "recommendation": "Review the external resource before closing the investigation."
}`

	resp := &http.Response{
		Request:    fakeReq,
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
	}

	respJob := NewResponseJob(resp, "api-observer-playground")

	respJob.Body = []byte(responseBody)

	a, f, err = respJob.Process(engine)
	if err != nil {
		utils.WriteJSONError(
			w,
			http.StatusInternalServerError,
			"response_audit_failed",
			fmt.Sprintf("Failed to audit response: %s", err),
		)
		return
	}

	jobs = append(jobs, a)
	findings = append(findings, f...)

	utils.WriteJSON(
		w,
		http.StatusOK,
		PlaygroundResponse{
			Jobs:     jobs,
			Findings: findings,
		},
	)
}

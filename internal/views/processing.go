package views

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Job interface {
	JobType() JobType
	Metadata() Metadata
	Process(e *RuleEngine) (AuditJob, []Finding, error)
}

type Queue struct {
	jobs chan Job
	mu   sync.RWMutex
	done bool
	once sync.Once
}

type HTTPExchangeEvent struct {
	HostName string        `json:"host"`
	Request  *RequestCopy  `json:"request,omitempty"`
	Response *ResponseCopy `json:"response,omitempty"`
	Failure  *FailureCopy  `json:"failure,omitempty"`
}

type RequestCopy struct {
	Method string      `json:"method"`
	URL    string      `json:"url"`
	Header http.Header `json:"header"`
	Body   []byte      `json:"body"`
}

type ResponseCopy struct {
	Request    *RequestCopy `json:"request"`
	StatusCode int          `json:"status_code"`
	Headers    http.Header  `json:"headers"`
	Body       []byte       `json:"body"`
}

type FailureCopy struct {
	Request *RequestCopy `json:"request"`
	Error   string       `json:"error"`
}

func (r *RequestJob) JobType() JobType {
	return RequestJobType
}

func (r *RequestJob) Metadata() Metadata {
	return r.Meta
}

func (r *RequestJob) Process(engine *RuleEngine) (AuditJob, []Finding, error) {
	jobID := uuid.NewString()

	auditJob, err := SaveJob(r, jobID)
	if err != nil {
		return AuditJob{}, []Finding{}, fmt.Errorf("failed to save audit job: %s", err)
	}

	findings, err := engine.Evaluate(r, jobID)
	if err != nil {
		return AuditJob{}, []Finding{}, err
	}

	return auditJob, findings, nil
}

func (r *ResponseJob) JobType() JobType {
	return ResponseJobType
}

func (r *ResponseJob) Metadata() Metadata {
	return r.Meta
}

func (r *ResponseJob) Process(engine *RuleEngine) (AuditJob, []Finding, error) {
	jobID := uuid.NewString()

	auditJob, err := SaveJob(r, jobID)
	if err != nil {
		return AuditJob{}, []Finding{}, fmt.Errorf("failed to save audit job: %s", err)
	}

	findings, err := engine.Evaluate(r, jobID)
	if err != nil {
		return AuditJob{}, []Finding{}, err
	}

	return auditJob, findings, nil
}

func (r *FailureJob) JobType() JobType {
	return FailureJobType
}

func (r *FailureJob) Metadata() Metadata {
	return r.Meta
}

func (r *FailureJob) Process(engine *RuleEngine) (AuditJob, []Finding, error) {
	jobID := uuid.NewString()

	auditJob, err := SaveJob(r, jobID)
	if err != nil {
		return AuditJob{}, []Finding{}, fmt.Errorf("failed to save audit job: %s", err)
	}

	findings, err := engine.Evaluate(r, jobID)
	if err != nil {
		return AuditJob{}, []Finding{}, err
	}

	return auditJob, findings, nil
}

func SaveJob(job Job, jobID string) (AuditJob, error) {
	if job == nil {
		return AuditJob{}, fmt.Errorf("missing job")
	}

	if jobID == "" {
		return AuditJob{}, fmt.Errorf("missing job ID")

	}

	var (
		jobType string
		meta    Metadata
		headers http.Header
		body    string
		errStr  string
	)

	switch j := job.(type) {
	case *RequestJob:
		jobType = string(j.JobType())
		meta = j.Meta

		headers = j.Headers

		body = string(j.Body)
	case *ResponseJob:
		jobType = string(j.JobType())
		meta = j.Meta

		headers = j.Headers

		body = string(j.Body)
	case *FailureJob:
		jobType = string(j.JobType())
		meta = j.Meta

		errStr = j.Error
	default:
		return AuditJob{}, fmt.Errorf("unknown job type: %T", job)
	}

	auditJob := AuditJob{
		ID:        jobID,
		Type:      jobType,
		RequestID: meta.RequestID,
		Host:      meta.Host,
		Method:    meta.Method,
		Path:      meta.Path,
		Query:     meta.Query,
		Status:    meta.Status,
		Timestamp: meta.Timestamp.Format(time.RFC3339Nano),
		Headers:   headers,
		Body:      body,
		Error:     errStr,
	}

	return auditJob, nil
}

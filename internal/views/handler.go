package views

import (
	"api-observer-website/internal/utils"
	"fmt"
	"html/template"
	"net/http"
)

type Handler struct {
	service   *Service
	templates *template.Template
}

func NewHandler(service *Service) (*Handler, error) {
	templates, err := template.ParseGlob("internal/views/templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse view templates: %w", err)
	}

	return &Handler{
		service:   service,
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
	mux.HandleFunc("GET /{$}", h.HomePage)
}

func (h *Handler) HomePage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	h.renderTemplate(w, http.StatusOK, "index.html", nil)
}

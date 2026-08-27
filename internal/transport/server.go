package transport

import (
	"archive-release/internal/application"
	"archive-release/internal/domain"
	"archive-release/internal/release"
	webassets "archive-release/web"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type Server struct {
	app *application.Service
	mux *http.ServeMux
}

func New(app *application.Service) *Server {
	s := &Server{app: app, mux: http.NewServeMux()}
	s.routes()
	return s
}
func (s *Server) routes() {
	s.mux.HandleFunc("/", s.index)
	s.mux.HandleFunc("/api/cases", s.cases)
	s.mux.HandleFunc("/api/review-queue", s.reviewQueue)
	s.mux.HandleFunc("/api/review-pending", s.reviewQueue)
	s.mux.HandleFunc("/api/cases/", s.caseRoutes)
	s.mux.Handle("/static/", http.FileServer(http.FS(webassets.Files)))
}
func (s *Server) Handler() http.Handler { return logging(s.mux) }
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}
func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(webassets.IndexHTML)
}
func (s *Server) cases(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, s.app.ListCases())
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in struct{ Title, ArchiveUnit, Creator, Actor, Role, IdempotencyKey string }
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		bad(w, err)
		return
	}
	c, err := s.app.CreateCase(in.Title, in.ArchiveUnit, in.Creator, in.Actor, domain.Role(in.Role), in.IdempotencyKey)
	if err != nil {
		bad(w, err)
		return
	}
	writeJSON(w, c)
}
func (s *Server) caseRoutes(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.NotFound(w, r)
		return
	}
	id := parts[2]
	if len(parts) == 3 && r.Method == http.MethodGet {
		c, e := s.app.GetCase(id)
		if e != nil {
			bad(w, e)
			return
		}
		writeJSON(w, c)
		return
	}
	if len(parts) < 4 {
		http.NotFound(w, r)
		return
	}
	if len(parts) >= 5 && ((parts[3] == "manifest" && parts[4] == "preview") || (parts[3] == "freeze" && parts[4] == "preview")) {
		s.preview(w, r, id)
		return
	}
	if len(parts) >= 5 && parts[3] == "findings" && parts[4] == "stats" {
		s.findingStats(w, r, id)
		return
	}
	if len(parts) >= 5 && parts[3] == "findings" && (parts[4] == "batch" || parts[4] == "resolve-batch" || parts[4] == "remediate") {
		s.remediations(w, r, id)
		return
	}
	if len(parts) >= 5 && parts[3] == "artifact" && parts[4] == "verify" {
		s.verify(w, r, id)
		return
	}
	switch parts[3] {
	case "revisions":
		s.revision(w, r, id)
	case "findings":
		s.finding(w, r, id)
	case "finding-stats":
		s.findingStats(w, r, id)
	case "remediations":
		s.remediations(w, r, id)
	case "review":
		s.review(w, r, id)
	case "freeze":
		s.freeze(w, r, id)
	case "manifest":
		s.manifest(w, r, id)
	case "preview":
		s.preview(w, r, id)
	case "manifest-preview":
		s.preview(w, r, id)
	case "artifact":
		s.artifact(w, r, id)
	case "verify":
		s.verify(w, r, id)
	case "timeline":
		writeJSON(w, s.app.Timeline(id))
	case "audit":
		writeJSON(w, s.app.Audit(id))
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) findingStats(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	v, e := s.app.FindingStats(id)
	if e != nil {
		if errors.Is(e, os.ErrNotExist) {
			http.NotFound(w, r)
		} else {
			bad(w, e)
		}
		return
	}
	writeJSON(w, v)
}
func (s *Server) remediations(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in struct {
		Actor, Role, IdempotencyKey string
		ExpectedVersion             int
		Resolutions                 map[string]string `json:"resolutions"`
		Findings                    []struct {
			FindingID string `json:"findingID"`
			Note      string `json:"note"`
		} `json:"findings"`
	}
	if e := decode(r, &in); e != nil {
		bad(w, e)
		return
	}
	if in.Resolutions == nil {
		in.Resolutions = map[string]string{}
	}
	for _, f := range in.Findings {
		in.Resolutions[f.FindingID] = f.Note
	}
	c, e := s.app.ResolveFindings(id, in.Actor, domain.Role(in.Role), in.ExpectedVersion, in.IdempotencyKey, in.Resolutions)
	if e != nil {
		bad(w, e)
		return
	}
	writeJSON(w, c)
}
func (s *Server) preview(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	v, e := s.app.Preview(id)
	if e != nil {
		if errors.Is(e, os.ErrNotExist) {
			http.NotFound(w, r)
		} else {
			bad(w, e)
		}
		return
	}
	writeJSON(w, v)
}
func (s *Server) artifact(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	a, e := s.app.ArtifactForCase(id)
	if e != nil {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, map[string]any{"manifest": a.Manifest, "credential": a.Credential, "digest": a.Digest, "schemaVersion": "1.0"})
}
func (s *Server) manifest(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	a, err := s.app.ArtifactForCase(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, a.Manifest)
}
func (s *Server) verify(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method == http.MethodGet {
		writeJSON(w, s.app.VerifyArtifact(id))
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var a release.Artifact
	if e := decode(r, &a); e != nil {
		bad(w, e)
		return
	}
	writeJSON(w, s.app.VerifySubmittedContext(r.Context(), id, a))
}
func decode(r *http.Request, v any) error { return json.NewDecoder(r.Body).Decode(v) }
func (s *Server) revision(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in struct {
		Actor, Role, IdempotencyKey, ParentRevisionID string
		ExpectedVersion                               int
		Pages                                         []domain.PageMeasurement
		Metadata                                      map[string]string
	}
	if e := decode(r, &in); e != nil {
		bad(w, e)
		return
	}
	c, e := s.app.SubmitRevision(id, in.Actor, domain.Role(in.Role), in.ExpectedVersion, in.IdempotencyKey, in.Pages, in.Metadata, in.ParentRevisionID)
	if e != nil {
		bad(w, e)
		return
	}
	writeJSON(w, c)
}
func (s *Server) finding(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method == http.MethodGet {
		page := 0
		if v := r.URL.Query().Get("pageNumber"); v != "" {
			if _, e := fmt.Sscanf(v, "%d", &page); e != nil || page < 1 {
				bad(w, errors.New("pageNumber 参数无效"))
				return
			}
		}
		f, e := s.app.FindingsFiltered(id, r.URL.Query().Get("severity"), r.URL.Query().Get("ruleCode"), page)
		if e != nil {
			if errors.Is(e, os.ErrNotExist) {
				http.NotFound(w, r)
			} else {
				bad(w, e)
			}
			return
		}
		writeJSON(w, f)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in struct {
		FindingID, Actor, Role, IdempotencyKey, Note string
		ExpectedVersion                              int
	}
	if e := decode(r, &in); e != nil {
		bad(w, e)
		return
	}
	c, e := s.app.ResolveFinding(id, in.FindingID, in.Actor, domain.Role(in.Role), in.ExpectedVersion, in.IdempotencyKey, in.Note)
	if e != nil {
		bad(w, e)
		return
	}
	writeJSON(w, c)
}

func (s *Server) reviewQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, s.app.ReviewQueue())
}
func (s *Server) review(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in struct {
		Actor, Role, IdempotencyKey, Note string
		ExpectedVersion                   int
		Approved                          bool
	}
	if e := decode(r, &in); e != nil {
		bad(w, e)
		return
	}
	c, e := s.app.Review(id, in.Actor, domain.Role(in.Role), in.ExpectedVersion, in.IdempotencyKey, in.Approved, in.Note)
	if e != nil {
		bad(w, e)
		return
	}
	writeJSON(w, c)
}
func (s *Server) freeze(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var in struct {
		Actor, Role, IdempotencyKey, PreviewDigest string
		ExpectedVersion                            int
	}
	if e := decode(r, &in); e != nil {
		bad(w, e)
		return
	}
	c, m, cred, e := s.app.FreezeWithPreview(id, in.Actor, domain.Role(in.Role), in.ExpectedVersion, in.IdempotencyKey, in.PreviewDigest)
	if e != nil {
		bad(w, e)
		return
	}
	writeJSON(w, map[string]any{"case": c, "manifest": m, "credential": cred})
}
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
func bad(w http.ResponseWriter, e error) {
	code := http.StatusBadRequest
	if errors.Is(e, os.ErrNotExist) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if strings.Contains(e.Error(), "版本冲突") || strings.Contains(e.Error(), "已存在") || strings.Contains(e.Error(), "重复案卷") || strings.Contains(e.Error(), "摘要不一致") {
		code = http.StatusConflict
	}
	http.Error(w, e.Error(), code)
}
func methodNotAllowed(w http.ResponseWriter) { w.WriteHeader(http.StatusMethodNotAllowed) }

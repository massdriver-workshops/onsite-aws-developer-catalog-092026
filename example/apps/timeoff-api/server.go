package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

type server struct {
	cfg    config
	store  *Store
	events *Publisher
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /info", s.info)
	mux.HandleFunc("GET /employees", s.listEmployees)
	mux.HandleFunc("POST /session", s.createSession)
	mux.HandleFunc("GET /requests", s.listRequests)
	mux.HandleFunc("POST /requests", s.createRequest)
	mux.HandleFunc("POST /requests/{id}/approve", s.decide("approved"))
	mux.HandleFunc("POST /requests/{id}/deny", s.decide("denied"))
	return logRequests(mux)
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.store.Healthy(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unreachable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) info(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"service":              "timeoff-api",
		"version":              version,
		"namespace":            s.cfg.Namespace,
		"company_name":         s.cfg.CompanyName,
		"kinds":                requestKinds,
		"max_days_per_request": s.cfg.MaxDaysPerRequest,
		"auto_approve":         s.cfg.AutoApprove,
		"approval_window_days": s.cfg.ApprovalWindowDays,
		"events_enabled":       s.events != nil,
		"events_topic":         s.cfg.Kafka.Topic,
	})
}

func (s *server) listEmployees(w http.ResponseWriter, r *http.Request) {
	emps, err := s.store.ListEmployees(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, emps)
}

// createSession issues a signed token naming the employee the browser acts as.
// The demo has no login; the token exists so that decisions carry a signed identity.
func (s *server) createSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		EmployeeID int64 `json:"employee_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	emp, err := s.store.GetEmployee(r.Context(), body.EmployeeID)
	if err != nil {
		serverError(w, err)
		return
	}
	if emp == nil {
		writeError(w, http.StatusNotFound, "no such employee")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token":    s.signToken(emp.ID),
		"employee": emp,
	})
}

func (s *server) listRequests(w http.ResponseWriter, r *http.Request) {
	reqs, err := s.store.ListRequests(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.present(reqs))
}

func (s *server) createRequest(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.actor(w, r)
	if !ok {
		return
	}
	var body struct {
		Kind      string `json:"kind"`
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
		Note      string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if !slices.Contains(requestKinds, body.Kind) {
		writeError(w, http.StatusUnprocessableEntity, "kind must be one of "+strings.Join(requestKinds, ", "))
		return
	}
	start, err1 := time.Parse("2006-01-02", body.StartDate)
	end, err2 := time.Parse("2006-01-02", body.EndDate)
	if err1 != nil || err2 != nil {
		writeError(w, http.StatusUnprocessableEntity, "start_date and end_date must be YYYY-MM-DD")
		return
	}
	if end.Before(start) {
		writeError(w, http.StatusUnprocessableEntity, "end_date must not be before start_date")
		return
	}
	days := int(end.Sub(start).Hours()/24) + 1
	if days > s.cfg.MaxDaysPerRequest {
		writeError(w, http.StatusUnprocessableEntity, "a single request may cover at most "+strconv.Itoa(s.cfg.MaxDaysPerRequest)+" days")
		return
	}
	if len(body.Note) > 500 {
		writeError(w, http.StatusUnprocessableEntity, "note must be 500 characters or fewer")
		return
	}

	status, decidedBy := "pending", ""
	if s.cfg.AutoApprove {
		status, decidedBy = "approved", "auto-approval"
	}
	req, err := s.store.CreateRequest(r.Context(), actor.ID, body.Kind, start, end, body.Note, status, decidedBy)
	if err != nil {
		serverError(w, err)
		return
	}
	s.events.Publish(r.Context(), eventFromRequest(EventRequested, req, s.cfg.Namespace))
	if s.cfg.AutoApprove {
		s.events.Publish(r.Context(), eventFromRequest(EventDecided, req, s.cfg.Namespace))
	}
	writeJSON(w, http.StatusCreated, s.presentOne(*req))
}

func (s *server) decide(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := s.actor(w, r)
		if !ok {
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid request id")
			return
		}
		req, err := s.store.Decide(r.Context(), id, status, actor.Name)
		if err != nil {
			serverError(w, err)
			return
		}
		if req == nil {
			writeError(w, http.StatusConflict, "request not found or already decided")
			return
		}
		s.events.Publish(r.Context(), eventFromRequest(EventDecided, req, s.cfg.Namespace))
		writeJSON(w, http.StatusOK, s.presentOne(*req))
	}
}

// --- presentation ---

type requestView struct {
	ID         int64  `json:"id"`
	EmployeeID int64  `json:"employee_id"`
	Employee   string `json:"employee"`
	Team       string `json:"team"`
	Kind       string `json:"kind"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	Days       int    `json:"days"`
	Note       string `json:"note"`
	Status     string `json:"status"`
	DecidedBy  string `json:"decided_by,omitempty"`
	CreatedAt  string `json:"created_at"`
	Overdue    bool   `json:"overdue"`
}

func (s *server) presentOne(r Request) requestView {
	window := time.Duration(s.cfg.ApprovalWindowDays) * 24 * time.Hour
	return requestView{
		ID:         r.ID,
		EmployeeID: r.EmployeeID,
		Employee:   r.Employee,
		Team:       r.Team,
		Kind:       r.Kind,
		StartDate:  r.StartDate.Format("2006-01-02"),
		EndDate:    r.EndDate.Format("2006-01-02"),
		Days:       r.Days(),
		Note:       r.Note,
		Status:     r.Status,
		DecidedBy:  r.DecidedBy.String,
		CreatedAt:  r.CreatedAt.UTC().Format(time.RFC3339),
		Overdue:    r.Status == "pending" && !s.cfg.AutoApprove && time.Since(r.CreatedAt) > window,
	}
}

func (s *server) present(rs []Request) []requestView {
	out := make([]requestView, 0, len(rs))
	for _, r := range rs {
		out = append(out, s.presentOne(r))
	}
	return out
}

// --- session tokens ---

func (s *server) signToken(employeeID int64) string {
	id := strconv.FormatInt(employeeID, 10)
	return id + "." + s.mac(id)
}

func (s *server) mac(msg string) string {
	h := hmac.New(sha256.New, []byte(s.cfg.SessionSecret))
	h.Write([]byte(msg))
	return hex.EncodeToString(h.Sum(nil))
}

var errBadToken = errors.New("invalid session token")

func (s *server) parseToken(token string) (int64, error) {
	id, sig, ok := strings.Cut(token, ".")
	if !ok || !hmac.Equal([]byte(sig), []byte(s.mac(id))) {
		return 0, errBadToken
	}
	return strconv.ParseInt(id, 10, 64)
}

func (s *server) actor(w http.ResponseWriter, r *http.Request) (*Employee, bool) {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		writeError(w, http.StatusUnauthorized, "choose who you are acting as first")
		return nil, false
	}
	id, err := s.parseToken(strings.TrimSpace(token))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid session token")
		return nil, false
	}
	emp, err := s.store.GetEmployee(r.Context(), id)
	if err != nil {
		serverError(w, err)
		return nil, false
	}
	if emp == nil {
		writeError(w, http.StatusUnauthorized, "session refers to an unknown employee")
		return nil, false
	}
	return emp, true
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Yanami-2K/todo-board/internal/store"
	"github.com/Yanami-2K/todo-board/web"
	"golang.org/x/crypto/bcrypt"
)

const (
	cookieName = "tb_session"
	version    = "0.1.0"
)

// Config is process configuration, mostly from the environment.
type Config struct {
	OwnerPassword  string
	OwnerName      string
	IntakeIntro    string
	IntakeHomepage string
	IntakeMode     string // open (default) or token
	IntakeToken    string
	BcryptCost     int
	Location       *time.Location
	SessionTTL     time.Duration
}

// Server serves the UI, REST API and MCP endpoint.
type Server struct {
	cfg Config
	st  *store.Store
	mux *http.ServeMux
}

func New(cfg Config, st *store.Store) *Server {
	if cfg.Location == nil {
		cfg.Location = time.Local
	}
	if cfg.SessionTTL == 0 {
		cfg.SessionTTL = 14 * 24 * time.Hour
	}
	if cfg.BcryptCost == 0 {
		cfg.BcryptCost = bcrypt.DefaultCost
	}
	if cfg.IntakeMode == "" {
		cfg.IntakeMode = "open"
	}
	s := &Server{cfg: cfg, st: st, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler { return s.log(s.mux) }

func (s *Server) routes() {
	_ = mime.AddExtensionType(".woff2", "font/woff2")
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("POST /api/login", s.handleLogin)
	s.mux.HandleFunc("POST /api/logout", s.handleLogout)
	s.mux.HandleFunc("GET /api/me", s.handleMe)
	s.mux.HandleFunc("GET /api/board", s.handleBoard)
	s.mux.HandleFunc("GET /api/todos", s.handleListTodos)
	s.mux.HandleFunc("POST /api/todos", s.handleCreateTodo)
	s.mux.HandleFunc("GET /api/todos/{id}", s.handleGetTodo)
	s.mux.HandleFunc("PATCH /api/todos/{id}", s.handlePatchTodo)
	s.mux.HandleFunc("DELETE /api/todos/{id}", s.handleDeleteTodo)
	s.mux.HandleFunc("POST /api/todos/{id}/complete", s.handleComplete)
	s.mux.HandleFunc("POST /api/todos/{id}/comments", s.handleComment)
	s.mux.HandleFunc("GET /api/questions", s.handleListQuestions)
	s.mux.HandleFunc("POST /api/questions", s.handleAsk)
	s.mux.HandleFunc("POST /api/questions/{id}/answer", s.handleAnswer)
	s.mux.HandleFunc("GET /api/inbox", s.handleListInbox)
	s.mux.HandleFunc("POST /api/inbox/{id}/accept", s.handleAccept)
	s.mux.HandleFunc("POST /api/inbox/{id}/decline", s.handleDecline)
	s.mux.HandleFunc("GET /api/intake", s.handleIntakeInfo)
	s.mux.HandleFunc("POST /api/intake", s.handleIntakeSubmit)
	s.mux.HandleFunc("GET /api/agents", s.handleListAgents)
	s.mux.HandleFunc("POST /api/agents", s.handleCreateAgent)
	s.mux.HandleFunc("PATCH /api/agents/{id}", s.handlePatchAgent)
	s.mux.HandleFunc("DELETE /api/agents/{id}", s.handleDeleteAgent)
	s.mux.HandleFunc("POST /api/agents/{id}/token", s.handleRotate)
	s.mux.HandleFunc("POST /api/agents/{id}/revoke", s.handleRevoke)
	s.mux.HandleFunc("GET /api/projects", s.handleListProjects)
	s.mux.HandleFunc("POST /api/projects", s.handleCreateProject)
	s.mux.HandleFunc("PATCH /api/projects/{id}", s.handlePatchProject)
	s.mux.HandleFunc("DELETE /api/projects/{id}", s.handleDeleteProject)
	s.mux.HandleFunc("GET /openapi.yaml", s.handleOpenAPI)
	s.mux.HandleFunc("POST /mcp", s.handleMCP)
	s.mux.HandleFunc("DELETE /mcp", s.handleMCPDelete)
	s.mux.HandleFunc("GET /mcp", s.handleMCPGet)
	s.mux.HandleFunc("GET /{$}", s.handlePage("index.html"))
	s.mux.HandleFunc("GET /login", s.handlePage("login.html"))
	s.mux.HandleFunc("GET /intake", s.handlePage("intake.html"))
	sub := http.FileServer(http.FS(web.FS))
	s.mux.Handle("GET /css/", sub)
	s.mux.Handle("GET /js/", sub)
	s.mux.Handle("GET /fonts/", sub)
}

func (s *Server) log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handlePage(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := web.FS.ReadFile(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(b)
	}
}

func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	b, err := web.FS.ReadFile("openapi.yaml")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	_, _ = w.Write(b)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "version": version})
}

type principal struct {
	Kind  string // owner, agent, intake, anon
	Agent store.Agent
	Actor store.Actor
}

func (s *Server) principal(r *http.Request) (principal, error) {
	if h := r.Header.Get("Authorization"); h != "" {
		const pfx = "Bearer "
		if !strings.HasPrefix(h, pfx) || strings.TrimSpace(h[len(pfx):]) == "" {
			return principal{}, &store.Error{Code: "unauthorized", Msg: "token 无效或已吊销"}
		}
		token := strings.TrimSpace(h[len(pfx):])
		ag, err := s.st.AgentByToken(token)
		if err == nil {
			_ = s.st.TouchAgent(ag.ID)
			return principal{Kind: "agent", Agent: ag, Actor: store.Actor{Kind: "agent", ID: ag.ID, Name: ag.Name}}, nil
		}
		if s.cfg.IntakeToken != "" && constEq(token, s.cfg.IntakeToken) {
			return principal{Kind: "intake", Actor: store.Actor{Kind: "external", Name: "登记"}}, nil
		}
		return principal{}, &store.Error{Code: "unauthorized", Msg: "token 无效或已吊销"}
	}
	c, err := r.Cookie(cookieName)
	if err == nil && c.Value != "" {
		ok, err := s.st.ValidSession(c.Value)
		if err != nil {
			return principal{}, err
		}
		if ok {
			name := s.ownerName()
			return principal{Kind: "owner", Actor: store.Actor{Kind: "owner", ID: "owner", Name: name}}, nil
		}
	}
	return principal{Kind: "anon"}, nil
}

func (s *Server) ownerName() string {
	n, err := s.st.GetSetting("owner_name")
	if err != nil || strings.TrimSpace(n) == "" {
		if s.cfg.OwnerName != "" {
			return s.cfg.OwnerName
		}
		return "我"
	}
	return n
}

func (s *Server) requireOwner(w http.ResponseWriter, r *http.Request) (principal, bool) {
	p, err := s.principal(r)
	if err != nil {
		s.writeErr(w, err)
		return principal{}, false
	}
	if p.Kind != "owner" {
		if p.Kind == "anon" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "先登录"})
		} else {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "只有主人能做这件事"})
		}
		return principal{}, false
	}
	if !safeMethod(r.Method) && !sameOrigin(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "来源不对"})
		return principal{}, false
	}
	return p, true
}

func (s *Server) requireUser(w http.ResponseWriter, r *http.Request) (principal, bool) {
	p, err := s.principal(r)
	if err != nil {
		s.writeErr(w, err)
		return principal{}, false
	}
	if p.Kind != "owner" && p.Kind != "agent" {
		if p.Kind == "anon" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "先登录"})
		} else {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "这个 token 不能动 todo"})
		}
		return principal{}, false
	}
	if p.Kind == "owner" && !safeMethod(r.Method) && !sameOrigin(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "来源不对"})
		return principal{}, false
	}
	return p, true
}

func safeMethod(m string) bool { return m == http.MethodGet || m == http.MethodHead }

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

func constEq(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return &store.Error{Code: store.CodeInvalid, Msg: "缺少请求体"}
		}
		return &store.Error{Code: store.CodeInvalid, Msg: "请求体不是合法的 JSON"}
	}
	return nil
}

func (s *Server) writeErr(w http.ResponseWriter, err error) {
	if e := store.AsError(err); e != nil {
		code := http.StatusBadRequest
		switch e.Code {
		case store.CodeNotFound:
			code = http.StatusNotFound
		case store.CodeConflict:
			code = http.StatusConflict
		case store.CodeForbidden:
			code = http.StatusForbidden
		case "unauthorized":
			code = http.StatusUnauthorized
		}
		writeJSON(w, code, map[string]string{"error": e.Msg})
		return
	}
	log.Printf("server error: %v", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "服务器出错了"})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Password string `json:"password"`
	}
	if err := readJSON(w, r, &in); err != nil {
		s.writeErr(w, err)
		return
	}
	hash, err := s.st.GetSetting("password_hash")
	if err != nil {
		s.writeErr(w, err)
		return
	}
	if hash == "" || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "密码不对"})
		return
	}
	raw, err := s.st.CreateSession(s.cfg.SessionTTL)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    raw,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(s.cfg.SessionTTL.Seconds()),
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": s.ownerName()})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		_ = s.st.DeleteSession(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", HttpOnly: true, MaxAge: -1})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireOwner(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"name": p.Actor.Name, "look": store.GenLook(p.Actor.Name, 0)})
}

// EnsureOwner stores the owner password and display name. A generated password
// is returned when none was configured and none is stored yet.
func EnsureOwner(st *store.Store, cfg Config) (string, error) {
	if cfg.BcryptCost == 0 {
		cfg.BcryptCost = bcrypt.DefaultCost
	}
	generated := ""
	hash, err := st.GetSetting("password_hash")
	if err != nil {
		return "", err
	}
	pass := cfg.OwnerPassword
	if pass == "" && hash == "" {
		pass, err = randomPassword()
		if err != nil {
			return "", err
		}
		generated = pass
	}
	if pass != "" {
		sum, err := bcrypt.GenerateFromPassword([]byte(pass), cfg.BcryptCost)
		if err != nil {
			return "", err
		}
		if err := st.SetSetting("password_hash", string(sum)); err != nil {
			return "", err
		}
	}
	if cfg.OwnerName != "" {
		if err := st.SetSetting("owner_name", cfg.OwnerName); err != nil {
			return "", err
		}
	} else {
		n, err := st.GetSetting("owner_name")
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(n) == "" {
			if err := st.SetSetting("owner_name", "我"); err != nil {
				return "", err
			}
		}
	}
	if cfg.IntakeIntro != "" {
		if err := st.SetSetting("intake_intro", cfg.IntakeIntro); err != nil {
			return "", err
		}
	}
	if cfg.IntakeHomepage != "" {
		if err := st.SetSetting("intake_homepage", cfg.IntakeHomepage); err != nil {
			return "", err
		}
	}
	return generated, nil
}

func randomPassword() (string, error) {
	b := make([]byte, 9)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

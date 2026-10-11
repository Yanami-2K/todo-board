package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Yanami-2K/todo-board/internal/store"
)

func (s *Server) now() time.Time { return time.Now().In(s.cfg.Location) }

func (s *Server) ago(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	now := time.Now()
	if d := now.Sub(t); d < time.Minute {
		return "刚刚"
	} else if d < time.Hour {
		return fmt.Sprintf("%d 分钟前", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%d 小时前", int(d.Hours()))
	}
	loc := s.cfg.Location
	lt := t.In(loc)
	ln := now.In(loc)
	day := func(x time.Time) time.Time {
		return time.Date(x.Year(), x.Month(), x.Day(), 0, 0, 0, 0, loc)
	}
	if day(ln).Sub(day(lt)) == 24*time.Hour {
		return "昨天 " + lt.Format("15:04")
	}
	if lt.Year() == ln.Year() {
		return fmt.Sprintf("%d月%d日 %s", int(lt.Month()), lt.Day(), lt.Format("15:04"))
	}
	return lt.Format("2006-01-02")
}

func (s *Server) eyebrow() string {
	now := s.now()
	weeks := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	h := now.Hour()
	g := "晚上好"
	switch {
	case h < 5:
		g = "夜深了"
	case h < 11:
		g = "早上好"
	case h < 14:
		g = "中午好"
	case h < 18:
		g = "下午好"
	}
	return fmt.Sprintf("%s · %d 月 %d 日 · %s", weeks[now.Weekday()], int(now.Month()), now.Day(), g)
}

type agentView struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Role        string     `json:"role"`
	Look        store.Look `json:"look"`
	TokenPrefix string     `json:"token_prefix"`
	Revoked     bool       `json:"revoked"`
	Since       string     `json:"since"`
	SinceShort  string     `json:"since_short"`
	LastLabel   string     `json:"last_label"`
}

func (s *Server) viewAgent(a store.Agent) agentView {
	lt := a.CreatedAt.In(s.cfg.Location)
	last := "还没来过"
	if a.LastSeenAt != nil {
		last = s.ago(*a.LastSeenAt)
	}
	return agentView{
		ID: a.ID, Name: a.Name, Role: a.Role, Look: a.Look, TokenPrefix: a.TokenPrefix, Revoked: a.Revoked,
		Since: lt.Format("2006-01-02"), SinceShort: lt.Format("01.02"), LastLabel: last,
	}
}

func (s *Server) viewAgents(ags []store.Agent) []agentView {
	out := make([]agentView, 0, len(ags))
	for _, a := range ags {
		out = append(out, s.viewAgent(a))
	}
	return out
}

type questionView struct {
	ID        string    `json:"id"`
	TodoID    string    `json:"todo_id"`
	AgentID   string    `json:"agent_id"`
	Text      string    `json:"text"`
	Context   string    `json:"context"`
	Options   []string  `json:"options"`
	CreatedAt time.Time `json:"created_at"`
	Ago       string    `json:"ago"`
	Answer    string    `json:"answer,omitempty"`
}

func (s *Server) viewQuestions(qs []store.Question) []questionView {
	out := make([]questionView, 0, len(qs))
	for _, q := range qs {
		opts := q.Options
		if opts == nil {
			opts = []string{}
		}
		out = append(out, questionView{
			ID: q.ID, TodoID: q.TodoID, AgentID: q.AgentID, Text: q.Text, Context: q.Context,
			Options: opts, CreatedAt: q.CreatedAt, Ago: s.ago(q.CreatedAt), Answer: q.Answer,
		})
	}
	return out
}

type intakeView struct {
	store.Intake
	Ago string `json:"ago"`
}

func (s *Server) viewIntakes(items []store.Intake) []intakeView {
	out := make([]intakeView, 0, len(items))
	for _, it := range items {
		out = append(out, intakeView{Intake: it, Ago: s.ago(it.CreatedAt)})
	}
	return out
}

type activityView struct {
	ID        int64     `json:"id"`
	ActorKind string    `json:"actor_kind"`
	ActorID   string    `json:"actor_id"`
	ActorName string    `json:"actor_name"`
	Kind      string    `json:"kind"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
	Ago       string    `json:"ago"`
}

func (s *Server) viewActivity(items []store.Activity) []activityView {
	out := make([]activityView, 0, len(items))
	for _, a := range items {
		out = append(out, activityView{
			ID: a.ID, ActorKind: a.ActorKind, ActorID: a.ActorID, ActorName: a.ActorName,
			Kind: a.Kind, Text: a.Text, CreatedAt: a.CreatedAt, Ago: s.ago(a.CreatedAt),
		})
	}
	return out
}

func (s *Server) handleBoard(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireOwner(w, r)
	if !ok {
		return
	}
	ags, err := s.st.ListAgents()
	if err != nil {
		s.writeErr(w, err)
		return
	}
	ps, err := s.st.ListProjects()
	if err != nil {
		s.writeErr(w, err)
		return
	}
	todos, err := s.st.ListTodos()
	if err != nil {
		s.writeErr(w, err)
		return
	}
	qs, err := s.st.ListOpenQuestions()
	if err != nil {
		s.writeErr(w, err)
		return
	}
	pending, err := s.st.ListIntakes("pending")
	if err != nil {
		s.writeErr(w, err)
		return
	}
	declined, err := s.st.ListIntakes("declined")
	if err != nil {
		s.writeErr(w, err)
		return
	}
	acts, err := s.st.ListAllActivity()
	if err != nil {
		s.writeErr(w, err)
		return
	}
	byTodo := map[string][]activityView{}
	for _, a := range acts {
		v := s.viewActivity([]store.Activity{a})[0]
		byTodo[a.TodoID] = append(byTodo[a.TodoID], v)
	}
	if ps == nil {
		ps = []store.Project{}
	}
	if todos == nil {
		todos = []store.Todo{}
	}
	if byTodo == nil {
		byTodo = map[string][]activityView{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"me":        map[string]any{"name": p.Actor.Name, "look": store.GenLook(p.Actor.Name, 0)},
		"eyebrow":   s.eyebrow(),
		"projects":  ps,
		"agents":    s.viewAgents(ags),
		"todos":     todos,
		"questions": s.viewQuestions(qs),
		"inbox":     s.viewIntakes(pending),
		"declined":  s.viewIntakes(declined),
		"activity":  byTodo,
	})
}

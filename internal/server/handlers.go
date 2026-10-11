package server

import (
	"net/http"
	"strings"

	"github.com/Yanami-2K/todo-board/internal/store"
)

func (s *Server) handleListTodos(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	all, err := s.st.ListTodos()
	if err != nil {
		s.writeErr(w, err)
		return
	}
	q := r.URL.Query()
	project := q.Get("project")
	if project != "" {
		if _, err := s.st.GetProject(project); err != nil {
			if pr, err2 := s.st.ProjectByName(project); err2 == nil {
				project = pr.ID
			}
		}
	}
	out := filterTodos(all, p, q.Get("status"), project, q.Get("lane"), q.Get("q"))
	if out == nil {
		out = []store.Todo{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"todos": out})
}

func filterTodos(all []store.Todo, p principal, status, project, lane, q string) []store.Todo {
	var out []store.Todo
	q = strings.ToLower(strings.TrimSpace(q))
	for _, t := range all {
		if p.Kind == "agent" && t.DelegateID != p.Agent.ID {
			continue
		}
		if status != "" && t.Status != status {
			continue
		}
		if project != "" && t.ProjectID != project && !strings.EqualFold(t.ProjectID, project) {
			continue
		}
		if lane != "" && p.Kind == "owner" && t.Lane != lane {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(t.Title), q) && !strings.Contains(strings.ToLower(t.Description), q) {
			continue
		}
		out = append(out, t)
	}
	return out
}

type todoIn struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Note        string   `json:"note"`
	ProjectID   string   `json:"project_id"`
	Project     string   `json:"project"`
	Due         string   `json:"due"`
	Status      string   `json:"status"`
	Steps       []string `json:"steps"`
	Step        int      `json:"step"`
	DelegateID  string   `json:"delegate_id"`
}

func (s *Server) handleCreateTodo(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in todoIn
	if err := readJSON(w, r, &in); err != nil {
		s.writeErr(w, err)
		return
	}
	pid, err := s.resolveProjectRefs(strPtr(in.ProjectID, in.ProjectID != "" || in.Project != ""), strPtr(in.Project, in.Project != ""))
	if err != nil {
		s.writeErr(w, err)
		return
	}
	projectID := ""
	if pid != nil {
		projectID = *pid
	}
	// Both fields empty: resolveProjectRefs returns nil. Good.
	if in.ProjectID == "" && in.Project == "" {
		projectID = ""
	}
	status := in.Status
	if status == "" {
		status = "todo"
	}
	ct := store.CreateTodo{
		Title: in.Title, Description: in.Description, Note: in.Note,
		ProjectID: projectID, Due: in.Due, Status: status, Steps: in.Steps, Step: in.Step,
	}
	if p.Kind == "agent" {
		if status != "todo" && status != "in_progress" {
			s.writeErr(w, &store.Error{Code: store.CodeInvalid, Msg: "新建时状态只能是 todo 或 in_progress"})
			return
		}
		ct.Source = "agent"
		ct.Creator = p.Actor
		ct.DelegateID = p.Agent.ID
		ct.Lane = p.Agent.ID
	} else {
		if status != "todo" && status != "in_progress" && status != "done" {
			s.writeErr(w, &store.Error{Code: store.CodeInvalid, Msg: "状态不对"})
			return
		}
		ct.Source = "self"
		ct.Creator = p.Actor
		ct.DelegateID = in.DelegateID
	}
	t, err := s.st.CreateTodo(ct)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func strPtr(v string, ok bool) *string {
	if !ok {
		return nil
	}
	return &v
}

func (s *Server) resolveProjectRefs(id, name *string) (*string, error) {
	if id == nil && name == nil {
		return nil, nil
	}
	if name != nil && strings.TrimSpace(*name) != "" {
		if id != nil && strings.TrimSpace(*id) != "" {
			return nil, &store.Error{Code: store.CodeInvalid, Msg: "project_id 和 project 只要一个"}
		}
		p, err := s.st.ProjectByName(*name)
		if err != nil {
			return nil, err
		}
		return &p.ID, nil
	}
	if id != nil {
		pid := strings.TrimSpace(*id)
		if pid != "" {
			if _, err := s.st.GetProject(pid); err != nil {
				return nil, err
			}
		}
		return &pid, nil
	}
	empty := ""
	return &empty, nil
}

func (s *Server) handleGetTodo(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	t, err := s.st.GetTodo(r.PathValue("id"))
	if err != nil {
		s.writeErr(w, err)
		return
	}
	if p.Kind == "agent" && t.DelegateID != p.Agent.ID {
		s.writeErr(w, &store.Error{Code: store.CodeForbidden, Msg: "只能看自己泳道里的 todo"})
		return
	}
	s.writeTodoDetail(w, t)
}

func (s *Server) writeTodoDetail(w http.ResponseWriter, t store.Todo) {
	act, err := s.st.ListActivity(t.ID)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	qs, err := s.st.ListOpenQuestions()
	if err != nil {
		s.writeErr(w, err)
		return
	}
	var oq *store.Question
	for i := range qs {
		if qs[i].TodoID == t.ID {
			oq = &qs[i]
			break
		}
	}
	if act == nil {
		act = []store.Activity{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"todo": t, "activity": s.viewActivity(act), "open_question": oq})
}

type todoPatchIn struct {
	Title       *string   `json:"title"`
	Description *string   `json:"description"`
	Note        *string   `json:"note"`
	ProjectID   *string   `json:"project_id"`
	Project     *string   `json:"project"`
	Due         *string   `json:"due"`
	Status      *string   `json:"status"`
	Steps       *[]string `json:"steps"`
	Step        *int      `json:"step"`
	DelegateID  *string   `json:"delegate_id"`
}

func (s *Server) handlePatchTodo(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in todoPatchIn
	if err := readJSON(w, r, &in); err != nil {
		s.writeErr(w, err)
		return
	}
	pid, err := s.resolveProjectRefs(in.ProjectID, in.Project)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	patch := store.TodoPatch{
		Title: in.Title, Description: in.Description, Note: in.Note, Due: in.Due,
		Status: in.Status, Steps: in.Steps, Step: in.Step, ProjectID: pid,
	}
	if p.Kind == "owner" {
		patch.DelegateID = in.DelegateID
	} else if in.DelegateID != nil {
		s.writeErr(w, &store.Error{Code: store.CodeForbidden, Msg: "不能改执行人"})
		return
	}
	t, err := s.st.UpdateTodo(r.PathValue("id"), patch, p.Actor, p.Kind == "owner")
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleDeleteTodo(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	if err := s.st.DeleteTodo(r.PathValue("id"), p.Kind == "owner"); err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleComplete(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		Note string `json:"note"`
	}
	if r.ContentLength != 0 {
		if err := readJSON(w, r, &in); err != nil {
			s.writeErr(w, err)
			return
		}
	}
	t, err := s.st.CompleteTodo(r.PathValue("id"), in.Note, p.Actor, p.Kind == "owner")
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleComment(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		Body string `json:"body"`
	}
	if err := readJSON(w, r, &in); err != nil {
		s.writeErr(w, err)
		return
	}
	act, err := s.st.AddComment(r.PathValue("id"), in.Body, p.Actor, p.Kind == "owner")
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, s.viewActivity([]store.Activity{act})[0])
}

func (s *Server) handleListQuestions(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireOwner(w, r); !ok {
		return
	}
	qs, err := s.st.ListOpenQuestions()
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"questions": s.viewQuestions(qs)})
}

func (s *Server) handleAsk(w http.ResponseWriter, r *http.Request) {
	p, err := s.principal(r)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	if p.Kind != "agent" {
		if p.Kind == "anon" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "先登录"})
		} else {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "只有 Agent 能向主人提问"})
		}
		return
	}
	var in struct {
		TodoID   string   `json:"todo_id"`
		Title    string   `json:"title"`
		Question string   `json:"question"`
		Context  string   `json:"context"`
		Options  []string `json:"options"`
	}
	if err := readJSON(w, r, &in); err != nil {
		s.writeErr(w, err)
		return
	}
	q, t, err := s.st.AskOwner(store.Ask{
		TodoID: in.TodoID, Title: in.Title, Question: in.Question, Context: in.Context, Options: in.Options, Agent: p.Actor,
	})
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"question": q, "todo": t})
}

func (s *Server) handleAnswer(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireOwner(w, r)
	if !ok {
		return
	}
	var in struct {
		Option string `json:"option"`
	}
	if err := readJSON(w, r, &in); err != nil {
		s.writeErr(w, err)
		return
	}
	q, t, err := s.st.AnswerQuestion(r.PathValue("id"), in.Option, p.Actor)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"question": q, "todo": t})
}

func (s *Server) handleListInbox(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireOwner(w, r); !ok {
		return
	}
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	if status != "pending" && status != "accepted" && status != "declined" {
		s.writeErr(w, &store.Error{Code: store.CodeInvalid, Msg: "status 只能是 pending、accepted 或 declined"})
		return
	}
	items, err := s.st.ListIntakes(status)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	if items == nil {
		items = []store.Intake{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": s.viewIntakes(items)})
}

func (s *Server) handleAccept(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireOwner(w, r)
	if !ok {
		return
	}
	var in struct {
		Assignee string `json:"assignee"`
	}
	if err := readJSON(w, r, &in); err != nil {
		s.writeErr(w, err)
		return
	}
	assignee := strings.TrimSpace(in.Assignee)
	if assignee == "" || assignee == "我" || assignee == "我自己" {
		assignee = "owner"
	}
	if assignee != "owner" {
		if _, err := s.st.GetAgent(assignee); err != nil {
			// allow a name
			id, rerr := s.st.ResolveSuggest(assignee)
			if rerr != nil || id == "" {
				s.writeErr(w, &store.Error{Code: store.CodeInvalid, Msg: "交给谁：主人或一个 Agent"})
				return
			}
			assignee = id
		}
	}
	item, todo, err := s.st.AcceptIntake(r.PathValue("id"), assignee, p.Actor)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"intake": item, "todo": todo})
}

func (s *Server) handleDecline(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireOwner(w, r); !ok {
		return
	}
	item, err := s.st.DeclineIntake(r.PathValue("id"))
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"intake": item})
}

func (s *Server) intakeIntro(name string) string {
	intro, err := s.st.GetSetting("intake_intro")
	if err != nil || strings.TrimSpace(intro) == "" {
		return "我是 **" + name + "**。有事想让我做，在下面登记就行：我会看到，然后留给自己，或者交给合适的 Agent。"
	}
	return intro
}

func (s *Server) handleIntakeInfo(w http.ResponseWriter, r *http.Request) {
	name := s.ownerName()
	home, _ := s.st.GetSetting("intake_homepage")
	writeJSON(w, http.StatusOK, map[string]any{
		"name":     name,
		"intro":    s.intakeIntro(name),
		"homepage": home,
		"look":     store.GenLook(name, 0),
		"mode":     s.cfg.IntakeMode,
		"hello":    "有事找他？交给我~",
		"thanks":   "收到！我这就去送~",
	})
}

type intakeIn struct {
	Title   string `json:"title"`
	Body    string `json:"body"`
	Name    string `json:"name"`
	Contact string `json:"contact"`
	Want    string `json:"want"`
	Kind    string `json:"kind"`
	Suggest string `json:"suggest"`
	Token   string `json:"token"`
}

func (s *Server) handleIntakeSubmit(w http.ResponseWriter, r *http.Request) {
	p, err := s.principal(r)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	var in intakeIn
	if err := readJSON(w, r, &in); err != nil {
		s.writeErr(w, err)
		return
	}
	if !s.intakeAllowed(p, in.Token, r.URL.Query().Get("token")) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "需要登记 token"})
		return
	}
	via := "公开登记页"
	name := in.Name
	kind := in.Kind
	if p.Kind == "agent" {
		via = "登记 API"
		if strings.TrimSpace(name) == "" {
			name = p.Agent.Name
		}
		if kind == "" {
			kind = "agent"
		}
	} else if p.Kind == "intake" || (s.cfg.IntakeMode == "token" && s.tokenOK(in.Token, r.URL.Query().Get("token"))) {
		via = "登记 API"
	}
	suggest, err := s.st.ResolveSuggest(in.Suggest)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	item, err := s.st.SubmitIntake(store.IntakeIn{
		Title: in.Title, Body: in.Body, FromName: name, FromKind: kind,
		Contact: in.Contact, Want: in.Want, Via: via, SuggestID: suggest,
	})
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) tokenOK(body, query string) bool {
	if s.cfg.IntakeToken == "" {
		return false
	}
	return constEq(body, s.cfg.IntakeToken) || constEq(query, s.cfg.IntakeToken)
}

func (s *Server) intakeAllowed(p principal, bodyToken, queryToken string) bool {
	if p.Kind == "agent" || p.Kind == "owner" || p.Kind == "intake" {
		return true
	}
	if s.cfg.IntakeMode != "token" {
		return true
	}
	return s.tokenOK(bodyToken, queryToken)
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireOwner(w, r); !ok {
		return
	}
	ags, err := s.st.ListAgents()
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"agents": s.viewAgents(ags)})
}

func (s *Server) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireOwner(w, r); !ok {
		return
	}
	var in struct {
		Name string     `json:"name"`
		Role string     `json:"role"`
		Look store.Look `json:"look"`
	}
	if err := readJSON(w, r, &in); err != nil {
		s.writeErr(w, err)
		return
	}
	if in.Look.Ear == "" && in.Look.Color == "" && in.Look.Tool == "" {
		in.Look = store.GenLook(in.Name, 0)
	}
	ag, token, err := s.st.CreateAgent(in.Name, in.Role, in.Look)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"agent": s.viewAgent(ag), "token": token})
}

func (s *Server) handlePatchAgent(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireOwner(w, r); !ok {
		return
	}
	cur, err := s.st.GetAgent(r.PathValue("id"))
	if err != nil {
		s.writeErr(w, err)
		return
	}
	var in struct {
		Name *string     `json:"name"`
		Role *string     `json:"role"`
		Look *store.Look `json:"look"`
	}
	if err := readJSON(w, r, &in); err != nil {
		s.writeErr(w, err)
		return
	}
	name, role := cur.Name, cur.Role
	if in.Name != nil {
		name = *in.Name
	}
	if in.Role != nil {
		role = *in.Role
	}
	look := cur.Look
	set := false
	if in.Look != nil {
		look = *in.Look
		set = true
	}
	ag, err := s.st.UpdateAgent(cur.ID, name, role, look, set)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.viewAgent(ag))
}

func (s *Server) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireOwner(w, r)
	if !ok {
		return
	}
	if err := s.st.DeleteAgent(r.PathValue("id"), p.Actor); err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleRotate(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireOwner(w, r); !ok {
		return
	}
	ag, token, err := s.st.RotateToken(r.PathValue("id"))
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"agent": s.viewAgent(ag), "token": token})
}

func (s *Server) handleRevoke(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireOwner(w, r); !ok {
		return
	}
	if err := s.st.RevokeToken(r.PathValue("id")); err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireOwner(w, r); !ok {
		return
	}
	ps, err := s.st.ListProjects()
	if err != nil {
		s.writeErr(w, err)
		return
	}
	if ps == nil {
		ps = []store.Project{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": ps})
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireOwner(w, r); !ok {
		return
	}
	var in struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := readJSON(w, r, &in); err != nil {
		s.writeErr(w, err)
		return
	}
	p, err := s.st.CreateProject(in.Name, in.Color)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handlePatchProject(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireOwner(w, r); !ok {
		return
	}
	cur, err := s.st.GetProject(r.PathValue("id"))
	if err != nil {
		s.writeErr(w, err)
		return
	}
	var in struct {
		Name  *string `json:"name"`
		Color *string `json:"color"`
	}
	if err := readJSON(w, r, &in); err != nil {
		s.writeErr(w, err)
		return
	}
	name, color := cur.Name, cur.Color
	if in.Name != nil {
		name = *in.Name
	}
	if in.Color != nil {
		color = *in.Color
	}
	p, err := s.st.UpdateProject(cur.ID, name, color)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireOwner(w, r); !ok {
		return
	}
	if err := s.st.DeleteProject(r.PathValue("id")); err != nil {
		s.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

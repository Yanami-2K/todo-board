package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Yanami-2K/todo-board/internal/store"
)

func (s *Server) handleMCPGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", "POST, DELETE")
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "这个服务用 JSON 响应，不提供 SSE 流"})
}

func (s *Server) handleMCPDelete(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

type rpcRequest struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	p, err := s.mcpPrincipal(r)
	if err != nil {
		s.writeErr(w, err)
		return
	}
	var req rpcRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := dec.Decode(&req); err != nil {
		writeRPCError(w, nil, -32700, "请求体不是合法的 JSON")
		return
	}
	sid := r.Header.Get("Mcp-Session-Id")
	if sid == "" {
		sid = newSession()
	}
	w.Header().Set("Mcp-Session-Id", sid)
	if req.isNote() {
		if req.Method == "notifications/initialized" || req.Method == "notifications/cancelled" {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		return
	}
	switch req.Method {
	case "initialize":
		writeRPC(w, req.ID, map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]any{"name": "todo-board", "version": version},
		})
	case "ping":
		writeRPC(w, req.ID, json.RawMessage(`{}`))
	case "tools/list":
		writeRPC(w, req.ID, map[string]any{"tools": s.toolDefs(p)})
	case "tools/call":
		s.callTool(w, req, p)
	default:
		writeRPCError(w, req.ID, -32601, "不认识的方法："+req.Method)
	}
}

func (req rpcRequest) isNote() bool {
	return req.ID == nil || string(*req.ID) == "null" || len(*req.ID) == 0
}

func (s *Server) mcpPrincipal(r *http.Request) (principal, error) {
	h := r.Header.Get("Authorization")
	if h == "" {
		if s.cfg.IntakeMode != "token" {
			return principal{Kind: "anon"}, nil
		}
		return principal{}, &store.Error{Code: "unauthorized", Msg: "需要 Bearer token"}
	}
	return s.principal(r)
}

func newSession() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func writeRPC(w http.ResponseWriter, id *json.RawMessage, result any) {
	rawID := json.RawMessage(`null`)
	if id != nil {
		rawID = *id
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Result  any             `json:"result"`
	}{"2.0", rawID, result})
}

func writeRPCError(w http.ResponseWriter, id *json.RawMessage, code int, msg string) {
	rawID := json.RawMessage(`null`)
	if id != nil {
		rawID = *id
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Error   rpcError        `json:"error"`
	}{"2.0", rawID, rpcError{Code: code, Message: msg}})
}

func toolResult(v any, isErr bool) map[string]any {
	var text string
	switch t := v.(type) {
	case string:
		text = t
	default:
		b, _ := json.MarshalIndent(v, "", "  ")
		text = string(b)
	}
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
		"isError": isErr,
	}
}

type toolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

func objSchema(props map[string]any, required ...string) map[string]any {
	s := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

func (s *Server) toolDefs(p principal) []toolDef {
	status := map[string]any{"type": "string", "enum": []string{"todo", "in_progress", "waiting", "done"}}
	list := toolDef{Name: "list_todos", Description: "列出当前身份看得到的 todo。Agent 只能看到自己泳道。可按 status 或 id 过滤。",
		InputSchema: objSchema(map[string]any{
			"status": status,
			"id":     map[string]any{"type": "string"},
			"limit":  map[string]any{"type": "integer", "minimum": 1, "maximum": 100},
		})}
	create := toolDef{Name: "create_todo", Description: "在自己的泳道新建一条 todo。你是创建者也是执行者。做完用 complete_todo，没有单独的验收。",
		InputSchema: objSchema(map[string]any{
			"title":       map[string]any{"type": "string"},
			"description": map[string]any{"type": "string"},
			"project":     map[string]any{"type": "string", "description": "已有项目的名字"},
			"due":         map[string]any{"type": "string"},
			"status":      map[string]any{"type": "string", "enum": []string{"todo", "in_progress"}},
			"steps":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		}, "title")}
	update := toolDef{Name: "update_todo", Description: "修改自己的 todo。认领就是把 status 改成 in_progress。用 note 写进度，用 steps 和 step（从 1 开始）报告步骤。",
		InputSchema: objSchema(map[string]any{
			"id":          map[string]any{"type": "string"},
			"title":       map[string]any{"type": "string"},
			"description": map[string]any{"type": "string"},
			"project":     map[string]any{"type": "string", "description": "项目名，空字符串表示移出项目"},
			"due":         map[string]any{"type": "string"},
			"status":      map[string]any{"type": "string", "enum": []string{"todo", "in_progress"}},
			"steps":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"step":        map[string]any{"type": "integer"},
			"note":        map[string]any{"type": "string", "description": "追加一条进度，不是改标题"},
		}, "id")}
	complete := toolDef{Name: "complete_todo", Description: "把自己的 todo 标为完成。没有单独的验收步骤。",
		InputSchema: objSchema(map[string]any{
			"id":   map[string]any{"type": "string"},
			"note": map[string]any{"type": "string"},
		}, "id")}
	ask := toolDef{Name: "ask_owner", Description: "向主人提一个需要拍板的问题。todo 会变成 waiting，出现在首页「待确认」。给出 2 到 4 个短选项。不要自己猜。",
		InputSchema: objSchema(map[string]any{
			"todo_id":  map[string]any{"type": "string"},
			"title":    map[string]any{"type": "string", "description": "没有 todo_id 时用来新建一条 todo"},
			"question": map[string]any{"type": "string"},
			"context":  map[string]any{"type": "string"},
			"options":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		}, "question")}
	submit := toolDef{Name: "submit_to_intake", Description: "给主人登记一件事。它先进收件箱，接受后才上看板。给自己建任务用 create_todo，不要用这个。",
		InputSchema: objSchema(map[string]any{
			"title":       map[string]any{"type": "string"},
			"description": map[string]any{"type": "string"},
			"name":        map[string]any{"type": "string"},
			"contact":     map[string]any{"type": "string"},
			"want":        map[string]any{"type": "string"},
			"kind":        map[string]any{"type": "string", "enum": []string{"human", "agent"}},
			"suggest":     map[string]any{"type": "string", "description": "建议交给哪个 Agent，名字"},
		}, "title")}
	if p.Kind == "agent" {
		return []toolDef{list, create, update, complete, ask, submit}
	}
	return []toolDef{submit}
}

func (s *Server) callTool(w http.ResponseWriter, req rpcRequest, p principal) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			writeRPCError(w, req.ID, -32602, "参数不对")
			return
		}
	}
	if len(params.Arguments) == 0 || string(params.Arguments) == "null" {
		params.Arguments = []byte(`{}`)
	}
	if p.Kind != "agent" && params.Name != "submit_to_intake" {
		writeRPC(w, req.ID, toolResult("这个身份只能用 submit_to_intake", true))
		return
	}
	var result any
	var err error
	switch params.Name {
	case "list_todos":
		result, err = s.mcpList(p, params.Arguments)
	case "create_todo":
		result, err = s.mcpCreate(p, params.Arguments)
	case "update_todo":
		result, err = s.mcpUpdate(p, params.Arguments)
	case "complete_todo":
		result, err = s.mcpComplete(p, params.Arguments)
	case "ask_owner":
		result, err = s.mcpAsk(p, params.Arguments)
	case "submit_to_intake":
		result, err = s.mcpSubmit(p, params.Arguments)
	default:
		writeRPC(w, req.ID, toolResult("没有这个工具", true))
		return
	}
	if err != nil {
		if e := store.AsError(err); e != nil {
			writeRPC(w, req.ID, toolResult(e.Msg, true))
			return
		}
		writeRPC(w, req.ID, toolResult("服务器出错了", true))
		return
	}
	writeRPC(w, req.ID, toolResult(result, false))
}

func (s *Server) mcpList(p principal, raw json.RawMessage) (any, error) {
	var in struct {
		Status string `json:"status"`
		ID     string `json:"id"`
		Limit  int    `json:"limit"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, &store.Error{Code: store.CodeInvalid, Msg: "参数不对"}
	}
	if in.Limit <= 0 || in.Limit > 100 {
		in.Limit = 50
	}
	all, err := s.st.ListTodos()
	if err != nil {
		return nil, err
	}
	qs, err := s.st.ListOpenQuestions()
	if err != nil {
		return nil, err
	}
	ps, err := s.st.ListProjects()
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	for _, pr := range ps {
		names[pr.ID] = pr.Name
	}
	var out []any
	for _, t := range all {
		if p.Kind == "agent" && t.DelegateID != p.Agent.ID {
			continue
		}
		if in.Status != "" && t.Status != in.Status {
			continue
		}
		if in.ID != "" && t.ID != in.ID {
			continue
		}
		act, err := s.st.ListActivity(t.ID)
		if err != nil {
			return nil, err
		}
		if len(act) > 8 {
			act = act[len(act)-8:]
		}
		var oq any
		for _, q := range qs {
			if q.TodoID == t.ID {
				oq = map[string]any{"id": q.ID, "text": q.Text, "context": q.Context, "options": q.Options}
				break
			}
		}
		brief := make([]map[string]string, 0, len(act))
		for _, a := range act {
			brief = append(brief, map[string]string{"actor": a.ActorName, "text": a.Text, "at": s.ago(a.CreatedAt)})
		}
		out = append(out, map[string]any{
			"id": t.ID, "title": t.Title, "description": t.Description, "note": t.Note,
			"status": t.Status, "project": names[t.ProjectID], "due": t.Due,
			"steps": t.Steps, "step": t.Step, "open_question": oq, "activity": brief,
		})
		if len(out) >= in.Limit {
			break
		}
	}
	if out == nil {
		out = []any{}
	}
	return map[string]any{"todos": out}, nil
}

func (s *Server) mcpCreate(p principal, raw json.RawMessage) (any, error) {
	var in struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Project     string   `json:"project"`
		Due         string   `json:"due"`
		Status      string   `json:"status"`
		Steps       []string `json:"steps"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, &store.Error{Code: store.CodeInvalid, Msg: "参数不对"}
	}
	pid := ""
	if strings.TrimSpace(in.Project) != "" {
		pr, err := s.st.ProjectByName(in.Project)
		if err != nil {
			return nil, err
		}
		pid = pr.ID
	}
	t, err := s.st.CreateTodo(store.CreateTodo{
		Title: in.Title, Description: in.Description, ProjectID: pid, Due: in.Due,
		Status: in.Status, Steps: in.Steps, Source: "agent", Creator: p.Actor,
		DelegateID: p.Agent.ID, Lane: p.Agent.ID,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": t.ID, "title": t.Title, "status": t.Status}, nil
}

func (s *Server) mcpUpdate(p principal, raw json.RawMessage) (any, error) {
	var in struct {
		ID          string    `json:"id"`
		Title       *string   `json:"title"`
		Description *string   `json:"description"`
		Project     *string   `json:"project"`
		Due         *string   `json:"due"`
		Status      *string   `json:"status"`
		Steps       *[]string `json:"steps"`
		Step        *int      `json:"step"`
		Note        *string   `json:"note"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, &store.Error{Code: store.CodeInvalid, Msg: "参数不对"}
	}
	pid, err := s.resolveProjectRefs(nil, in.Project)
	if err != nil {
		return nil, err
	}
	progress := ""
	if in.Note != nil {
		progress = *in.Note
	}
	t, err := s.st.UpdateTodo(in.ID, store.TodoPatch{
		Title: in.Title, Description: in.Description, ProjectID: pid, Due: in.Due,
		Status: in.Status, Steps: in.Steps, Step: in.Step, Progress: progress,
	}, p.Actor, false)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": t.ID, "title": t.Title, "status": t.Status, "step": t.Step}, nil
}

func (s *Server) mcpComplete(p principal, raw json.RawMessage) (any, error) {
	var in struct {
		ID   string `json:"id"`
		Note string `json:"note"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, &store.Error{Code: store.CodeInvalid, Msg: "参数不对"}
	}
	t, err := s.st.CompleteTodo(in.ID, in.Note, p.Actor, false)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": t.ID, "status": t.Status}, nil
}

func (s *Server) mcpAsk(p principal, raw json.RawMessage) (any, error) {
	var in struct {
		TodoID   string   `json:"todo_id"`
		Title    string   `json:"title"`
		Question string   `json:"question"`
		Context  string   `json:"context"`
		Options  []string `json:"options"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, &store.Error{Code: store.CodeInvalid, Msg: "参数不对"}
	}
	q, t, err := s.st.AskOwner(store.Ask{
		TodoID: in.TodoID, Title: in.Title, Question: in.Question, Context: in.Context, Options: in.Options, Agent: p.Actor,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"question_id": q.ID, "todo_id": t.ID, "status": t.Status, "options": q.Options}, nil
}

func (s *Server) mcpSubmit(p principal, raw json.RawMessage) (any, error) {
	if p.Kind != "agent" && p.Kind != "intake" && !(p.Kind == "anon" && s.cfg.IntakeMode != "token") {
		return nil, &store.Error{Code: "unauthorized", Msg: "需要登记 token"}
	}
	var in intakeIn
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, &store.Error{Code: store.CodeInvalid, Msg: "参数不对"}
	}
	via := "登记 API"
	name := in.Name
	kind := in.Kind
	if p.Kind == "agent" {
		if strings.TrimSpace(name) == "" {
			name = p.Agent.Name
		}
		if kind == "" {
			kind = "agent"
		}
	}
	suggest, err := s.st.ResolveSuggest(in.Suggest)
	if err != nil {
		return nil, err
	}
	item, err := s.st.SubmitIntake(store.IntakeIn{
		Title: in.Title, Body: in.Body, FromName: name, FromKind: kind,
		Contact: in.Contact, Want: in.Want, Via: via, SuggestID: suggest,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": item.ID, "status": item.Status, "title": item.Title}, nil
}

package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Yanami-2K/todo-board/internal/store"
)

type env struct {
	t      *testing.T
	srv    *httptest.Server
	st     *store.Store
	cookie string
}

func newEnv(t *testing.T, cfg Config) *env {
	t.Helper()
	if cfg.OwnerPassword == "" {
		cfg.OwnerPassword = "secret"
	}
	if cfg.OwnerName == "" {
		cfg.OwnerName = "测试"
	}
	if cfg.BcryptCost == 0 {
		cfg.BcryptCost = 4
	}
	if cfg.Location == nil {
		cfg.Location = time.FixedZone("CST", 8*3600)
	}
	if cfg.IntakeMode == "" {
		cfg.IntakeMode = "open"
	}
	st, err := store.Open(filepath.Join(t.TempDir(), "todo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if _, err := EnsureOwner(st, cfg); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(New(cfg, st).Handler())
	t.Cleanup(srv.Close)
	return &env{t: t, srv: srv, st: st}
}

func (e *env) do(method, path string, body any, token string) (int, map[string]any) {
	e.t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			e.t.Fatal(err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, e.srv.URL+path, rdr)
	if err != nil {
		e.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if e.cookie != "" && token == "" {
		req.AddCookie(&http.Cookie{Name: cookieName, Value: e.cookie})
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		out = map[string]any{"_raw": string(raw)}
	}
	return res.StatusCode, out
}

func (e *env) login() {
	e.t.Helper()
	req, _ := http.NewRequest(http.MethodPost, e.srv.URL+"/api/login", strings.NewReader(`{"password":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		b, _ := io.ReadAll(res.Body)
		e.t.Fatalf("login %d %s", res.StatusCode, b)
	}
	for _, c := range res.Cookies() {
		if c.Name == cookieName {
			e.cookie = c.Value
		}
	}
	if e.cookie == "" {
		e.t.Fatal("no cookie")
	}
}

func TestCoreFlow(t *testing.T) {
	e := newEnv(t, Config{})
	if code, _ := e.do(http.MethodGet, "/api/board", nil, ""); code != 401 {
		t.Fatalf("board without login = %d", code)
	}
	if code, body := e.do(http.MethodPost, "/api/login", map[string]string{"password": "nope"}, ""); code != 401 || body["error"] != "密码不对" {
		t.Fatalf("bad login %+v %d", body, code)
	}
	e.login()

	code, proj := e.do(http.MethodPost, "/api/projects", map[string]string{"name": "基建", "color": "mint"}, "")
	if code != 201 {
		t.Fatalf("project %d %+v", code, proj)
	}
	pid := proj["id"].(string)

	code, ag := e.do(http.MethodPost, "/api/agents", map[string]any{"name": "builder", "role": "构建"}, "")
	if code != 201 {
		t.Fatalf("agent %d %+v", code, ag)
	}
	token := ag["token"].(string)
	agent := ag["agent"].(map[string]any)
	aid := agent["id"].(string)
	if !strings.HasPrefix(token, "tb_") || agent["token_prefix"] == "" {
		t.Fatalf("token not returned once: %+v", ag)
	}

	code, mine := e.do(http.MethodPost, "/api/todos", map[string]any{"title": "主人自己的事", "project_id": pid}, "")
	if code != 201 || mine["lane"] != "self" {
		t.Fatalf("owner todo %d %+v", code, mine)
	}
	ownerID := mine["id"].(string)

	code, hidden := e.do(http.MethodGet, "/api/todos", nil, token)
	if code != 200 {
		t.Fatal(code, hidden)
	}
	if n := len(hidden["todos"].([]any)); n != 0 {
		t.Fatalf("agent saw %d todos, want 0", n)
	}
	if code, _ := e.do(http.MethodGet, "/api/todos/"+ownerID, nil, token); code != 403 {
		t.Fatalf("agent read owner todo %d", code)
	}

	code, created := e.do(http.MethodPost, "/api/todos", map[string]any{
		"title": "打镜像", "description": "ubuntu", "project": "基建", "status": "in_progress",
		"steps": []string{"拉基础层", "安装", "冒烟"}, "step": 1,
	}, token)
	if code != 201 {
		t.Fatalf("agent create %d %+v", code, created)
	}
	tid := created["id"].(string)
	if created["delegate_id"] != aid || created["source"] != "agent" || created["project_id"] != pid {
		t.Fatalf("created shape %+v", created)
	}

	code, asked := e.do(http.MethodPost, "/api/questions", map[string]any{
		"todo_id": tid, "question": "先修还是先忽略？", "context": "两个高危",
		"options": []string{"先忽略", "现在修"},
	}, token)
	if code != 201 {
		t.Fatalf("ask %d %+v", code, asked)
	}
	q := asked["question"].(map[string]any)
	qid := q["id"].(string)

	code, board := e.do(http.MethodGet, "/api/board", nil, "")
	if code != 200 {
		t.Fatal(code, board)
	}
	if qs := board["questions"].([]any); len(qs) != 1 {
		t.Fatalf("questions %+v", board["questions"])
	}

	if code, _ := e.do(http.MethodPost, "/api/questions/"+qid+"/answer", map[string]string{"option": "不在列表"}, ""); code != 400 {
		t.Fatal("bad option")
	}
	code, answered := e.do(http.MethodPost, "/api/questions/"+qid+"/answer", map[string]string{"option": "现在修"}, "")
	if code != 200 {
		t.Fatalf("answer %d %+v", code, answered)
	}
	todo := answered["todo"].(map[string]any)
	if todo["status"] != "in_progress" {
		t.Fatalf("status after answer %+v", todo["status"])
	}
	code, board = e.do(http.MethodGet, "/api/board", nil, "")
	if len(board["questions"].([]any)) != 0 {
		t.Fatalf("question still open")
	}

	code, got := e.do(http.MethodGet, "/api/todos/"+tid, nil, token)
	if code != 200 {
		t.Fatal(got)
	}
	act := got["activity"].([]any)
	joined := ""
	for _, row := range act {
		joined += row.(map[string]any)["text"].(string) + "\n"
	}
	if !strings.Contains(joined, "回复：现在修") || !strings.Contains(joined, "builder") && !strings.Contains(joined, "向你确认") {
		t.Fatalf("activity %s", joined)
	}
	var sawAgent bool
	for _, row := range act {
		m := row.(map[string]any)
		if m["actor_name"] == "builder" {
			sawAgent = true
		}
	}
	if !sawAgent {
		t.Fatal("activity missing agent identity", joined)
	}

	code, done := e.do(http.MethodPost, "/api/todos/"+tid+"/complete", map[string]string{"note": "镜像好了"}, token)
	if code != 200 || done["status"] != "done" {
		t.Fatalf("complete %d %+v", code, done)
	}

	code, intake := e.do(http.MethodPost, "/api/intake", map[string]any{
		"title": "帮我看一个改动", "body": "链接在评论里", "name": "林乔", "contact": "lin@example.com",
		"want": "周四前", "kind": "human", "suggest": "builder",
	}, "")
	if code != 201 || intake["status"] != "pending" || intake["via"] != "公开登记页" {
		t.Fatalf("intake %d %+v", code, intake)
	}
	rid := intake["id"].(string)
	if intake["suggest_id"] != aid {
		t.Fatalf("suggest %+v", intake["suggest_id"])
	}

	code, acc := e.do(http.MethodPost, "/api/inbox/"+rid+"/accept", map[string]string{"assignee": aid}, "")
	if code != 200 {
		t.Fatalf("accept %d %+v", code, acc)
	}
	accepted := acc["todo"].(map[string]any)
	if accepted["delegate_id"] != aid || accepted["source"] != "intake_form" {
		t.Fatalf("accepted %+v", accepted)
	}
	code, list := e.do(http.MethodGet, "/api/todos?status=todo", nil, token)
	found := false
	for _, row := range list["todos"].([]any) {
		if row.(map[string]any)["id"] == accepted["id"] {
			found = true
		}
	}
	if !found {
		t.Fatalf("agent did not see accepted todo %+v", list)
	}

	code, intake2 := e.do(http.MethodPost, "/api/intake", map[string]string{"title": "另一件"}, "")
	if code != 201 {
		t.Fatal(intake2)
	}
	code, dec := e.do(http.MethodPost, "/api/inbox/"+intake2["id"].(string)+"/decline", map[string]string{}, "")
	if code != 200 || dec["intake"].(map[string]any)["status"] != "declined" {
		t.Fatalf("decline %d %+v", code, dec)
	}

	if code, _ := e.do(http.MethodPost, "/api/inbox/"+intake2["id"].(string)+"/accept", map[string]string{"assignee": "owner"}, token); code != 403 {
		t.Fatal("agent accepted inbox")
	}

	code, rot := e.do(http.MethodPost, "/api/agents/"+aid+"/revoke", map[string]string{}, "")
	if code != 200 {
		t.Fatal(rot)
	}
	if code, _ := e.do(http.MethodGet, "/api/todos", nil, token); code != 401 {
		t.Fatal("revoked token still works")
	}
	code, rot = e.do(http.MethodPost, "/api/agents/"+aid+"/token", map[string]string{}, "")
	if code != 200 {
		t.Fatal(rot)
	}
	token2 := rot["token"].(string)
	if code, _ := e.do(http.MethodGet, "/api/todos", nil, token2); code != 200 {
		t.Fatal("rotated token")
	}

	code, renamed := e.do(http.MethodPatch, "/api/projects/"+pid, map[string]string{"name": "基建二期", "color": "blue"}, "")
	if code != 200 || renamed["name"] != "基建二期" {
		t.Fatalf("rename %d %+v", code, renamed)
	}
	if code, _ := e.do(http.MethodDelete, "/api/projects/"+pid, nil, ""); code != 200 {
		t.Fatal("delete project")
	}
	code, after := e.do(http.MethodGet, "/api/todos/"+ownerID, nil, "")
	if after["todo"].(map[string]any)["project_id"] != "" && after["todo"].(map[string]any)["project_id"] != nil {
		t.Fatalf("project not cleared %+v", after["todo"])
	}

	if code, dup := e.do(http.MethodPost, "/api/agents", map[string]string{"name": "Builder"}, ""); code != 409 {
		t.Fatalf("dup %d %+v", code, dup)
	}
}

func TestMCPAndIntakeToken(t *testing.T) {
	e := newEnv(t, Config{IntakeMode: "token", IntakeToken: "ext_secret"})
	e.login()
	_, ag := e.do(http.MethodPost, "/api/agents", map[string]string{"name": "runner"}, "")
	token := ag["token"].(string)

	saved := e.cookie
	e.cookie = ""
	code, _ := e.do(http.MethodPost, "/api/intake", map[string]string{"title": "不该进"}, "")
	e.cookie = saved
	if code != 401 {
		t.Fatalf("open intake in token mode %d", code)
	}
	code, item := e.do(http.MethodPost, "/api/intake", map[string]string{"title": "用 token 登记", "name": "外部"}, "ext_secret")
	if code != 201 || item["via"] != "登记 API" {
		t.Fatalf("token intake %d %+v", code, item)
	}

	initBody := map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{"protocolVersion": "2025-03-26", "capabilities": map[string]any{}, "clientInfo": map[string]any{"name": "test", "version": "0"}},
	}
	code, res := e.mcp(token, initBody)
	if code != 200 || res["result"].(map[string]any)["protocolVersion"] != "2025-03-26" {
		t.Fatalf("initialize %d %+v", code, res)
	}
	code, res = e.mcp(token, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/list"})
	tools := res["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 6 {
		t.Fatalf("tools %+v", tools)
	}
	code, res = e.mcp(token, map[string]any{"jsonrpc": "2.0", "id": 3, "method": "tools/call", "params": map[string]any{
		"name": "create_todo", "arguments": map[string]any{"title": "跑评测", "status": "todo"},
	}})
	if code != 200 || res["result"].(map[string]any)["isError"] == true {
		t.Fatalf("mcp create %d %+v", code, res)
	}
	text := res["result"].(map[string]any)["content"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "跑评测") {
		t.Fatal(text)
	}
	var created map[string]any
	if err := json.Unmarshal([]byte(text), &created); err != nil {
		t.Fatal(err)
	}
	id := created["id"].(string)
	code, res = e.mcp(token, map[string]any{"jsonrpc": "2.0", "id": 4, "method": "tools/call", "params": map[string]any{
		"name": "ask_owner", "arguments": map[string]any{"todo_id": id, "question": "预算不够，还跑吗？", "options": []string{"停", "继续"}},
	}})
	if code != 200 || res["result"].(map[string]any)["isError"] == true {
		t.Fatalf("mcp ask %d %+v", code, res)
	}
	code, board := e.do(http.MethodGet, "/api/board", nil, "")
	if len(board["questions"].([]any)) != 1 {
		t.Fatalf("board questions %+v", board["questions"])
	}
	code, res = e.mcp(token, map[string]any{"jsonrpc": "2.0", "id": 5, "method": "tools/call", "params": map[string]any{
		"name": "submit_to_intake", "arguments": map[string]any{"title": "请主人看一眼配额"},
	}})
	if code != 200 || res["result"].(map[string]any)["isError"] == true {
		t.Fatalf("mcp intake %d %+v", code, res)
	}
	code, inbox := e.do(http.MethodGet, "/api/inbox", nil, "")
	if len(inbox["items"].([]any)) < 2 {
		t.Fatalf("inbox %+v", inbox)
	}

	code, res = e.mcp("", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"})
	if code != 401 {
		t.Fatalf("anon mcp in token mode %d %+v", code, res)
	}

	open := newEnv(t, Config{IntakeMode: "open"})
	code, res = open.mcp("", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
		"name": "submit_to_intake", "arguments": map[string]any{"title": "匿名登记", "name": "路人"},
	}})
	if code != 200 || res["result"].(map[string]any)["isError"] == true {
		t.Fatalf("anon submit %d %+v", code, res)
	}
	code, res = open.mcp("", map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": map[string]any{
		"name": "create_todo", "arguments": map[string]any{"title": "不该成功"},
	}})
	if res["result"].(map[string]any)["isError"] != true {
		t.Fatalf("anon create should fail %+v", res)
	}
}

func (e *env) mcp(token string, body map[string]any) (int, map[string]any) {
	e.t.Helper()
	return e.do(http.MethodPost, "/mcp", body, token)
}

func TestHealthAndSpec(t *testing.T) {
	e := newEnv(t, Config{})
	code, body := e.do(http.MethodGet, "/api/health", nil, "")
	if code != 200 || body["ok"] != true {
		t.Fatal(body)
	}
	for _, p := range []string{"/", "/login", "/intake"} {
		res, err := http.Get(e.srv.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != 200 || !bytes.Contains(b, []byte("todo")) {
			t.Fatalf("%s %d %s", p, res.StatusCode, b)
		}
	}
	res, err := http.Get(e.srv.URL + "/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if !bytes.Contains(b, []byte("openapi:")) {
		t.Fatalf("spec %s", b)
	}
}

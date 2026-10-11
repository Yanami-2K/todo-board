package store

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

func (s *Store) CreateAgent(name, role string, look Look) (Agent, string, error) {
	name, err := trimReq(name, 24, "名字", true)
	if err != nil {
		return Agent{}, "", err
	}
	role, err = trimReq(role, 30, "职责", false)
	if err != nil {
		return Agent{}, "", err
	}
	look, err = NormalizeLook(look)
	if err != nil {
		return Agent{}, "", err
	}
	raw, err := randomToken("tb_", 24)
	if err != nil {
		return Agent{}, "", err
	}
	var a Agent
	err = s.withTx(func(tx *sql.Tx) error {
		taken, err := foldTaken(tx, `SELECT id, name FROM agents`, name, "")
		if err != nil {
			return err
		}
		if taken {
			return conflict("已经有叫「" + name + "」的了")
		}
		id, err := nextID(tx, "agent", "A-")
		if err != nil {
			return err
		}
		now := nowUTC()
		lookJSON, _ := json.Marshal(look)
		prefix := tokenPrefix(raw)
		_, err = tx.Exec(`INSERT INTO agents(id, name, role, look, token_hash, token_prefix, created_at, revoked)
			VALUES(?, ?, ?, ?, ?, ?, ?, 0)`, id, name, role, string(lookJSON), HashToken(raw), prefix, fmtT(now))
		if err != nil {
			return err
		}
		a = Agent{ID: id, Name: name, Role: role, Look: look, TokenPrefix: prefix, CreatedAt: now}
		return nil
	})
	return a, raw, err
}

func tokenPrefix(raw string) string {
	if len(raw) <= 7 {
		return raw
	}
	return raw[:7]
}

func (s *Store) UpdateAgent(id, name, role string, look Look, setLook bool) (Agent, error) {
	var out Agent
	err := s.withTx(func(tx *sql.Tx) error {
		a, err := getAgentTx(tx, id)
		if err != nil {
			return err
		}
		if name != "" {
			name, err = trimReq(name, 24, "名字", true)
			if err != nil {
				return err
			}
			taken, err := foldTaken(tx, `SELECT id, name FROM agents`, name, id)
			if err != nil {
				return err
			}
			if taken {
				return conflict("已经有叫「" + name + "」的了")
			}
			a.Name = name
		}
		if role != "" || role == "" {
			// caller passes the new role, including empty to clear
		}
		role, err = trimReq(role, 30, "职责", false)
		if err != nil {
			return err
		}
		a.Role = role
		if setLook {
			look, err = NormalizeLook(look)
			if err != nil {
				return err
			}
			a.Look = look
		}
		lookJSON, _ := json.Marshal(a.Look)
		_, err = tx.Exec(`UPDATE agents SET name=?, role=?, look=? WHERE id=?`, a.Name, a.Role, string(lookJSON), id)
		out = a
		return err
	})
	return out, err
}

func (s *Store) DeleteAgent(id string, owner Actor) error {
	return s.withTx(func(tx *sql.Tx) error {
		a, err := getAgentTx(tx, id)
		if err != nil {
			return err
		}
		rows, err := tx.Query(`SELECT id, lane, delegate_id FROM todos WHERE delegate_id = ? OR lane = ?`, id, id)
		if err != nil {
			return err
		}
		type row struct{ id, lane, del string }
		var todos []row
		for rows.Next() {
			var r row
			var del sql.NullString
			if err := rows.Scan(&r.id, &r.lane, &del); err != nil {
				rows.Close()
				return err
			}
			r.del = del.String
			todos = append(todos, r)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		now := fmtT(nowUTC())
		for _, t := range todos {
			lane := t.lane
			if lane == id {
				lane = "self"
			}
			_, err = tx.Exec(`UPDATE todos SET delegate_id = NULL, lane = ?, updated_at = ? WHERE id = ?`, lane, now, t.id)
			if err != nil {
				return err
			}
			if err := addActivity(tx, t.id, owner, "c", "执行者 "+a.Name+" 已删除，转回给我"); err != nil {
				return err
			}
		}
		_, err = tx.Exec(`UPDATE questions SET answered_at = ?, answer = ? WHERE agent_id = ? AND answered_at IS NULL`,
			now, "（Agent 已删除）", id)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`UPDATE intakes SET suggest_id = '' WHERE suggest_id = ? AND status = 'pending'`, id)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`DELETE FROM agents WHERE id = ?`, id)
		return err
	})
}

func (s *Store) RotateToken(id string) (Agent, string, error) {
	raw, err := randomToken("tb_", 24)
	if err != nil {
		return Agent{}, "", err
	}
	var a Agent
	err = s.withTx(func(tx *sql.Tx) error {
		cur, err := getAgentTx(tx, id)
		if err != nil {
			return err
		}
		prefix := tokenPrefix(raw)
		_, err = tx.Exec(`UPDATE agents SET token_hash = ?, token_prefix = ?, revoked = 0 WHERE id = ?`,
			HashToken(raw), prefix, id)
		if err != nil {
			return err
		}
		cur.TokenPrefix = prefix
		cur.Revoked = false
		a = cur
		return nil
	})
	return a, raw, err
}

func (s *Store) RevokeToken(id string) error {
	return s.withTx(func(tx *sql.Tx) error {
		if _, err := getAgentTx(tx, id); err != nil {
			return err
		}
		_, err := tx.Exec(`UPDATE agents SET token_hash = NULL, revoked = 1, token_prefix = '' WHERE id = ?`, id)
		return err
	})
}

func (s *Store) AgentByToken(token string) (Agent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row := s.db.QueryRow(`SELECT id, name, role, look, token_prefix, created_at, last_seen_at, revoked
		FROM agents WHERE token_hash = ? AND revoked = 0`, HashToken(token))
	a, err := scanAgent(row)
	if err == sql.ErrNoRows {
		return Agent{}, notFound("token 无效或已吊销")
	}
	return a, err
}

func (s *Store) TouchAgent(id string) error {
	return s.withTx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`UPDATE agents SET last_seen_at = ? WHERE id = ?`, fmtT(nowUTC()), id)
		return err
	})
}

func (s *Store) ListAgents() ([]Agent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT id, name, role, look, token_prefix, created_at, last_seen_at, revoked
		FROM agents ORDER BY created_at ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Agent
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) GetAgent(id string) (Agent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, err := getAgent(s.db, id)
	return a, err
}

type queryer interface {
	QueryRow(string, ...any) *sql.Row
}

func getAgent(q queryer, id string) (Agent, error) {
	a, err := scanAgent(q.QueryRow(`SELECT id, name, role, look, token_prefix, created_at, last_seen_at, revoked FROM agents WHERE id = ?`, id))
	if err == sql.ErrNoRows {
		return Agent{}, notFound("找不到这个 Agent")
	}
	return a, err
}

func getAgentTx(tx *sql.Tx, id string) (Agent, error) { return getAgent(tx, id) }

type scanner interface {
	Scan(dest ...any) error
}

func scanAgent(sc scanner) (Agent, error) {
	var a Agent
	var look, created string
	var seen sql.NullString
	var revoked int
	err := sc.Scan(&a.ID, &a.Name, &a.Role, &look, &a.TokenPrefix, &created, &seen, &revoked)
	if err != nil {
		return Agent{}, err
	}
	_ = json.Unmarshal([]byte(look), &a.Look)
	a.CreatedAt = parseT(created)
	if seen.Valid && seen.String != "" {
		t := parseT(seen.String)
		a.LastSeenAt = &t
	}
	a.Revoked = revoked == 1
	return a, nil
}

func (s *Store) CreateProject(name, color string) (Project, error) {
	name, err := trimReq(name, 24, "项目名", true)
	if err != nil {
		return Project{}, err
	}
	if color == "" {
		color = colors[int(HashName(name))%len(colors)]
	}
	if !ValidColor(color) {
		return Project{}, invalid("颜色不对")
	}
	var p Project
	err = s.withTx(func(tx *sql.Tx) error {
		taken, err := foldTaken(tx, `SELECT id, name FROM projects`, name, "")
		if err != nil {
			return err
		}
		if taken {
			return conflict("已经有这个项目了")
		}
		id, err := nextID(tx, "project", "P-")
		if err != nil {
			return err
		}
		now := nowUTC()
		_, err = tx.Exec(`INSERT INTO projects(id, name, color, created_at) VALUES(?, ?, ?, ?)`, id, name, color, fmtT(now))
		if err != nil {
			return err
		}
		p = Project{ID: id, Name: name, Color: color, CreatedAt: now}
		return nil
	})
	return p, err
}

func (s *Store) UpdateProject(id, name, color string) (Project, error) {
	name, err := trimReq(name, 24, "项目名", true)
	if err != nil {
		return Project{}, err
	}
	if !ValidColor(color) {
		return Project{}, invalid("颜色不对")
	}
	var p Project
	err = s.withTx(func(tx *sql.Tx) error {
		cur, err := getProjectTx(tx, id)
		if err != nil {
			return err
		}
		taken, err := foldTaken(tx, `SELECT id, name FROM projects`, name, id)
		if err != nil {
			return err
		}
		if taken {
			return conflict("已经有这个项目了")
		}
		_, err = tx.Exec(`UPDATE projects SET name = ?, color = ? WHERE id = ?`, name, color, id)
		if err != nil {
			return err
		}
		cur.Name = name
		cur.Color = color
		p = cur
		return nil
	})
	return p, err
}

func (s *Store) DeleteProject(id string) error {
	return s.withTx(func(tx *sql.Tx) error {
		if _, err := getProjectTx(tx, id); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE todos SET project_id = NULL WHERE project_id = ?`, id); err != nil {
			return err
		}
		_, err := tx.Exec(`DELETE FROM projects WHERE id = ?`, id)
		return err
	})
}

func (s *Store) ListProjects() ([]Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT id, name, color, created_at FROM projects ORDER BY created_at ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetProject(id string) (Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return getProject(s.db, id)
}

func (s *Store) ProjectByName(name string) (Project, error) {
	name = strings.TrimSpace(name)
	ps, err := s.ListProjects()
	if err != nil {
		return Project{}, err
	}
	for _, p := range ps {
		if strings.EqualFold(p.Name, name) {
			return p, nil
		}
	}
	return Project{}, notFound("没有叫「" + name + "」的项目")
}

func getProject(q queryer, id string) (Project, error) {
	p, err := scanProject(q.QueryRow(`SELECT id, name, color, created_at FROM projects WHERE id = ?`, id))
	if err == sql.ErrNoRows {
		return Project{}, notFound("找不到这个项目")
	}
	return p, err
}

func getProjectTx(tx *sql.Tx, id string) (Project, error) { return getProject(tx, id) }

func scanProject(sc scanner) (Project, error) {
	var p Project
	var created string
	err := sc.Scan(&p.ID, &p.Name, &p.Color, &created)
	if err != nil {
		return Project{}, err
	}
	p.CreatedAt = parseT(created)
	return p, nil
}

func normalizeSteps(steps []string, step int) ([]string, int, error) {
	if len(steps) > 12 {
		return nil, 0, invalid("步骤太多了")
	}
	out := make([]string, 0, len(steps))
	for _, st := range steps {
		st, err := trimReq(st, 80, "步骤", true)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, st)
	}
	if step < 0 {
		return nil, 0, invalid("步骤序号不对")
	}
	if len(out) == 0 {
		step = 0
	} else if step > len(out) {
		step = len(out)
	}
	return out, step, nil
}

func (s *Store) CreateTodo(in CreateTodo) (Todo, error) {
	title, err := trimReq(in.Title, 200, "标题", true)
	if err != nil {
		return Todo{}, err
	}
	desc := strings.TrimSpace(in.Description)
	if runeLen(desc) > 20000 {
		return Todo{}, invalid("说明太长了")
	}
	note, err := trimReq(in.Note, 200, "备注", false)
	if err != nil {
		return Todo{}, err
	}
	due, err := trimReq(in.Due, 40, "截止", false)
	if err != nil {
		return Todo{}, err
	}
	status := in.Status
	if status == "" {
		status = "todo"
	}
	if status != "todo" && status != "in_progress" && status != "waiting" && status != "done" {
		return Todo{}, invalid("状态不对")
	}
	steps, step, err := normalizeSteps(in.Steps, in.Step)
	if err != nil {
		return Todo{}, err
	}
	if in.Source == "" {
		in.Source = "self"
	}
	switch in.Source {
	case "self", "agent", "intake_form", "intake_api":
	default:
		return Todo{}, invalid("来源不对")
	}
	lane := in.Lane
	if lane == "" {
		if in.DelegateID != "" {
			lane = in.DelegateID
		} else if in.Source == "intake_form" || in.Source == "intake_api" {
			lane = "others"
		} else {
			lane = "self"
		}
	}
	var todo Todo
	err = s.withTx(func(tx *sql.Tx) error {
		if in.ProjectID != "" {
			if _, err := getProjectTx(tx, in.ProjectID); err != nil {
				return err
			}
		}
		if in.DelegateID != "" {
			if _, err := getAgentTx(tx, in.DelegateID); err != nil {
				return err
			}
		}
		id, err := nextID(tx, "todo", "T-")
		if err != nil {
			return err
		}
		now := nowUTC()
		stepsJSON, _ := json.Marshal(steps)
		var completed any
		if status == "done" {
			completed = fmtT(now)
		}
		_, err = tx.Exec(`INSERT INTO todos(id, title, description, note, status, project_id, source, creator_kind, creator_id, creator_name, delegate_id, lane, due, steps, step_index, intake_id, created_at, updated_at, completed_at)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, title, desc, note, status, nullStr(in.ProjectID), in.Source, in.Creator.Kind, in.Creator.ID, in.Creator.Name,
			nullStr(in.DelegateID), lane, due, string(stepsJSON), step, nullStr(in.IntakeID), fmtT(now), fmtT(now), completed)
		if err != nil {
			return err
		}
		if err := addActivity(tx, id, in.Creator, "c", "创建"); err != nil {
			return err
		}
		todo, err = getTodoTx(tx, id)
		return err
	})
	return todo, err
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (s *Store) GetTodo(id string) (Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return getTodo(s.db, id)
}

func (s *Store) ListTodos() ([]Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return listTodos(s.db, `SELECT `+todoCols+` FROM todos ORDER BY updated_at DESC, id DESC`)
}

func (s *Store) UpdateTodo(id string, patch TodoPatch, actor Actor, owner bool) (Todo, error) {
	var out Todo
	err := s.withTx(func(tx *sql.Tx) error {
		t, err := getTodoTx(tx, id)
		if err != nil {
			return err
		}
		if !owner && t.DelegateID != actor.ID {
			return forbidden("只能改自己泳道里的 todo")
		}
		oldStatus := t.Status
		oldProject := t.ProjectID
		oldDelegate := t.DelegateID
		if patch.Title != nil {
			title, err := trimReq(*patch.Title, 200, "标题", true)
			if err != nil {
				return err
			}
			t.Title = title
		}
		if patch.Description != nil {
			desc := strings.TrimSpace(*patch.Description)
			if runeLen(desc) > 20000 {
				return invalid("说明太长了")
			}
			t.Description = desc
		}
		if patch.Note != nil {
			note, err := trimReq(*patch.Note, 200, "备注", false)
			if err != nil {
				return err
			}
			t.Note = note
		}
		if patch.Due != nil {
			due, err := trimReq(*patch.Due, 40, "截止", false)
			if err != nil {
				return err
			}
			t.Due = due
		}
		if patch.ProjectID != nil {
			if *patch.ProjectID != "" {
				if _, err := getProjectTx(tx, *patch.ProjectID); err != nil {
					return err
				}
			}
			t.ProjectID = *patch.ProjectID
		}
		if patch.Steps != nil || patch.Step != nil {
			steps := t.Steps
			step := t.Step
			if patch.Steps != nil {
				steps = *patch.Steps
			}
			if patch.Step != nil {
				step = *patch.Step
			}
			steps, step, err = normalizeSteps(steps, step)
			if err != nil {
				return err
			}
			t.Steps = steps
			t.Step = step
		}
		if patch.Status != nil {
			st := *patch.Status
			if st != "todo" && st != "in_progress" && st != "done" {
				return invalid("状态不对")
			}
			if !owner && st == "done" {
				return invalid("标完成请用 complete")
			}
			if !owner && t.Status == "waiting" && st != t.Status {
				return invalid("还在等主人回复，先别改状态")
			}
			t.Status = st
		}
		if owner && patch.DelegateID != nil {
			del := *patch.DelegateID
			if del != "" {
				if _, err := getAgentTx(tx, del); err != nil {
					return err
				}
				t.DelegateID = del
				t.Lane = del
			} else {
				t.DelegateID = ""
				if t.Lane != "others" {
					t.Lane = "self"
				}
			}
		}
		if patch.Lane != nil && owner {
			t.Lane = *patch.Lane
		}
		now := nowUTC()
		var completed any
		if t.Status == "done" {
			if t.CompletedAt != nil {
				completed = fmtT(*t.CompletedAt)
			} else {
				completed = fmtT(now)
			}
		}
		stepsJSON, _ := json.Marshal(t.Steps)
		_, err = tx.Exec(`UPDATE todos SET title=?, description=?, note=?, status=?, project_id=?, delegate_id=?, lane=?, due=?, steps=?, step_index=?, updated_at=?, completed_at=? WHERE id=?`,
			t.Title, t.Description, t.Note, t.Status, nullStr(t.ProjectID), nullStr(t.DelegateID), t.Lane, t.Due, string(stepsJSON), t.Step, fmtT(now), completed, id)
		if err != nil {
			return err
		}
		if patch.Status != nil && *patch.Status == "in_progress" && oldStatus != "in_progress" {
			if err := addActivity(tx, id, actor, "c", "开始处理"); err != nil {
				return err
			}
		}
		if patch.Status != nil && *patch.Status == "done" && oldStatus != "done" {
			if err := closeQuestions(tx, id, now, "（todo 已完成）"); err != nil {
				return err
			}
			if err := addActivity(tx, id, actor, "c", "标记完成"); err != nil {
				return err
			}
		}
		if patch.ProjectID != nil && *patch.ProjectID != oldProject {
			msg := "移出项目"
			if *patch.ProjectID != "" {
				p, err := getProjectTx(tx, *patch.ProjectID)
				if err != nil {
					return err
				}
				msg = "归到项目 " + p.Name
			}
			if err := addActivity(tx, id, actor, "c", msg); err != nil {
				return err
			}
		}
		if patch.Progress != "" {
			note, err := trimReq(patch.Progress, 2000, "进度", true)
			if err != nil {
				return err
			}
			if err := addActivity(tx, id, actor, "m", note); err != nil {
				return err
			}
		}
		if owner && patch.DelegateID != nil && t.DelegateID != oldDelegate {
			who := "我自己"
			if t.DelegateID != "" {
				ag, err := getAgentTx(tx, t.DelegateID)
				if err != nil {
					return err
				}
				who = ag.Name
			}
			if err := addActivity(tx, id, actor, "c", "执行改为 "+who); err != nil {
				return err
			}
		}
		out, err = getTodoTx(tx, id)
		return err
	})
	return out, err
}

func (s *Store) CompleteTodo(id, note string, actor Actor, owner bool) (Todo, error) {
	var out Todo
	err := s.withTx(func(tx *sql.Tx) error {
		t, err := getTodoTx(tx, id)
		if err != nil {
			return err
		}
		if !owner && t.DelegateID != actor.ID {
			return forbidden("只能完成自己泳道里的 todo")
		}
		if t.Status == "done" {
			out = t
			return nil
		}
		now := nowUTC()
		_, err = tx.Exec(`UPDATE todos SET status='done', updated_at=?, completed_at=? WHERE id=?`, fmtT(now), fmtT(now), id)
		if err != nil {
			return err
		}
		if err := closeQuestions(tx, id, now, "（todo 已完成）"); err != nil {
			return err
		}
		if strings.TrimSpace(note) != "" {
			n, err := trimReq(note, 2000, "备注", true)
			if err != nil {
				return err
			}
			if err := addActivity(tx, id, actor, "m", n); err != nil {
				return err
			}
		}
		if err := addActivity(tx, id, actor, "c", "标记完成"); err != nil {
			return err
		}
		out, err = getTodoTx(tx, id)
		return err
	})
	return out, err
}

func (s *Store) DeleteTodo(id string, owner bool) error {
	if !owner {
		return forbidden("只有主人能删除 todo")
	}
	return s.withTx(func(tx *sql.Tx) error {
		if _, err := getTodoTx(tx, id); err != nil {
			return err
		}
		for _, q := range []string{
			`DELETE FROM activity WHERE todo_id = ?`,
			`DELETE FROM questions WHERE todo_id = ?`,
			`DELETE FROM todos WHERE id = ?`,
		} {
			if _, err := tx.Exec(q, id); err != nil {
				return err
			}
		}
		_, err := tx.Exec(`UPDATE intakes SET todo_id = '' WHERE todo_id = ?`, id)
		return err
	})
}

func (s *Store) AddComment(id, body string, actor Actor, owner bool) (Activity, error) {
	body, err := trimReq(body, 2000, "备注", true)
	if err != nil {
		return Activity{}, err
	}
	var act Activity
	err = s.withTx(func(tx *sql.Tx) error {
		t, err := getTodoTx(tx, id)
		if err != nil {
			return err
		}
		if !owner && t.DelegateID != actor.ID {
			return forbidden("只能改自己泳道里的 todo")
		}
		now := nowUTC()
		_, err = tx.Exec(`UPDATE todos SET updated_at = ? WHERE id = ?`, fmtT(now), id)
		if err != nil {
			return err
		}
		if err := addActivity(tx, id, actor, "m", body); err != nil {
			return err
		}
		act = Activity{TodoID: id, ActorKind: actor.Kind, ActorID: actor.ID, ActorName: actor.Name, Kind: "m", Text: body, CreatedAt: now}
		return nil
	})
	return act, err
}

func closeQuestions(tx *sql.Tx, todoID string, now time.Time, answer string) error {
	_, err := tx.Exec(`UPDATE questions SET answered_at = ?, answer = ? WHERE todo_id = ? AND answered_at IS NULL`,
		fmtT(now), answer, todoID)
	return err
}

const todoCols = `id, title, description, note, status, project_id, source, creator_kind, creator_id, creator_name, delegate_id, lane, due, steps, step_index, intake_id, created_at, updated_at, completed_at`

func listTodos(q interface {
	Query(string, ...any) (*sql.Rows, error)
}, query string, args ...any) ([]Todo, error) {
	rows, err := q.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Todo
	for rows.Next() {
		t, err := scanTodo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func getTodo(q queryer, id string) (Todo, error) {
	t, err := scanTodo(q.QueryRow(`SELECT `+todoCols+` FROM todos WHERE id = ?`, id))
	if err == sql.ErrNoRows {
		return Todo{}, notFound("找不到这条 todo")
	}
	return t, err
}

func getTodoTx(tx *sql.Tx, id string) (Todo, error) { return getTodo(tx, id) }

func scanTodo(sc scanner) (Todo, error) {
	var t Todo
	var project, delegate, intake sql.NullString
	var steps, created, updated string
	var completed sql.NullString
	err := sc.Scan(&t.ID, &t.Title, &t.Description, &t.Note, &t.Status, &project, &t.Source,
		&t.CreatorKind, &t.CreatorID, &t.CreatorName, &delegate, &t.Lane, &t.Due, &steps, &t.Step, &intake,
		&created, &updated, &completed)
	if err != nil {
		return Todo{}, err
	}
	t.ProjectID = project.String
	t.DelegateID = delegate.String
	t.IntakeID = intake.String
	t.Steps = []string{}
	_ = json.Unmarshal([]byte(steps), &t.Steps)
	if t.Steps == nil {
		t.Steps = []string{}
	}
	t.CreatedAt = parseT(created)
	t.UpdatedAt = parseT(updated)
	if completed.Valid && completed.String != "" {
		c := parseT(completed.String)
		t.CompletedAt = &c
	}
	return t, nil
}

type Ask struct {
	TodoID   string
	Title    string
	Question string
	Context  string
	Options  []string
	Agent    Actor
}

func (s *Store) AskOwner(in Ask) (Question, Todo, error) {
	qtext, err := trimReq(in.Question, 300, "问题", true)
	if err != nil {
		return Question{}, Todo{}, err
	}
	ctx, err := trimReq(in.Context, 1000, "补充", false)
	if err != nil {
		return Question{}, Todo{}, err
	}
	opts, err := normalizeOptions(in.Options)
	if err != nil {
		return Question{}, Todo{}, err
	}
	var question Question
	var todo Todo
	err = s.withTx(func(tx *sql.Tx) error {
		var t Todo
		if in.TodoID != "" {
			t, err = getTodoTx(tx, in.TodoID)
			if err != nil {
				return err
			}
			if t.DelegateID != in.Agent.ID {
				return forbidden("只能在自己的 todo 上提问")
			}
			if t.Status == "done" {
				return invalid("这条已经做完了")
			}
		} else {
			title := strings.TrimSpace(in.Title)
			if title == "" {
				title = qtext
			}
			title, err = trimReq(title, 200, "标题", true)
			if err != nil {
				return err
			}
			id, err := nextID(tx, "todo", "T-")
			if err != nil {
				return err
			}
			now := nowUTC()
			_, err = tx.Exec(`INSERT INTO todos(id, title, description, note, status, source, creator_kind, creator_id, creator_name, delegate_id, lane, due, steps, step_index, created_at, updated_at)
				VALUES(?, ?, '', '', 'waiting', 'agent', ?, ?, ?, ?, ?, '', '[]', 0, ?, ?)`,
				id, title, in.Agent.Kind, in.Agent.ID, in.Agent.Name, in.Agent.ID, in.Agent.ID, fmtT(now), fmtT(now))
			if err != nil {
				return err
			}
			if err := addActivity(tx, id, in.Agent, "c", "创建"); err != nil {
				return err
			}
			t, err = getTodoTx(tx, id)
			if err != nil {
				return err
			}
		}
		var existing int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM questions WHERE todo_id = ? AND answered_at IS NULL`, t.ID).Scan(&existing); err != nil {
			return err
		}
		if existing > 0 {
			return conflict("这条 todo 已经有一个还没回答的问题")
		}
		qid, err := nextID(tx, "question", "Q-")
		if err != nil {
			return err
		}
		now := nowUTC()
		optJSON, _ := json.Marshal(opts)
		_, err = tx.Exec(`INSERT INTO questions(id, todo_id, agent_id, text, context, options, created_at) VALUES(?, ?, ?, ?, ?, ?, ?)`,
			qid, t.ID, in.Agent.ID, qtext, ctx, string(optJSON), fmtT(now))
		if err != nil {
			return err
		}
		_, err = tx.Exec(`UPDATE todos SET status = 'waiting', updated_at = ? WHERE id = ?`, fmtT(now), t.ID)
		if err != nil {
			return err
		}
		if err := addActivity(tx, t.ID, in.Agent, "m", "向你确认："+qtext); err != nil {
			return err
		}
		question, err = getQuestionTx(tx, qid)
		if err != nil {
			return err
		}
		todo, err = getTodoTx(tx, t.ID)
		return err
	})
	return question, todo, err
}

func normalizeOptions(opts []string) ([]string, error) {
	if len(opts) == 0 {
		return []string{"可以", "先不要"}, nil
	}
	if len(opts) < 2 || len(opts) > 4 {
		return nil, invalid("请给 2 到 4 个选项")
	}
	out := make([]string, 0, len(opts))
	seen := map[string]bool{}
	for _, o := range opts {
		o, err := trimReq(o, 24, "选项", true)
		if err != nil {
			return nil, err
		}
		if seen[o] {
			return nil, invalid("选项重复了")
		}
		seen[o] = true
		out = append(out, o)
	}
	return out, nil
}

func (s *Store) AnswerQuestion(id, option string, owner Actor) (Question, Todo, error) {
	option = strings.TrimSpace(option)
	if option == "" {
		return Question{}, Todo{}, invalid("先选一个回答")
	}
	var q Question
	var t Todo
	err := s.withTx(func(tx *sql.Tx) error {
		cur, err := getQuestionTx(tx, id)
		if err != nil {
			return err
		}
		if cur.AnsweredAt != nil {
			return conflict("已经回答过了")
		}
		ok := false
		for _, o := range cur.Options {
			if o == option {
				ok = true
			}
		}
		if !ok {
			return invalid("这个选项不在列表里")
		}
		now := nowUTC()
		_, err = tx.Exec(`UPDATE questions SET answered_at = ?, answer = ? WHERE id = ?`, fmtT(now), option, id)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`UPDATE todos SET status = 'in_progress', updated_at = ? WHERE id = ? AND status != 'done'`, fmtT(now), cur.TodoID)
		if err != nil {
			return err
		}
		if err := addActivity(tx, cur.TodoID, owner, "m", "回复："+option); err != nil {
			return err
		}
		q, err = getQuestionTx(tx, id)
		if err != nil {
			return err
		}
		t, err = getTodoTx(tx, cur.TodoID)
		return err
	})
	return q, t, err
}

func (s *Store) ListOpenQuestions() ([]Question, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return listQuestions(s.db, `SELECT `+qCols+` FROM questions WHERE answered_at IS NULL ORDER BY created_at ASC, id ASC`)
}

func (s *Store) ListQuestions() ([]Question, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return listQuestions(s.db, `SELECT `+qCols+` FROM questions ORDER BY created_at ASC, id ASC`)
}

const qCols = `id, todo_id, agent_id, text, context, options, created_at, answered_at, answer`

func listQuestions(q interface {
	Query(string, ...any) (*sql.Rows, error)
}, query string, args ...any) ([]Question, error) {
	rows, err := q.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Question
	for rows.Next() {
		item, err := scanQuestion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func getQuestionTx(tx *sql.Tx, id string) (Question, error) {
	item, err := scanQuestion(tx.QueryRow(`SELECT `+qCols+` FROM questions WHERE id = ?`, id))
	if err == sql.ErrNoRows {
		return Question{}, notFound("找不到这个问题")
	}
	return item, err
}

func scanQuestion(sc scanner) (Question, error) {
	var q Question
	var opts, created string
	var answered sql.NullString
	err := sc.Scan(&q.ID, &q.TodoID, &q.AgentID, &q.Text, &q.Context, &opts, &created, &answered, &q.Answer)
	if err != nil {
		return Question{}, err
	}
	q.Options = []string{}
	_ = json.Unmarshal([]byte(opts), &q.Options)
	q.CreatedAt = parseT(created)
	if answered.Valid && answered.String != "" {
		t := parseT(answered.String)
		q.AnsweredAt = &t
	}
	return q, nil
}

type IntakeIn struct {
	Title     string
	Body      string
	FromName  string
	FromKind  string
	Contact   string
	Want      string
	Via       string
	SuggestID string
}

func (s *Store) SubmitIntake(in IntakeIn) (Intake, error) {
	title, err := trimReq(in.Title, 200, "标题", true)
	if err != nil {
		return Intake{}, err
	}
	body := strings.TrimSpace(in.Body)
	if runeLen(body) > 8000 {
		return Intake{}, invalid("说明太长了")
	}
	name, err := trimReq(in.FromName, 40, "名字", false)
	if err != nil {
		return Intake{}, err
	}
	if name == "" {
		name = "匿名"
	}
	kind := in.FromKind
	if kind == "" {
		kind = "human"
	}
	if kind != "human" && kind != "agent" {
		return Intake{}, invalid("身份不对")
	}
	contact, err := trimReq(in.Contact, 120, "联系方式", false)
	if err != nil {
		return Intake{}, err
	}
	want, err := trimReq(in.Want, 40, "希望时间", false)
	if err != nil {
		return Intake{}, err
	}
	via := in.Via
	if via == "" {
		via = "公开登记页"
	}
	var item Intake
	err = s.withTx(func(tx *sql.Tx) error {
		suggest := in.SuggestID
		if suggest != "" {
			if _, err := getAgentTx(tx, suggest); err != nil {
				return err
			}
		}
		id, err := nextID(tx, "intake", "R-")
		if err != nil {
			return err
		}
		now := nowUTC()
		_, err = tx.Exec(`INSERT INTO intakes(id, title, body, from_name, from_kind, contact, want, via, status, suggest_id, created_at)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?)`,
			id, title, body, name, kind, contact, want, via, suggest, fmtT(now))
		if err != nil {
			return err
		}
		item, err = getIntakeTx(tx, id)
		return err
	})
	return item, err
}

func (s *Store) AcceptIntake(id, assignee string, owner Actor) (Intake, Todo, error) {
	var item Intake
	var todo Todo
	err := s.withTx(func(tx *sql.Tx) error {
		cur, err := getIntakeTx(tx, id)
		if err != nil {
			return err
		}
		if cur.Status != "pending" {
			return conflict("这条登记已经处理过了")
		}
		delegate := ""
		lane := "others"
		if assignee != "" && assignee != "owner" {
			ag, err := getAgentTx(tx, assignee)
			if err != nil {
				return err
			}
			delegate = ag.ID
			lane = ag.ID
		}
		tid, err := nextID(tx, "todo", "T-")
		if err != nil {
			return err
		}
		now := nowUTC()
		source := "intake_form"
		if cur.Via == "登记 API" {
			source = "intake_api"
		}
		ckind := "external"
		if cur.FromKind == "agent" {
			ckind = "extagent"
		}
		_, err = tx.Exec(`INSERT INTO todos(id, title, description, note, status, source, creator_kind, creator_id, creator_name, delegate_id, lane, due, steps, step_index, intake_id, created_at, updated_at)
			VALUES(?, ?, ?, '', 'todo', ?, ?, '', ?, ?, ?, ?, '[]', 0, ?, ?, ?)`,
			tid, cur.Title, cur.Body, source, ckind, cur.FromName, nullStr(delegate), lane, cur.Want, cur.ID, fmtT(now), fmtT(now))
		if err != nil {
			return err
		}
		ext := Actor{Kind: ckind, Name: cur.FromName}
		if err := addActivity(tx, tid, ext, "c", "从"+cur.Via+"登记"); err != nil {
			return err
		}
		who := "自己处理"
		if delegate != "" {
			ag, _ := getAgentTx(tx, delegate)
			who = "交给 " + ag.Name + " 执行"
		}
		if err := addActivity(tx, tid, owner, "c", "接受，"+who); err != nil {
			return err
		}
		picked := "owner"
		if delegate != "" {
			picked = delegate
		}
		_, err = tx.Exec(`UPDATE intakes SET status='accepted', picked_id=?, todo_id=?, decided_at=? WHERE id=?`,
			picked, tid, fmtT(now), id)
		if err != nil {
			return err
		}
		item, err = getIntakeTx(tx, id)
		if err != nil {
			return err
		}
		todo, err = getTodoTx(tx, tid)
		return err
	})
	return item, todo, err
}

func (s *Store) DeclineIntake(id string) (Intake, error) {
	var item Intake
	err := s.withTx(func(tx *sql.Tx) error {
		cur, err := getIntakeTx(tx, id)
		if err != nil {
			return err
		}
		if cur.Status != "pending" {
			return conflict("这条登记已经处理过了")
		}
		now := fmtT(nowUTC())
		_, err = tx.Exec(`UPDATE intakes SET status='declined', decided_at=? WHERE id=?`, now, id)
		if err != nil {
			return err
		}
		item, err = getIntakeTx(tx, id)
		return err
	})
	return item, err
}

func (s *Store) ListIntakes(status string) ([]Intake, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	q := `SELECT ` + intakeCols + ` FROM intakes`
	var args []any
	if status != "" {
		q += ` WHERE status = ?`
		args = append(args, status)
	}
	if status == "pending" || status == "" {
		q += ` ORDER BY created_at DESC, id DESC`
	} else {
		q += ` ORDER BY decided_at DESC, id DESC`
	}
	return listIntakes(s.db, q, args...)
}

const intakeCols = `id, title, body, from_name, from_kind, contact, want, via, status, suggest_id, picked_id, todo_id, created_at, decided_at`

func listIntakes(q interface {
	Query(string, ...any) (*sql.Rows, error)
}, query string, args ...any) ([]Intake, error) {
	rows, err := q.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Intake
	for rows.Next() {
		item, err := scanIntake(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func getIntakeTx(tx *sql.Tx, id string) (Intake, error) {
	item, err := scanIntake(tx.QueryRow(`SELECT `+intakeCols+` FROM intakes WHERE id = ?`, id))
	if err == sql.ErrNoRows {
		return Intake{}, notFound("找不到这条登记")
	}
	return item, err
}

func scanIntake(sc scanner) (Intake, error) {
	var it Intake
	var created string
	var decided sql.NullString
	err := sc.Scan(&it.ID, &it.Title, &it.Body, &it.FromName, &it.FromKind, &it.Contact, &it.Want, &it.Via,
		&it.Status, &it.SuggestID, &it.PickedID, &it.TodoID, &created, &decided)
	if err != nil {
		return Intake{}, err
	}
	it.CreatedAt = parseT(created)
	if decided.Valid && decided.String != "" {
		t := parseT(decided.String)
		it.DecidedAt = &t
	}
	return it, nil
}

func addActivity(tx *sql.Tx, todoID string, actor Actor, kind, text string) error {
	_, err := tx.Exec(`INSERT INTO activity(todo_id, actor_kind, actor_id, actor_name, kind, text, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)`, todoID, actor.Kind, actor.ID, actor.Name, kind, text, fmtT(nowUTC()))
	return err
}

func (s *Store) ListActivity(todoID string) ([]Activity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return listActivity(s.db, `SELECT id, todo_id, actor_kind, actor_id, actor_name, kind, text, created_at FROM activity WHERE todo_id = ? ORDER BY id ASC`, todoID)
}

func (s *Store) ListAllActivity() ([]Activity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return listActivity(s.db, `SELECT id, todo_id, actor_kind, actor_id, actor_name, kind, text, created_at FROM activity ORDER BY id ASC`)
}

func listActivity(q interface {
	Query(string, ...any) (*sql.Rows, error)
}, query string, args ...any) ([]Activity, error) {
	rows, err := q.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Activity
	for rows.Next() {
		var a Activity
		var created string
		if err := rows.Scan(&a.ID, &a.TodoID, &a.ActorKind, &a.ActorID, &a.ActorName, &a.Kind, &a.Text, &created); err != nil {
			return nil, err
		}
		a.CreatedAt = parseT(created)
		out = append(out, a)
	}
	return out, rows.Err()
}

// ResetDomain wipes todos, agents, projects and inbox items. Sessions and
// the owner password stay.
func (s *Store) ResetDomain() error {
	return s.withTx(func(tx *sql.Tx) error {
		for _, table := range []string{"activity", "questions", "intakes", "todos", "projects", "agents", "counters"} {
			if _, err := tx.Exec("DELETE FROM " + table); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) DomainEmpty() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int
	err := s.db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM agents) + (SELECT COUNT(*) FROM todos) +
		(SELECT COUNT(*) FROM projects) + (SELECT COUNT(*) FROM intakes)`).Scan(&n)
	return n == 0, err
}

// ResolveSuggest finds an agent by name for an intake suggestion. Empty name
// means the owner. Unknown names are ignored.
func (s *Store) ResolveSuggest(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == "owner" || name == "我" || name == "我自己" {
		return "", nil
	}
	ags, err := s.ListAgents()
	if err != nil {
		return "", err
	}
	for _, a := range ags {
		if strings.EqualFold(a.Name, name) {
			return a.ID, nil
		}
	}
	return "", nil
}

package store

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	_ "modernc.org/sqlite"
)

const (
	CodeInvalid   = "invalid"
	CodeNotFound  = "not_found"
	CodeConflict  = "conflict"
	CodeForbidden = "forbidden"
)

// Error is a client-facing failure with a stable code.
type Error struct {
	Code string
	Msg  string
}

func (e *Error) Error() string { return e.Msg }

func invalid(msg string) error   { return &Error{Code: CodeInvalid, Msg: msg} }
func notFound(msg string) error  { return &Error{Code: CodeNotFound, Msg: msg} }
func conflict(msg string) error  { return &Error{Code: CodeConflict, Msg: msg} }
func forbidden(msg string) error { return &Error{Code: CodeForbidden, Msg: msg} }

func AsError(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return nil
}

// Actor is who performed an action.
type Actor struct {
	Kind string // owner, agent, external, extagent
	ID   string
	Name string
}

type Agent struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Role        string     `json:"role"`
	Look        Look       `json:"look"`
	TokenPrefix string     `json:"token_prefix"`
	CreatedAt   time.Time  `json:"created_at"`
	LastSeenAt  *time.Time `json:"last_seen_at"`
	Revoked     bool       `json:"revoked"`
}

type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

type Todo struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Note        string     `json:"note"`
	Status      string     `json:"status"`
	ProjectID   string     `json:"project_id"`
	Source      string     `json:"source"`
	CreatorKind string     `json:"creator_kind"`
	CreatorID   string     `json:"creator_id"`
	CreatorName string     `json:"creator_name"`
	DelegateID  string     `json:"delegate_id"`
	Lane        string     `json:"lane"`
	Due         string     `json:"due"`
	Steps       []string   `json:"steps"`
	Step        int        `json:"step"`
	IntakeID    string     `json:"intake_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

type Question struct {
	ID         string     `json:"id"`
	TodoID     string     `json:"todo_id"`
	AgentID    string     `json:"agent_id"`
	Text       string     `json:"text"`
	Context    string     `json:"context"`
	Options    []string   `json:"options"`
	CreatedAt  time.Time  `json:"created_at"`
	AnsweredAt *time.Time `json:"answered_at"`
	Answer     string     `json:"answer"`
}

type Intake struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	FromName  string     `json:"from_name"`
	FromKind  string     `json:"from_kind"`
	Contact   string     `json:"contact"`
	Want      string     `json:"want"`
	Via       string     `json:"via"`
	Status    string     `json:"status"`
	SuggestID string     `json:"suggest_id"`
	PickedID  string     `json:"picked_id"`
	TodoID    string     `json:"todo_id"`
	CreatedAt time.Time  `json:"created_at"`
	DecidedAt *time.Time `json:"decided_at"`
}

type Activity struct {
	ID        int64     `json:"id"`
	TodoID    string    `json:"todo_id"`
	ActorKind string    `json:"actor_kind"`
	ActorID   string    `json:"actor_id"`
	ActorName string    `json:"actor_name"`
	Kind      string    `json:"kind"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateTodo struct {
	Title       string
	Description string
	Note        string
	ProjectID   string
	Due         string
	Status      string
	Steps       []string
	Step        int
	Source      string
	Creator     Actor
	DelegateID  string
	Lane        string
	IntakeID    string
}

type TodoPatch struct {
	Title       *string
	Description *string
	Note        *string
	ProjectID   *string
	Due         *string
	Status      *string
	Steps       *[]string
	Step        *int
	DelegateID  *string
	Lane        *string
	Progress    string
}

// Store is the SQLite persistence layer. A mutex serializes access; this is
// a single-user app and one connection keeps SQLite simple.
type Store struct {
	db *sql.DB
	mu sync.Mutex
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	_, err := s.db.Exec(schema)
	return err
}

const schema = `
CREATE TABLE IF NOT EXISTS settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions (
  token_hash TEXT PRIMARY KEY,
  created_at TEXT NOT NULL,
  expires_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS counters (
  name TEXT PRIMARY KEY,
  n INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS agents (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT '',
  look TEXT NOT NULL,
  token_hash TEXT,
  token_prefix TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  last_seen_at TEXT,
  revoked INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS projects (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  color TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS todos (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  note TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  project_id TEXT,
  source TEXT NOT NULL,
  creator_kind TEXT NOT NULL,
  creator_id TEXT NOT NULL DEFAULT '',
  creator_name TEXT NOT NULL,
  delegate_id TEXT,
  lane TEXT NOT NULL,
  due TEXT NOT NULL DEFAULT '',
  steps TEXT NOT NULL DEFAULT '[]',
  step_index INTEGER NOT NULL DEFAULT 0,
  intake_id TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  completed_at TEXT
);
CREATE TABLE IF NOT EXISTS questions (
  id TEXT PRIMARY KEY,
  todo_id TEXT NOT NULL,
  agent_id TEXT NOT NULL,
  text TEXT NOT NULL,
  context TEXT NOT NULL DEFAULT '',
  options TEXT NOT NULL,
  created_at TEXT NOT NULL,
  answered_at TEXT,
  answer TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS intakes (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  body TEXT NOT NULL DEFAULT '',
  from_name TEXT NOT NULL,
  from_kind TEXT NOT NULL,
  contact TEXT NOT NULL DEFAULT '',
  want TEXT NOT NULL DEFAULT '',
  via TEXT NOT NULL,
  status TEXT NOT NULL,
  suggest_id TEXT NOT NULL DEFAULT '',
  picked_id TEXT NOT NULL DEFAULT '',
  todo_id TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  decided_at TEXT
);
CREATE TABLE IF NOT EXISTS activity (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  todo_id TEXT NOT NULL,
  actor_kind TEXT NOT NULL,
  actor_id TEXT NOT NULL DEFAULT '',
  actor_name TEXT NOT NULL,
  kind TEXT NOT NULL,
  text TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_todos_lane ON todos(lane);
CREATE INDEX IF NOT EXISTS idx_todos_delegate ON todos(delegate_id);
CREATE INDEX IF NOT EXISTS idx_questions_todo ON questions(todo_id);
CREATE INDEX IF NOT EXISTS idx_activity_todo ON activity(todo_id);
CREATE INDEX IF NOT EXISTS idx_agents_token ON agents(token_hash);
`

func (s *Store) withTx(fn func(*sql.Tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func nowUTC() time.Time { return time.Now().UTC().Truncate(time.Millisecond) }

func fmtT(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func parseT(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, s)
	}
	return t
}

func (s *Store) GetSetting(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var v string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}

func (s *Store) SetSetting(key, value string) error {
	return s.withTx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`INSERT INTO settings(key, value) VALUES(?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
		return err
	})
}

func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func randomToken(prefix string, nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(b), nil
}

func (s *Store) CreateSession(ttl time.Duration) (string, error) {
	raw, err := randomToken("", 32)
	if err != nil {
		return "", err
	}
	exp := nowUTC().Add(ttl)
	err = s.withTx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`INSERT INTO sessions(token_hash, created_at, expires_at) VALUES(?, ?, ?)`,
			HashToken(raw), fmtT(nowUTC()), fmtT(exp))
		return err
	})
	if err != nil {
		return "", err
	}
	return raw, nil
}

func (s *Store) DeleteSession(raw string) error {
	return s.withTx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`DELETE FROM sessions WHERE token_hash = ?`, HashToken(raw))
		return err
	})
}

func (s *Store) ValidSession(raw string) (bool, error) {
	if raw == "" {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := fmtT(nowUTC())
	_, _ = s.db.Exec(`DELETE FROM sessions WHERE expires_at < ?`, now)
	var one int
	err := s.db.QueryRow(`SELECT 1 FROM sessions WHERE token_hash = ? AND expires_at >= ?`, HashToken(raw), now).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func nextID(tx *sql.Tx, name, prefix string) (string, error) {
	var n int
	err := tx.QueryRow(`INSERT INTO counters(name, n) VALUES(?, 1)
		ON CONFLICT(name) DO UPDATE SET n = n + 1 RETURNING n`, name).Scan(&n)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%d", prefix, n), nil
}

func runeLen(s string) int { return utf8.RuneCountInString(s) }

func trimReq(s string, max int, label string, required bool) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		if required {
			return "", invalid(label + "不能空着")
		}
		return "", nil
	}
	if runeLen(s) > max {
		return "", invalid(label + "太长了")
	}
	if strings.ContainsAny(s, "\r\n") && max <= 300 {
		return "", invalid(label + "不能换行")
	}
	return s, nil
}

func foldTaken(tx *sql.Tx, query, name, except string, args ...any) (bool, error) {
	rows, err := tx.Query(query, args...)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, n string
		if err := rows.Scan(&id, &n); err != nil {
			return false, err
		}
		if id != except && strings.EqualFold(strings.TrimSpace(n), name) {
			return true, nil
		}
	}
	return false, rows.Err()
}

package sqlitestore

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	_ "modernc.org/sqlite"

	"plum/controller/internal/store"
)

type sqliteStore struct {
	db *sql.DB
}

func New(dbPath string) (store.Store, error) {
	// modernc sqlite driver name is "sqlite"
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL; PRAGMA foreign_keys=ON;`); err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return &sqliteStore{db: db}, nil
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS nodes (
			node_id TEXT PRIMARY KEY,
			ip TEXT,
			labels TEXT,
			last_seen INTEGER
		);`,
		`CREATE TABLE IF NOT EXISTS tasks (
			task_id TEXT PRIMARY KEY,
			name TEXT,
			labels TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS assignments (
			instance_id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			node_id TEXT NOT NULL,
			desired TEXT NOT NULL,
			artifact_url TEXT,
			start_cmd TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS statuses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			instance_id TEXT NOT NULL,
			phase TEXT,
			exit_code INTEGER,
			healthy INTEGER,
			ts_unix INTEGER
		);`,
		`CREATE TABLE IF NOT EXISTS artifacts (
			artifact_id TEXT PRIMARY KEY,
			app_name TEXT,
			version TEXT,
			path TEXT,
			sha256 TEXT,
			size_bytes INTEGER,
			created_at INTEGER
		);`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

func (s *sqliteStore) UpsertNode(id string, n store.Node) error {
	labelsJSON, _ := json.Marshal(n.Labels)
	_, err := s.db.Exec(
		`INSERT INTO nodes(node_id, ip, labels, last_seen) VALUES(?,?,?,?)
		 ON CONFLICT(node_id) DO UPDATE SET ip=excluded.ip, labels=excluded.labels, last_seen=excluded.last_seen`,
		id, n.IP, string(labelsJSON), n.LastSeen.Unix(),
	)
	return err
}

func (s *sqliteStore) GetNode(id string) (store.Node, bool, error) {
    row := s.db.QueryRow(`SELECT node_id, ip, labels, last_seen FROM nodes WHERE node_id=?`, id)
    var n store.Node
    var labelsStr string
    var last int64
    if err := row.Scan(&n.NodeID, &n.IP, &labelsStr, &last); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return store.Node{}, false, nil
        }
        return store.Node{}, false, err
    }
    _ = json.Unmarshal([]byte(labelsStr), &n.Labels)
    n.LastSeen = time.Unix(last, 0)
    return n, true, nil
}

func (s *sqliteStore) ListNodes() ([]store.Node, error) {
    rows, err := s.db.Query(`SELECT node_id, ip, labels, last_seen FROM nodes ORDER BY node_id`)
    if err != nil { return nil, err }
    defer rows.Close()
    var out []store.Node
    for rows.Next() {
        var n store.Node
        var labelsStr string
        var last int64
        if err := rows.Scan(&n.NodeID, &n.IP, &labelsStr, &last); err != nil { return nil, err }
        _ = json.Unmarshal([]byte(labelsStr), &n.Labels)
        n.LastSeen = time.Unix(last, 0)
        out = append(out, n)
    }
    return out, rows.Err()
}

func (s *sqliteStore) DeleteNode(id string) error {
    _, err := s.db.Exec(`DELETE FROM nodes WHERE node_id=?`, id)
    return err
}

func (s *sqliteStore) ListAssignmentsForNode(nodeID string) ([]store.Assignment, error) {
	rows, err := s.db.Query(`SELECT instance_id, task_id, node_id, desired, artifact_url, start_cmd FROM assignments WHERE node_id=?`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Assignment
	for rows.Next() {
		var a store.Assignment
		if err := rows.Scan(&a.InstanceID, &a.TaskID, &a.NodeID, &a.Desired, &a.ArtifactURL, &a.StartCmd); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *sqliteStore) AddAssignment(nodeID string, a store.Assignment) error {
	if nodeID == "" {
		return errors.New("nodeID required")
	}
	_, err := s.db.Exec(`INSERT INTO assignments(instance_id, task_id, node_id, desired, artifact_url, start_cmd) VALUES(?,?,?,?,?,?)`,
		a.InstanceID, a.TaskID, nodeID, a.Desired, a.ArtifactURL, a.StartCmd,
	)
	return err
}

func (s *sqliteStore) AppendStatus(instanceID string, st store.InstanceStatus) error {
	_, err := s.db.Exec(`INSERT INTO statuses(instance_id, phase, exit_code, healthy, ts_unix) VALUES(?,?,?,?,?)`,
		instanceID, st.Phase, st.ExitCode, boolToInt(st.Healthy), st.TsUnix,
	)
	return err
}

func (s *sqliteStore) LatestStatus(instanceID string) (store.InstanceStatus, bool, error) {
    row := s.db.QueryRow(`SELECT instance_id, phase, exit_code, healthy, ts_unix FROM statuses WHERE instance_id=? ORDER BY ts_unix DESC, id DESC LIMIT 1`, instanceID)
    var st store.InstanceStatus
    var healthy int
    if err := row.Scan(&st.InstanceID, &st.Phase, &st.ExitCode, &healthy, &st.TsUnix); err != nil {
        if errors.Is(err, sql.ErrNoRows) { return store.InstanceStatus{}, false, nil }
        return store.InstanceStatus{}, false, err
    }
    st.Healthy = healthy != 0
    return st, true, nil
}

func (s *sqliteStore) CreateTask(name string, labels map[string]string) (string, []string, error) {
	id := newID()
	labelsJSON, _ := json.Marshal(labels)
	if _, err := s.db.Exec(`INSERT INTO tasks(task_id, name, labels) VALUES(?,?,?)`, id, name, string(labelsJSON)); err != nil {
		return "", nil, err
	}
	return id, []string{}, nil
}

func (s *sqliteStore) NewInstanceID(taskID string) string {
	return taskID + "-" + newID()[:8]
}

func (s *sqliteStore) ListTasks() ([]store.Task, error) {
    rows, err := s.db.Query(`SELECT task_id, name, labels FROM tasks ORDER BY rowid DESC`)
    if err != nil { return nil, err }
    defer rows.Close()
    var out []store.Task
    for rows.Next() {
        var t store.Task
        var labelsStr string
        if err := rows.Scan(&t.TaskID, &t.Name, &labelsStr); err != nil { return nil, err }
        _ = json.Unmarshal([]byte(labelsStr), &t.Labels)
        out = append(out, t)
    }
    return out, rows.Err()
}

func (s *sqliteStore) GetTask(id string) (store.Task, bool, error) {
    row := s.db.QueryRow(`SELECT task_id, name, labels FROM tasks WHERE task_id=?`, id)
    var t store.Task
    var labelsStr string
    if err := row.Scan(&t.TaskID, &t.Name, &labelsStr); err != nil {
        if errors.Is(err, sql.ErrNoRows) { return store.Task{}, false, nil }
        return store.Task{}, false, err
    }
    _ = json.Unmarshal([]byte(labelsStr), &t.Labels)
    return t, true, nil
}

func (s *sqliteStore) DeleteTask(id string) error {
    _, err := s.db.Exec(`DELETE FROM tasks WHERE task_id=?`, id)
    return err
}

func (s *sqliteStore) ListAssignmentsForTask(taskID string) ([]store.Assignment, error) {
    rows, err := s.db.Query(`SELECT instance_id, task_id, node_id, desired, artifact_url, start_cmd FROM assignments WHERE task_id=?`, taskID)
    if err != nil { return nil, err }
    defer rows.Close()
    var out []store.Assignment
    for rows.Next() {
        var a store.Assignment
        if err := rows.Scan(&a.InstanceID, &a.TaskID, &a.NodeID, &a.Desired, &a.ArtifactURL, &a.StartCmd); err != nil { return nil, err }
        out = append(out, a)
    }
    return out, rows.Err()
}

func (s *sqliteStore) DeleteAssignment(instanceID string) error {
    _, err := s.db.Exec(`DELETE FROM assignments WHERE instance_id=?`, instanceID)
    return err
}

func (s *sqliteStore) UpdateAssignmentDesired(instanceID string, desired store.DesiredState) error {
    _, err := s.db.Exec(`UPDATE assignments SET desired=? WHERE instance_id=?`, desired, instanceID)
    return err
}

func (s *sqliteStore) DeleteStatusesForInstance(instanceID string) error {
    _, err := s.db.Exec(`DELETE FROM statuses WHERE instance_id=?`, instanceID)
    return err
}

func (s *sqliteStore) DeleteAssignmentsForTask(taskID string) error {
    _, err := s.db.Exec(`DELETE FROM assignments WHERE task_id=?`, taskID)
    return err
}

func (s *sqliteStore) CountAssignmentsByArtifactPath(path string) (int, error) {
    row := s.db.QueryRow(`SELECT COUNT(1) FROM assignments WHERE artifact_url=?`, path)
    var n int
    if err := row.Scan(&n); err != nil { return 0, err }
    return n, nil
}

func (s *sqliteStore) CountAssignmentsForNode(nodeID string) (int, error) {
    row := s.db.QueryRow(`SELECT COUNT(1) FROM assignments WHERE node_id=?`, nodeID)
    var n int
    if err := row.Scan(&n); err != nil { return 0, err }
    return n, nil
}

func (s *sqliteStore) SaveArtifact(a store.Artifact) (string, error) {
    if a.ArtifactID == "" { a.ArtifactID = newID() }
    _, err := s.db.Exec(`INSERT INTO artifacts(artifact_id, app_name, version, path, sha256, size_bytes, created_at) VALUES(?,?,?,?,?,?,?)`,
        a.ArtifactID, a.AppName, a.Version, a.Path, a.SHA256, a.SizeBytes, a.CreatedAt,
    )
    if err != nil { return "", err }
    return a.ArtifactID, nil
}

func (s *sqliteStore) ListArtifacts() ([]store.Artifact, error) {
    rows, err := s.db.Query(`SELECT artifact_id, app_name, version, path, sha256, size_bytes, created_at FROM artifacts ORDER BY created_at DESC`)
    if err != nil { return nil, err }
    defer rows.Close()
    var out []store.Artifact
    for rows.Next() {
        var a store.Artifact
        if err := rows.Scan(&a.ArtifactID, &a.AppName, &a.Version, &a.Path, &a.SHA256, &a.SizeBytes, &a.CreatedAt); err != nil { return nil, err }
        out = append(out, a)
    }
    return out, rows.Err()
}

func (s *sqliteStore) GetArtifact(id string) (store.Artifact, bool, error) {
    row := s.db.QueryRow(`SELECT artifact_id, app_name, version, path, sha256, size_bytes, created_at FROM artifacts WHERE artifact_id=?`, id)
    var a store.Artifact
    if err := row.Scan(&a.ArtifactID, &a.AppName, &a.Version, &a.Path, &a.SHA256, &a.SizeBytes, &a.CreatedAt); err != nil {
        if errors.Is(err, sql.ErrNoRows) { return store.Artifact{}, false, nil }
        return store.Artifact{}, false, err
    }
    return a, true, nil
}

func (s *sqliteStore) DeleteArtifact(id string) error {
    _, err := s.db.Exec(`DELETE FROM artifacts WHERE artifact_id=?`, id)
    return err
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}



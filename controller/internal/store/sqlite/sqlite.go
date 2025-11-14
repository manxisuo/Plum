package sqlitestore

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"time"

	_ "modernc.org/sqlite"

	"github.com/manxisuo/plum/controller/internal/store"
)

type sqliteStore struct {
	db        *sql.DB
	healthTTL int64
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
	// 在线迁移：为已存在的deployments表添加status列（忽略错误，如果列已存在）
	_ = ensureColumn(db, "deployments", "status", "TEXT DEFAULT 'Stopped'")
	// 在线迁移：为已存在的artifacts表添加新列（忽略错误，如果列已存在）
	_ = ensureColumn(db, "artifacts", "type", "TEXT DEFAULT 'zip'")
	_ = ensureColumn(db, "artifacts", "image_repository", "TEXT")
	_ = ensureColumn(db, "artifacts", "image_tag", "TEXT")
	_ = ensureColumn(db, "artifacts", "port_mappings", "TEXT")
	ttl := int64(15)
	if v := os.Getenv("SERVICE_HEALTH_TTL_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			ttl = int64(n)
		}
	}
	return &sqliteStore{db: db, healthTTL: ttl}, nil
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS nodes (
			node_id TEXT PRIMARY KEY,
			ip TEXT,
			labels TEXT,
			last_seen INTEGER
		);`,
		// Deployments storage
		`CREATE TABLE IF NOT EXISTS deployments (
            deployment_id TEXT PRIMARY KEY,
			name TEXT,
			labels TEXT,
			status TEXT DEFAULT 'Stopped'
		);`,
		`CREATE TABLE IF NOT EXISTS assignments (
			instance_id TEXT PRIMARY KEY,
            deployment_id TEXT NOT NULL,
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
			created_at INTEGER,
			type TEXT DEFAULT 'zip',
			image_repository TEXT,
			image_tag TEXT,
			port_mappings TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS endpoints (
            service_name TEXT,
            instance_id TEXT,
            node_id TEXT,
            ip TEXT,
            port INTEGER,
            protocol TEXT,
            version TEXT,
            labels TEXT,
            healthy INTEGER,
            last_seen INTEGER,
            PRIMARY KEY(service_name, instance_id, ip, port, protocol)
        );`,
		// Tasks (Phase A minimal)
		`CREATE TABLE IF NOT EXISTS tasks (
            task_id TEXT PRIMARY KEY,
            name TEXT,
            executor TEXT,
            target_kind TEXT,
            target_ref TEXT,
            state TEXT,
            payload_json TEXT,
            result_json TEXT,
            error TEXT,
            timeout_sec INTEGER,
            max_retries INTEGER,
            attempt INTEGER,
            scheduled_on TEXT,
            created_at INTEGER,
            started_at INTEGER,
            finished_at INTEGER,
            labels TEXT,
            origin_task_id TEXT
        );`,
		// Workers for embedded executor (legacy HTTP-based)
		`CREATE TABLE IF NOT EXISTS workers (
            worker_id TEXT PRIMARY KEY,
            node_id TEXT,
            url TEXT,
            tasks TEXT,
            labels TEXT,
            capacity INTEGER,
            last_seen INTEGER
        );`,
		// Embedded Workers (new gRPC-based)
		`CREATE TABLE IF NOT EXISTS embedded_workers (
            worker_id TEXT PRIMARY KEY,
            node_id TEXT,
            instance_id TEXT,
            app_name TEXT,
            app_version TEXT,
            grpc_address TEXT,
            tasks TEXT,
            labels TEXT,
            last_seen INTEGER
        );`,
		// Workflows (definitions)
		`CREATE TABLE IF NOT EXISTS workflows (
            workflow_id TEXT PRIMARY KEY,
            name TEXT,
            labels TEXT
        );`,
		`CREATE TABLE IF NOT EXISTS workflow_steps (
            workflow_id TEXT,
            step_id TEXT,
            name TEXT,
            executor TEXT,
            target_kind TEXT,
            target_ref TEXT,
            labels TEXT,
            payload_json TEXT,
            timeout_sec INTEGER,
            max_retries INTEGER,
            ord INTEGER,
            definition_id TEXT,
            PRIMARY KEY(workflow_id, step_id)
        );`,
		// DAG Workflows (v2)
		`CREATE TABLE IF NOT EXISTS workflow_dags (
            workflow_id TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            version INTEGER DEFAULT 2,
            nodes TEXT NOT NULL,
            edges TEXT NOT NULL,
            start_nodes TEXT NOT NULL,
            created_at INTEGER
        );`,
		// Workflow runs and step runs
		`CREATE TABLE IF NOT EXISTS workflow_runs (
            run_id TEXT PRIMARY KEY,
            workflow_id TEXT,
            state TEXT,
            created_at INTEGER,
            started_at INTEGER,
            finished_at INTEGER
        );`,
		`CREATE TABLE IF NOT EXISTS step_runs (
            run_id TEXT,
            step_id TEXT,
            task_id TEXT,
            state TEXT,
            started_at INTEGER,
            finished_at INTEGER,
            ord INTEGER,
            PRIMARY KEY(run_id, step_id)
        );`,
		// TaskDefinitions
		`CREATE TABLE IF NOT EXISTS task_defs (
            def_id TEXT PRIMARY KEY,
            name TEXT,
            executor TEXT,
            target_kind TEXT,
            target_ref TEXT,
            labels TEXT,
            default_payload_json TEXT,
			created_at INTEGER
		);`,
		// DistributedKV
		`CREATE TABLE IF NOT EXISTS distributed_kv (
            namespace TEXT NOT NULL,
            key TEXT NOT NULL,
            value TEXT NOT NULL,
            type TEXT NOT NULL,
            updated_at INTEGER NOT NULL,
            PRIMARY KEY(namespace, key)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_kv_namespace ON distributed_kv(namespace);`,
		`CREATE INDEX IF NOT EXISTS idx_kv_updated ON distributed_kv(updated_at);`,
		// Resources
		`CREATE TABLE IF NOT EXISTS resources (
            resource_id TEXT PRIMARY KEY,
            node_id TEXT,
            type TEXT,
            url TEXT,
            last_seen INTEGER,
            created_at INTEGER
        );`,
		`CREATE TABLE IF NOT EXISTS resource_state_desc (
            resource_id TEXT,
            type TEXT,
            name TEXT,
            value TEXT,
            unit TEXT,
            PRIMARY KEY(resource_id, name),
            FOREIGN KEY(resource_id) REFERENCES resources(resource_id) ON DELETE CASCADE
        );`,
		`CREATE TABLE IF NOT EXISTS resource_op_desc (
            resource_id TEXT,
            type TEXT,
            name TEXT,
            value TEXT,
            unit TEXT,
            min_val TEXT,
            max_val TEXT,
            PRIMARY KEY(resource_id, name),
            FOREIGN KEY(resource_id) REFERENCES resources(resource_id) ON DELETE CASCADE
        );`,
		`CREATE TABLE IF NOT EXISTS resource_states (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            resource_id TEXT,
            timestamp INTEGER,
            states_json TEXT,
            FOREIGN KEY(resource_id) REFERENCES resources(resource_id) ON DELETE CASCADE
        );`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	// ensure new columns for upgrades
	if err := ensureColumn(db, "workflow_steps", "definition_id", "TEXT"); err != nil {
		return err
	}
	// Online schema upgrades (add columns if missing)
	if err := ensureColumn(db, "tasks", "origin_task_id", "TEXT"); err != nil {
		return err
	}
	if err := ensureColumn(db, "task_defs", "default_payload_json", "TEXT"); err != nil {
		return err
	}
	// Add payload_json column to workflow_steps if not exists (online schema upgrade)
	if err := ensureColumn(db, "workflow_steps", "payload_json", "TEXT"); err != nil {
		return err
	}
	// Add target_kind and target_ref columns to workflow_steps if not exists (online schema upgrade)
	if err := ensureColumn(db, "workflow_steps", "target_kind", "TEXT"); err != nil {
		return err
	}
	if err := ensureColumn(db, "workflow_steps", "target_ref", "TEXT"); err != nil {
		return err
	}
	if err := ensureColumn(db, "workflow_steps", "labels", "TEXT"); err != nil {
		return err
	}
	// Add app_name and app_version columns to assignments table
	if err := ensureColumn(db, "assignments", "app_name", "TEXT"); err != nil {
		return err
	}
	if err := ensureColumn(db, "assignments", "app_version", "TEXT"); err != nil {
		return err
	}
	return nil
}

// ensureColumn adds a column if not exists for a given table
func ensureColumn(db *sql.DB, table string, col string, typ string) error {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return err
	}
	defer rows.Close()
	present := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt interface{}
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == col {
			present = true
			break
		}
	}
	if present {
		return nil
	}
	_, err = db.Exec("ALTER TABLE " + table + " ADD COLUMN " + col + " " + typ)
	return err
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
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Node
	for rows.Next() {
		var n store.Node
		var labelsStr string
		var last int64
		if err := rows.Scan(&n.NodeID, &n.IP, &labelsStr, &last); err != nil {
			return nil, err
		}
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
	// 只返回状态为Running的部署的实例
	rows, err := s.db.Query(`
		SELECT a.instance_id, a.deployment_id, a.node_id, a.desired, a.artifact_url, a.start_cmd, a.app_name, a.app_version 
		FROM assignments a 
		INNER JOIN deployments d ON a.deployment_id = d.deployment_id 
		WHERE a.node_id=? AND COALESCE(d.status, 'Stopped') = 'Running'`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Assignment
	for rows.Next() {
		var a store.Assignment
		if err := rows.Scan(&a.InstanceID, &a.DeploymentID, &a.NodeID, &a.Desired, &a.ArtifactURL, &a.StartCmd, &a.AppName, &a.AppVersion); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *sqliteStore) GetAssignment(instanceID string) (store.Assignment, bool, error) {
	row := s.db.QueryRow(`SELECT instance_id, deployment_id, node_id, desired, artifact_url, start_cmd, app_name, app_version FROM assignments WHERE instance_id=?`, instanceID)
	var a store.Assignment
	if err := row.Scan(&a.InstanceID, &a.DeploymentID, &a.NodeID, &a.Desired, &a.ArtifactURL, &a.StartCmd, &a.AppName, &a.AppVersion); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.Assignment{}, false, nil
		}
		return store.Assignment{}, false, err
	}
	return a, true, nil
}

func (s *sqliteStore) AddAssignment(nodeID string, a store.Assignment) error {
	if nodeID == "" {
		return errors.New("nodeID required")
	}
	_, err := s.db.Exec(`INSERT INTO assignments(instance_id, deployment_id, node_id, desired, artifact_url, start_cmd, app_name, app_version) VALUES(?,?,?,?,?,?,?,?)`,
		a.InstanceID, a.DeploymentID, nodeID, a.Desired, a.ArtifactURL, a.StartCmd, a.AppName, a.AppVersion,
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
		if errors.Is(err, sql.ErrNoRows) {
			return store.InstanceStatus{}, false, nil
		}
		return store.InstanceStatus{}, false, err
	}
	st.Healthy = healthy != 0
	return st, true, nil
}

// Deployments helpers
func (s *sqliteStore) CreateDeployment(name string, labels map[string]string) (string, []string, error) {
	id := newID()
	labelsJSON, _ := json.Marshal(labels)
	// 创建时默认状态为Stopped
	if _, err := s.db.Exec(`INSERT INTO deployments(deployment_id, name, labels, status) VALUES(?,?,?,?)`, id, name, string(labelsJSON), store.DeploymentStopped); err != nil {
		return "", nil, err
	}
	return id, []string{}, nil
}

func (s *sqliteStore) NewInstanceID(deploymentID string) string {
	return deploymentID + "-" + newID()[:8]
}

func (s *sqliteStore) ListDeployments() ([]store.Deployment, error) {
	rows, err := s.db.Query(`SELECT deployment_id, name, labels, COALESCE(status, 'Stopped') FROM deployments ORDER BY rowid DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Deployment
	for rows.Next() {
		var t store.Deployment
		var labelsStr string
		var statusStr string
		if err := rows.Scan(&t.DeploymentID, &t.Name, &labelsStr, &statusStr); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(labelsStr), &t.Labels)
		t.Status = store.DeploymentStatus(statusStr)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *sqliteStore) GetDeployment(id string) (store.Deployment, bool, error) {
	row := s.db.QueryRow(`SELECT deployment_id, name, labels, COALESCE(status, 'Stopped') FROM deployments WHERE deployment_id=?`, id)
	var t store.Deployment
	var labelsStr string
	var statusStr string
	if err := row.Scan(&t.DeploymentID, &t.Name, &labelsStr, &statusStr); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.Deployment{}, false, nil
		}
		return store.Deployment{}, false, err
	}
	_ = json.Unmarshal([]byte(labelsStr), &t.Labels)
	t.Status = store.DeploymentStatus(statusStr)
	return t, true, nil
}

func (s *sqliteStore) DeleteDeployment(id string) error {
	_, err := s.db.Exec(`DELETE FROM deployments WHERE deployment_id=?`, id)
	return err
}

func (s *sqliteStore) UpdateDeploymentStatus(id string, status store.DeploymentStatus) error {
	_, err := s.db.Exec(`UPDATE deployments SET status=? WHERE deployment_id=?`, status, id)
	return err
}

func (s *sqliteStore) ListAssignmentsForDeployment(deploymentID string) ([]store.Assignment, error) {
	rows, err := s.db.Query(`SELECT instance_id, deployment_id, node_id, desired, artifact_url, start_cmd, app_name, app_version FROM assignments WHERE deployment_id=?`, deploymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Assignment
	for rows.Next() {
		var a store.Assignment
		if err := rows.Scan(&a.InstanceID, &a.DeploymentID, &a.NodeID, &a.Desired, &a.ArtifactURL, &a.StartCmd, &a.AppName, &a.AppVersion); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// No backward-compat: project early stage

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

func (s *sqliteStore) DeleteAssignmentsForDeployment(deploymentID string) error {
	_, err := s.db.Exec(`DELETE FROM assignments WHERE deployment_id=?`, deploymentID)
	return err
}

func (s *sqliteStore) CountAssignmentsByArtifactPath(path string) (int, error) {
	row := s.db.QueryRow(`SELECT COUNT(1) FROM assignments WHERE artifact_url=?`, path)
	var n int
	if err := row.Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (s *sqliteStore) CountAssignmentsForNode(nodeID string) (int, error) {
	row := s.db.QueryRow(`SELECT COUNT(1) FROM assignments WHERE node_id=?`, nodeID)
	var n int
	if err := row.Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (s *sqliteStore) SaveArtifact(a store.Artifact) (string, error) {
	if a.ArtifactID == "" {
		a.ArtifactID = newID()
	}
	// 如果 Type 为空，默认为 "zip"（向后兼容）
	if a.Type == "" {
		a.Type = "zip"
	}
	_, err := s.db.Exec(`INSERT INTO artifacts(artifact_id, app_name, version, path, sha256, size_bytes, created_at, type, image_repository, image_tag, port_mappings) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		a.ArtifactID, a.AppName, a.Version, a.Path, a.SHA256, a.SizeBytes, a.CreatedAt, a.Type, a.ImageRepository, a.ImageTag, a.PortMappings,
	)
	if err != nil {
		return "", err
	}
	return a.ArtifactID, nil
}

func (s *sqliteStore) ReplaceEndpointsForInstance(nodeID string, instanceID string, eps []store.Endpoint) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM endpoints WHERE instance_id=?`, instanceID)
	if err != nil {
		tx.Rollback()
		return err
	}
	for _, e := range eps {
		labelsJSON, _ := json.Marshal(e.Labels)
		if _, err := tx.Exec(`INSERT INTO endpoints(service_name, instance_id, node_id, ip, port, protocol, version, labels, healthy, last_seen) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			e.ServiceName, instanceID, nodeID, e.IP, e.Port, e.Protocol, e.Version, string(labelsJSON), boolToInt(e.Healthy), e.LastSeen,
		); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// ReplaceEndpointsForInstanceAndService 替换指定服务的端点（只删除该实例下指定服务的端点，保留其他服务的端点）
func (s *sqliteStore) ReplaceEndpointsForInstanceAndService(nodeID string, instanceID string, serviceName string, eps []store.Endpoint) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	// 只删除该实例下指定服务的端点
	_, err = tx.Exec(`DELETE FROM endpoints WHERE instance_id=? AND service_name=?`, instanceID, serviceName)
	if err != nil {
		tx.Rollback()
		return err
	}
	// 插入该服务的新端点
	for _, e := range eps {
		if e.ServiceName != serviceName {
			continue // 跳过不属于该服务的端点（理论上不应该发生）
		}
		labelsJSON, _ := json.Marshal(e.Labels)
		if _, err := tx.Exec(`INSERT INTO endpoints(service_name, instance_id, node_id, ip, port, protocol, version, labels, healthy, last_seen) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			e.ServiceName, instanceID, nodeID, e.IP, e.Port, e.Protocol, e.Version, string(labelsJSON), boolToInt(e.Healthy), e.LastSeen,
		); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *sqliteStore) UpdateEndpointHealthForInstance(instanceID string, eps []store.Endpoint) error {
	ts := time.Now().Unix()
	for _, e := range eps {
		_, err := s.db.Exec(`UPDATE endpoints SET healthy=?, last_seen=? WHERE instance_id=? AND service_name=? AND ip=? AND port=? AND protocol=?`,
			boolToInt(e.Healthy), ts, instanceID, e.ServiceName, e.IP, e.Port, e.Protocol,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *sqliteStore) TouchEndpointsForInstance(instanceID string, ts int64) error {
	_, err := s.db.Exec(`UPDATE endpoints SET last_seen=? WHERE instance_id=?`, ts, instanceID)
	return err
}

func (s *sqliteStore) DeleteEndpointsForInstance(instanceID string) error {
	_, err := s.db.Exec(`DELETE FROM endpoints WHERE instance_id=?`, instanceID)
	return err
}

// AddEndpoint 添加单个端点（如果已存在则更新，不删除其他端点）
func (s *sqliteStore) AddEndpoint(ep store.Endpoint) error {
	labelsJSON, _ := json.Marshal(ep.Labels)
	ts := time.Now().Unix()
	// 使用 INSERT OR REPLACE（基于主键）
	// 主键是 (service_name, instance_id, ip, port, protocol)
	_, err := s.db.Exec(`INSERT OR REPLACE INTO endpoints(service_name, instance_id, node_id, ip, port, protocol, version, labels, healthy, last_seen) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		ep.ServiceName, ep.InstanceID, ep.NodeID, ep.IP, ep.Port, ep.Protocol, ep.Version, string(labelsJSON), boolToInt(ep.Healthy), ts)
	return err
}

// DeleteEndpoint 删除单个端点
func (s *sqliteStore) DeleteEndpoint(serviceName string, instanceID string, ip string, port int, protocol string) error {
	_, err := s.db.Exec(`DELETE FROM endpoints WHERE service_name=? AND instance_id=? AND ip=? AND port=? AND protocol=?`,
		serviceName, instanceID, ip, port, protocol)
	return err
}

// UpdateEndpoint 更新单个端点信息
func (s *sqliteStore) UpdateEndpoint(serviceName string, instanceID string, oldIP string, oldPort int, oldProtocol string, ep store.Endpoint) error {
	labelsJSON, _ := json.Marshal(ep.Labels)
	ts := time.Now().Unix()
	// 如果主键字段发生变化，需要先删除旧记录，再插入新记录
	if ep.ServiceName != serviceName || ep.InstanceID != instanceID || ep.IP != oldIP || ep.Port != oldPort || ep.Protocol != oldProtocol {
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		// 删除旧记录
		_, err = tx.Exec(`DELETE FROM endpoints WHERE service_name=? AND instance_id=? AND ip=? AND port=? AND protocol=?`,
			serviceName, instanceID, oldIP, oldPort, oldProtocol)
		if err != nil {
			tx.Rollback()
			return err
		}
		// 插入新记录
		_, err = tx.Exec(`INSERT INTO endpoints(service_name, instance_id, node_id, ip, port, protocol, version, labels, healthy, last_seen) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			ep.ServiceName, ep.InstanceID, ep.NodeID, ep.IP, ep.Port, ep.Protocol, ep.Version, string(labelsJSON), boolToInt(ep.Healthy), ts)
		if err != nil {
			tx.Rollback()
			return err
		}
		return tx.Commit()
	}
	// 只更新非主键字段
	_, err := s.db.Exec(`UPDATE endpoints SET node_id=?, version=?, labels=?, healthy=?, last_seen=? WHERE service_name=? AND instance_id=? AND ip=? AND port=? AND protocol=?`,
		ep.NodeID, ep.Version, string(labelsJSON), boolToInt(ep.Healthy), ts, serviceName, instanceID, oldIP, oldPort, oldProtocol)
	return err
}

func (s *sqliteStore) ListEndpointsByService(serviceName string, version string, protocol string) ([]store.Endpoint, error) {
	ttlThreshold := time.Now().Unix() - s.healthTTL
	if ttlThreshold < 0 {
		ttlThreshold = 0
	}
	sqlStr := `SELECT service_name, instance_id, node_id, ip, port, protocol, version, labels, healthy, last_seen FROM endpoints WHERE service_name=? AND healthy=1 AND last_seen > ?`
	args := []any{serviceName, ttlThreshold}
	if version != "" {
		sqlStr += ` AND version=?`
		args = append(args, version)
	}
	if protocol != "" {
		sqlStr += ` AND protocol=?`
		args = append(args, protocol)
	}
	rows, err := s.db.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Endpoint
	for rows.Next() {
		var e store.Endpoint
		var labelsStr string
		var healthy int
		if err := rows.Scan(&e.ServiceName, &e.InstanceID, &e.NodeID, &e.IP, &e.Port, &e.Protocol, &e.Version, &labelsStr, &healthy, &e.LastSeen); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(labelsStr), &e.Labels)
		e.Healthy = healthy != 0
		out = append(out, e)
	}
	return out, rows.Err()
}

// ListAllEndpointsByService 列出服务的所有端点（包括不健康的，用于管理界面）
func (s *sqliteStore) ListAllEndpointsByService(serviceName string) ([]store.Endpoint, error) {
	rows, err := s.db.Query(`SELECT service_name, instance_id, node_id, ip, port, protocol, version, labels, healthy, last_seen FROM endpoints WHERE service_name=?`, serviceName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Endpoint
	for rows.Next() {
		var e store.Endpoint
		var labelsStr string
		var healthy int
		if err := rows.Scan(&e.ServiceName, &e.InstanceID, &e.NodeID, &e.IP, &e.Port, &e.Protocol, &e.Version, &labelsStr, &healthy, &e.LastSeen); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(labelsStr), &e.Labels)
		e.Healthy = healthy != 0
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *sqliteStore) ListServices() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT service_name FROM endpoints ORDER BY service_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

func (s *sqliteStore) ListArtifacts() ([]store.Artifact, error) {
	rows, err := s.db.Query(`SELECT artifact_id, app_name, version, path, sha256, size_bytes, created_at, type, image_repository, image_tag, port_mappings FROM artifacts ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Artifact
	for rows.Next() {
		var a store.Artifact
		if err := rows.Scan(&a.ArtifactID, &a.AppName, &a.Version, &a.Path, &a.SHA256, &a.SizeBytes, &a.CreatedAt, &a.Type, &a.ImageRepository, &a.ImageTag, &a.PortMappings); err != nil {
			return nil, err
		}
		// 向后兼容：如果 Type 为空，默认为 "zip"
		if a.Type == "" {
			a.Type = "zip"
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *sqliteStore) GetArtifact(id string) (store.Artifact, bool, error) {
	row := s.db.QueryRow(`SELECT artifact_id, app_name, version, path, sha256, size_bytes, created_at, type, image_repository, image_tag, port_mappings FROM artifacts WHERE artifact_id=?`, id)
	var a store.Artifact
	if err := row.Scan(&a.ArtifactID, &a.AppName, &a.Version, &a.Path, &a.SHA256, &a.SizeBytes, &a.CreatedAt, &a.Type, &a.ImageRepository, &a.ImageTag, &a.PortMappings); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.Artifact{}, false, nil
		}
		return store.Artifact{}, false, err
	}
	return a, true, nil
}

func (s *sqliteStore) GetArtifactByPath(path string) (store.Artifact, bool, error) {
	row := s.db.QueryRow(`SELECT artifact_id, app_name, version, path, sha256, size_bytes, created_at, type, image_repository, image_tag, port_mappings FROM artifacts WHERE path=?`, path)
	var a store.Artifact
	if err := row.Scan(&a.ArtifactID, &a.AppName, &a.Version, &a.Path, &a.SHA256, &a.SizeBytes, &a.CreatedAt, &a.Type, &a.ImageRepository, &a.ImageTag, &a.PortMappings); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.Artifact{}, false, nil
		}
		return store.Artifact{}, false, err
	}
	// 向后兼容：如果 Type 为空，默认为 "zip"
	if a.Type == "" {
		a.Type = "zip"
	}
	return a, true, nil
}

func (s *sqliteStore) DeleteArtifact(id string) error {
	_, err := s.db.Exec(`DELETE FROM artifacts WHERE artifact_id=?`, id)
	return err
}

// Workers (embedded)
func (s *sqliteStore) RegisterWorker(w store.Worker) error {
	tasksJSON, _ := json.Marshal(w.Tasks)
	labelsJSON, _ := json.Marshal(w.Labels)
	_, err := s.db.Exec(`INSERT INTO workers(worker_id, node_id, url, tasks, labels, capacity, last_seen) VALUES(?,?,?,?,?,?,?)
        ON CONFLICT(worker_id) DO UPDATE SET node_id=excluded.node_id, url=excluded.url, tasks=excluded.tasks, labels=excluded.labels, capacity=excluded.capacity, last_seen=excluded.last_seen`,
		w.WorkerID, w.NodeID, w.URL, string(tasksJSON), string(labelsJSON), w.Capacity, w.LastSeen,
	)
	return err
}

func (s *sqliteStore) HeartbeatWorker(workerID string, capacity int, lastSeen int64) error {
	_, err := s.db.Exec(`UPDATE workers SET capacity=?, last_seen=? WHERE worker_id=?`, capacity, lastSeen, workerID)
	return err
}

func (s *sqliteStore) ListWorkers() ([]store.Worker, error) {
	rows, err := s.db.Query(`SELECT worker_id, node_id, url, tasks, labels, capacity, last_seen FROM workers ORDER BY last_seen DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Worker
	for rows.Next() {
		var w store.Worker
		var tasksStr, labelsStr string
		if err := rows.Scan(&w.WorkerID, &w.NodeID, &w.URL, &tasksStr, &labelsStr, &w.Capacity, &w.LastSeen); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(tasksStr), &w.Tasks)
		_ = json.Unmarshal([]byte(labelsStr), &w.Labels)
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *sqliteStore) GetWorker(workerID string) (store.Worker, bool, error) {
	row := s.db.QueryRow(`SELECT worker_id, node_id, url, tasks, labels, capacity, last_seen FROM workers WHERE worker_id=?`, workerID)
	var w store.Worker
	var tasksStr, labelsStr string
	if err := row.Scan(&w.WorkerID, &w.NodeID, &w.URL, &tasksStr, &labelsStr, &w.Capacity, &w.LastSeen); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.Worker{}, false, nil
		}
		return store.Worker{}, false, err
	}
	_ = json.Unmarshal([]byte(tasksStr), &w.Tasks)
	_ = json.Unmarshal([]byte(labelsStr), &w.Labels)
	return w, true, nil
}

func (s *sqliteStore) DeleteWorker(workerID string) error {
	_, err := s.db.Exec(`DELETE FROM workers WHERE worker_id=?`, workerID)
	return err
}

// Tasks (Phase A minimal)
func (s *sqliteStore) CreateTask(t store.Task) (string, error) {
	if t.TaskID == "" {
		t.TaskID = newID()
	}
	labelsJSON, _ := json.Marshal(t.Labels)
	_, err := s.db.Exec(`INSERT INTO tasks(task_id, name, executor, target_kind, target_ref, state, payload_json, result_json, error, timeout_sec, max_retries, attempt, scheduled_on, created_at, started_at, finished_at, labels, origin_task_id) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.TaskID, t.Name, t.Executor, t.TargetKind, t.TargetRef, t.State, t.PayloadJSON, t.ResultJSON, t.Error, t.TimeoutSec, t.MaxRetries, t.Attempt, t.ScheduledOn, t.CreatedAt, t.StartedAt, t.FinishedAt, string(labelsJSON), t.OriginTaskID,
	)
	if err != nil {
		return "", err
	}
	return t.TaskID, nil
}

func (s *sqliteStore) GetTask(id string) (store.Task, bool, error) {
	row := s.db.QueryRow(`SELECT task_id, name, executor, target_kind, target_ref, state, payload_json, result_json, error, timeout_sec, max_retries, attempt, scheduled_on, created_at, started_at, finished_at, labels, origin_task_id FROM tasks WHERE task_id=?`, id)
	var t store.Task
	var labelsStr string
	if err := row.Scan(&t.TaskID, &t.Name, &t.Executor, &t.TargetKind, &t.TargetRef, &t.State, &t.PayloadJSON, &t.ResultJSON, &t.Error, &t.TimeoutSec, &t.MaxRetries, &t.Attempt, &t.ScheduledOn, &t.CreatedAt, &t.StartedAt, &t.FinishedAt, &labelsStr, &t.OriginTaskID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.Task{}, false, nil
		}
		return store.Task{}, false, err
	}
	_ = json.Unmarshal([]byte(labelsStr), &t.Labels)
	return t, true, nil
}

func (s *sqliteStore) ListTasks() ([]store.Task, error) {
	rows, err := s.db.Query(`SELECT task_id, name, executor, target_kind, target_ref, state, payload_json, result_json, error, timeout_sec, max_retries, attempt, scheduled_on, created_at, started_at, finished_at, labels, origin_task_id FROM tasks ORDER BY created_at DESC, task_id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Task
	for rows.Next() {
		var t store.Task
		var labelsStr string
		if err := rows.Scan(&t.TaskID, &t.Name, &t.Executor, &t.TargetKind, &t.TargetRef, &t.State, &t.PayloadJSON, &t.ResultJSON, &t.Error, &t.TimeoutSec, &t.MaxRetries, &t.Attempt, &t.ScheduledOn, &t.CreatedAt, &t.StartedAt, &t.FinishedAt, &labelsStr, &t.OriginTaskID); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(labelsStr), &t.Labels)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *sqliteStore) DeleteTask(id string) error {
	_, err := s.db.Exec(`DELETE FROM tasks WHERE task_id=?`, id)
	return err
}

func (s *sqliteStore) UpdateTaskState(id string, state string) error {
	_, err := s.db.Exec(`UPDATE tasks SET state=? WHERE task_id=?`, state, id)
	return err
}

func (s *sqliteStore) UpdateTaskRunning(id string, startedAt int64, scheduledOn string, attempt int) error {
	_, err := s.db.Exec(`UPDATE tasks SET state='Running', started_at=?, scheduled_on=?, attempt=? WHERE task_id=?`, startedAt, scheduledOn, attempt, id)
	return err
}

func (s *sqliteStore) UpdateTaskFinished(id string, state string, resultJSON string, errMsg string, finishedAt int64, attempt int) error {
	_, err := s.db.Exec(`UPDATE tasks SET state=?, result_json=?, error=?, finished_at=?, attempt=? WHERE task_id=?`, state, resultJSON, errMsg, finishedAt, attempt, id)
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

// Embedded Workers (new gRPC-based)
func (s *sqliteStore) RegisterEmbeddedWorker(w store.EmbeddedWorker) error {
	tasksJSON, _ := json.Marshal(w.Tasks)
	labelsJSON, _ := json.Marshal(w.Labels)
	_, err := s.db.Exec(`INSERT INTO embedded_workers(worker_id, node_id, instance_id, app_name, app_version, grpc_address, tasks, labels, last_seen) VALUES(?,?,?,?,?,?,?,?,?)
        ON CONFLICT(worker_id) DO UPDATE SET node_id=excluded.node_id, instance_id=excluded.instance_id, app_name=excluded.app_name, app_version=excluded.app_version, grpc_address=excluded.grpc_address, tasks=excluded.tasks, labels=excluded.labels, last_seen=excluded.last_seen`,
		w.WorkerID, w.NodeID, w.InstanceID, w.AppName, w.AppVersion, w.GRPCAddress, string(tasksJSON), string(labelsJSON), w.LastSeen,
	)
	return err
}

func (s *sqliteStore) HeartbeatEmbeddedWorker(workerID string, lastSeen int64) error {
	_, err := s.db.Exec(`UPDATE embedded_workers SET last_seen=? WHERE worker_id=?`, lastSeen, workerID)
	return err
}

func (s *sqliteStore) ListEmbeddedWorkers() ([]store.EmbeddedWorker, error) {
	rows, err := s.db.Query(`SELECT worker_id, node_id, instance_id, app_name, app_version, grpc_address, tasks, labels, last_seen FROM embedded_workers ORDER BY last_seen DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.EmbeddedWorker
	for rows.Next() {
		var w store.EmbeddedWorker
		var tasksStr, labelsStr string
		if err := rows.Scan(&w.WorkerID, &w.NodeID, &w.InstanceID, &w.AppName, &w.AppVersion, &w.GRPCAddress, &tasksStr, &labelsStr, &w.LastSeen); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(tasksStr), &w.Tasks)
		_ = json.Unmarshal([]byte(labelsStr), &w.Labels)
		out = append(out, w)
	}
	return out, nil
}

func (s *sqliteStore) GetEmbeddedWorker(workerID string) (store.EmbeddedWorker, bool, error) {
	row := s.db.QueryRow(`SELECT worker_id, node_id, instance_id, app_name, app_version, grpc_address, tasks, labels, last_seen FROM embedded_workers WHERE worker_id=?`, workerID)
	var w store.EmbeddedWorker
	var tasksStr, labelsStr string
	if err := row.Scan(&w.WorkerID, &w.NodeID, &w.InstanceID, &w.AppName, &w.AppVersion, &w.GRPCAddress, &tasksStr, &labelsStr, &w.LastSeen); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.EmbeddedWorker{}, false, nil
		}
		return store.EmbeddedWorker{}, false, err
	}
	_ = json.Unmarshal([]byte(tasksStr), &w.Tasks)
	_ = json.Unmarshal([]byte(labelsStr), &w.Labels)
	return w, true, nil
}

func (s *sqliteStore) DeleteEmbeddedWorker(workerID string) error {
	_, err := s.db.Exec(`DELETE FROM embedded_workers WHERE worker_id=?`, workerID)
	return err
}

// ---- Workflows (sequential MVP - placeholder implementations) ----
func (s *sqliteStore) CreateWorkflow(wf store.Workflow) (string, error) {
	if wf.WorkflowID == "" {
		wf.WorkflowID = newID()
	}
	labelsJSON, _ := json.Marshal(wf.Labels)
	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(`INSERT INTO workflows(workflow_id, name, labels) VALUES(?,?,?)`, wf.WorkflowID, wf.Name, string(labelsJSON)); err != nil {
		tx.Rollback()
		return "", err
	}
	for _, st := range wf.Steps {
		labelsJSON, _ := json.Marshal(st.Labels)
		if _, err := tx.Exec(`INSERT INTO workflow_steps(workflow_id, step_id, name, executor, target_kind, target_ref, labels, payload_json, timeout_sec, max_retries, ord, definition_id) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
			wf.WorkflowID, st.StepID, st.Name, st.Executor, st.TargetKind, st.TargetRef, string(labelsJSON), st.PayloadJSON, st.TimeoutSec, st.MaxRetries, st.Ord, st.DefinitionID,
		); err != nil {
			tx.Rollback()
			return "", err
		}
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return wf.WorkflowID, nil
}

func (s *sqliteStore) ListWorkflows() ([]store.Workflow, error) {
	rows, err := s.db.Query(`SELECT workflow_id, name, labels FROM workflows ORDER BY rowid DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Workflow
	for rows.Next() {
		var wf store.Workflow
		var labelsStr string
		if err := rows.Scan(&wf.WorkflowID, &wf.Name, &labelsStr); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(labelsStr), &wf.Labels)
		// load steps
		st, _ := s.ListWorkflowSteps(wf.WorkflowID)
		wf.Steps = st
		out = append(out, wf)
	}
	return out, rows.Err()
}

func (s *sqliteStore) GetWorkflow(id string) (store.Workflow, bool, error) {
	row := s.db.QueryRow(`SELECT workflow_id, name, labels FROM workflows WHERE workflow_id=?`, id)
	var wf store.Workflow
	var labelsStr string
	if err := row.Scan(&wf.WorkflowID, &wf.Name, &labelsStr); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.Workflow{}, false, nil
		}
		return store.Workflow{}, false, err
	}
	_ = json.Unmarshal([]byte(labelsStr), &wf.Labels)
	st, _ := s.ListWorkflowSteps(id)
	wf.Steps = st
	return wf, true, nil
}

func (s *sqliteStore) DeleteWorkflow(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete workflow steps
	if _, err := tx.Exec(`DELETE FROM workflow_steps WHERE workflow_id=?`, id); err != nil {
		return err
	}

	// Delete workflow runs and step runs
	runRows, err := tx.Query(`SELECT run_id FROM workflow_runs WHERE workflow_id=?`, id)
	if err != nil {
		return err
	}
	defer runRows.Close()

	for runRows.Next() {
		var runID string
		if err := runRows.Scan(&runID); err != nil {
			return err
		}
		// Delete step runs
		if _, err := tx.Exec(`DELETE FROM step_runs WHERE run_id=?`, runID); err != nil {
			return err
		}
	}

	// Delete workflow runs
	if _, err := tx.Exec(`DELETE FROM workflow_runs WHERE workflow_id=?`, id); err != nil {
		return err
	}

	// Delete workflow
	if _, err := tx.Exec(`DELETE FROM workflows WHERE workflow_id=?`, id); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *sqliteStore) CreateWorkflowRun(workflowID string) (string, error) {
	runID := newID()
	_, err := s.db.Exec(`INSERT INTO workflow_runs(run_id, workflow_id, state, created_at, started_at, finished_at) VALUES(?,?,?,?,?,?)`,
		runID, workflowID, "Pending", time.Now().Unix(), 0, 0,
	)
	if err != nil {
		return "", err
	}
	return runID, nil
}

func (s *sqliteStore) CreateWorkflowRunWithID(run store.WorkflowRun) error {
	_, err := s.db.Exec(`INSERT INTO workflow_runs(run_id, workflow_id, state, created_at, started_at, finished_at) VALUES(?,?,?,?,?,?)`,
		run.RunID, run.WorkflowID, run.State, run.CreatedAt, run.StartedAt, run.FinishedAt,
	)
	return err
}

func (s *sqliteStore) GetWorkflowRun(runID string) (store.WorkflowRun, bool, error) {
	row := s.db.QueryRow(`SELECT run_id, workflow_id, state, created_at, started_at, finished_at FROM workflow_runs WHERE run_id=?`, runID)
	var r store.WorkflowRun
	if err := row.Scan(&r.RunID, &r.WorkflowID, &r.State, &r.CreatedAt, &r.StartedAt, &r.FinishedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.WorkflowRun{}, false, nil
		}
		return store.WorkflowRun{}, false, err
	}
	return r, true, nil
}

func (s *sqliteStore) ListWorkflowRuns() ([]store.WorkflowRun, error) {
	rows, err := s.db.Query(`SELECT run_id, workflow_id, state, created_at, started_at, finished_at FROM workflow_runs ORDER BY created_at DESC, run_id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.WorkflowRun
	for rows.Next() {
		var r store.WorkflowRun
		if err := rows.Scan(&r.RunID, &r.WorkflowID, &r.State, &r.CreatedAt, &r.StartedAt, &r.FinishedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *sqliteStore) ListWorkflowRunsByWorkflow(workflowID string) ([]store.WorkflowRun, error) {
	rows, err := s.db.Query(`SELECT run_id, workflow_id, state, created_at, started_at, finished_at FROM workflow_runs WHERE workflow_id=? ORDER BY created_at DESC`, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.WorkflowRun
	for rows.Next() {
		var r store.WorkflowRun
		if err := rows.Scan(&r.RunID, &r.WorkflowID, &r.State, &r.CreatedAt, &r.StartedAt, &r.FinishedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *sqliteStore) ListWorkflowSteps(id string) ([]store.WorkflowStep, error) {
	rows, err := s.db.Query(`SELECT step_id, name, executor, target_kind, target_ref, labels, payload_json, timeout_sec, max_retries, ord, definition_id FROM workflow_steps WHERE workflow_id=? ORDER BY ord ASC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.WorkflowStep
	for rows.Next() {
		var st store.WorkflowStep
		var labelsStr string
		var payloadStr sql.NullString
		if err := rows.Scan(&st.StepID, &st.Name, &st.Executor, &st.TargetKind, &st.TargetRef, &labelsStr, &payloadStr, &st.TimeoutSec, &st.MaxRetries, &st.Ord, &st.DefinitionID); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(labelsStr), &st.Labels)
		if payloadStr.Valid {
			st.PayloadJSON = payloadStr.String
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

func (s *sqliteStore) ListStepRuns(runID string) ([]store.StepRun, error) {
	rows, err := s.db.Query(`SELECT run_id, step_id, task_id, state, started_at, finished_at, ord FROM step_runs WHERE run_id=? ORDER BY ord ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.StepRun
	for rows.Next() {
		var sr store.StepRun
		if err := rows.Scan(&sr.RunID, &sr.StepID, &sr.TaskID, &sr.State, &sr.StartedAt, &sr.FinishedAt, &sr.Ord); err != nil {
			return nil, err
		}
		out = append(out, sr)
	}
	return out, rows.Err()
}

func (s *sqliteStore) InsertStepRun(sr store.StepRun) error {
	_, err := s.db.Exec(`INSERT INTO step_runs(run_id, step_id, task_id, state, started_at, finished_at, ord) VALUES(?,?,?,?,?,?,?)`,
		sr.RunID, sr.StepID, sr.TaskID, sr.State, sr.StartedAt, sr.FinishedAt, sr.Ord,
	)
	return err
}

func (s *sqliteStore) UpdateStepRunTask(runID string, stepID string, taskID string, state string, startedAt int64) error {
	_, err := s.db.Exec(`UPDATE step_runs SET task_id=?, state=?, started_at=? WHERE run_id=? AND step_id=?`, taskID, state, startedAt, runID, stepID)
	return err
}

func (s *sqliteStore) UpdateStepRunFinished(runID string, stepID string, state string, finishedAt int64) error {
	_, err := s.db.Exec(`UPDATE step_runs SET state=?, finished_at=? WHERE run_id=? AND step_id=?`, state, finishedAt, runID, stepID)
	return err
}

func (s *sqliteStore) UpdateWorkflowRunState(runID string, state string, ts int64) error {
	col := "started_at"
	if state == "Succeeded" || state == "Failed" || state == "Canceled" {
		col = "finished_at"
	}
	_, err := s.db.Exec(`UPDATE workflow_runs SET state=?, `+col+`=? WHERE run_id=?`, state, ts, runID)
	return err
}

func (s *sqliteStore) DeleteWorkflowRun(runID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete step runs first
	if _, err := tx.Exec(`DELETE FROM step_runs WHERE run_id=?`, runID); err != nil {
		return err
	}

	// Delete workflow run
	if _, err := tx.Exec(`DELETE FROM workflow_runs WHERE run_id=?`, runID); err != nil {
		return err
	}

	return tx.Commit()
}

// TaskDefinition CRUD
func (s *sqliteStore) CreateTaskDef(td store.TaskDefinition) (string, error) {
	if td.DefID == "" {
		td.DefID = newID()
	}
	labelsJSON, _ := json.Marshal(td.Labels)
	_, err := s.db.Exec(`INSERT INTO task_defs(def_id, name, executor, target_kind, target_ref, labels, default_payload_json, created_at) VALUES(?,?,?,?,?,?,?,?)`,
		td.DefID, td.Name, td.Executor, td.TargetKind, td.TargetRef, string(labelsJSON), td.DefaultPayloadJSON, time.Now().Unix(),
	)
	if err != nil {
		return "", err
	}
	return td.DefID, nil
}

func (s *sqliteStore) GetTaskDef(id string) (store.TaskDefinition, bool, error) {
	row := s.db.QueryRow(`SELECT def_id, name, executor, target_kind, target_ref, labels, default_payload_json, created_at FROM task_defs WHERE def_id=?`, id)
	var td store.TaskDefinition
	var labelsStr string
	if err := row.Scan(&td.DefID, &td.Name, &td.Executor, &td.TargetKind, &td.TargetRef, &labelsStr, &td.DefaultPayloadJSON, &td.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.TaskDefinition{}, false, nil
		}
		return store.TaskDefinition{}, false, err
	}
	_ = json.Unmarshal([]byte(labelsStr), &td.Labels)
	return td, true, nil
}

func (s *sqliteStore) GetTaskDefByName(name string) (store.TaskDefinition, bool, error) {
	row := s.db.QueryRow(`SELECT def_id, name, executor, target_kind, target_ref, labels, default_payload_json, created_at FROM task_defs WHERE name=?`, name)
	var td store.TaskDefinition
	var labelsStr string
	if err := row.Scan(&td.DefID, &td.Name, &td.Executor, &td.TargetKind, &td.TargetRef, &labelsStr, &td.DefaultPayloadJSON, &td.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.TaskDefinition{}, false, nil
		}
		return store.TaskDefinition{}, false, err
	}
	_ = json.Unmarshal([]byte(labelsStr), &td.Labels)
	return td, true, nil
}

func (s *sqliteStore) ListTaskDefs() ([]store.TaskDefinition, error) {
	rows, err := s.db.Query(`SELECT def_id, name, executor, target_kind, target_ref, labels, default_payload_json, created_at FROM task_defs ORDER BY created_at DESC, def_id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.TaskDefinition
	for rows.Next() {
		var td store.TaskDefinition
		var labelsStr string
		if err := rows.Scan(&td.DefID, &td.Name, &td.Executor, &td.TargetKind, &td.TargetRef, &labelsStr, &td.DefaultPayloadJSON, &td.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(labelsStr), &td.Labels)
		out = append(out, td)
	}
	return out, rows.Err()
}

func (s *sqliteStore) DeleteTaskDef(id string) error {
	_, err := s.db.Exec(`DELETE FROM task_defs WHERE def_id=?`, id)
	return err
}

func (s *sqliteStore) CountTasksByOrigin(defID string) (int, error) {
	row := s.db.QueryRow(`SELECT COUNT(1) FROM tasks WHERE origin_task_id=?`, defID)
	var n int
	if err := row.Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// ---- DistributedKV ----

func (s *sqliteStore) PutKV(namespace, key, value, valueType string) error {
	_, err := s.db.Exec(`
		INSERT INTO distributed_kv(namespace, key, value, type, updated_at)
		VALUES(?, ?, ?, ?, ?)
		ON CONFLICT(namespace, key) DO UPDATE SET
			value=excluded.value,
			type=excluded.type,
			updated_at=excluded.updated_at
	`, namespace, key, value, valueType, time.Now().Unix())
	return err
}

func (s *sqliteStore) GetKV(namespace, key string) (store.DistributedKV, bool, error) {
	row := s.db.QueryRow(`SELECT namespace, key, value, type, updated_at FROM distributed_kv WHERE namespace=? AND key=?`, namespace, key)
	var kv store.DistributedKV
	if err := row.Scan(&kv.Namespace, &kv.Key, &kv.Value, &kv.Type, &kv.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.DistributedKV{}, false, nil
		}
		return store.DistributedKV{}, false, err
	}
	return kv, true, nil
}

func (s *sqliteStore) DeleteKV(namespace, key string) error {
	_, err := s.db.Exec(`DELETE FROM distributed_kv WHERE namespace=? AND key=?`, namespace, key)
	return err
}

func (s *sqliteStore) ListKVByNamespace(namespace string) ([]store.DistributedKV, error) {
	rows, err := s.db.Query(`SELECT namespace, key, value, type, updated_at FROM distributed_kv WHERE namespace=? ORDER BY key ASC`, namespace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.DistributedKV
	for rows.Next() {
		var kv store.DistributedKV
		if err := rows.Scan(&kv.Namespace, &kv.Key, &kv.Value, &kv.Type, &kv.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, kv)
	}
	return out, rows.Err()
}

func (s *sqliteStore) ListKVByPrefix(namespace, prefix string) ([]store.DistributedKV, error) {
	rows, err := s.db.Query(`SELECT namespace, key, value, type, updated_at FROM distributed_kv WHERE namespace=? AND key LIKE ? ORDER BY key ASC`, namespace, prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.DistributedKV
	for rows.Next() {
		var kv store.DistributedKV
		if err := rows.Scan(&kv.Namespace, &kv.Key, &kv.Value, &kv.Type, &kv.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, kv)
	}
	return out, rows.Err()
}

func (s *sqliteStore) PutKVBatch(namespace string, kvs []store.DistributedKV) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO distributed_kv(namespace, key, value, type, updated_at)
		VALUES(?, ?, ?, ?, ?)
		ON CONFLICT(namespace, key) DO UPDATE SET
			value=excluded.value,
			type=excluded.type,
			updated_at=excluded.updated_at
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().Unix()
	for _, kv := range kvs {
		if _, err := stmt.Exec(namespace, kv.Key, kv.Value, kv.Type, now); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *sqliteStore) DeleteNamespace(namespace string) error {
	_, err := s.db.Exec(`DELETE FROM distributed_kv WHERE namespace=?`, namespace)
	return err
}

func (s *sqliteStore) ListAllNamespaces() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT namespace FROM distributed_kv ORDER BY namespace`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var namespaces []string
	for rows.Next() {
		var ns string
		if err := rows.Scan(&ns); err != nil {
			return nil, err
		}
		namespaces = append(namespaces, ns)
	}
	return namespaces, rows.Err()
}

func (s *sqliteStore) ListKeysByNamespace(namespace string) ([]string, error) {
	rows, err := s.db.Query(`SELECT key FROM distributed_kv WHERE namespace=? ORDER BY key`, namespace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

// Close 关闭数据库连接
func (s *sqliteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

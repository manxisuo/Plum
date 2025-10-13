package sqlitestore

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/manxisuo/plum/controller/internal/store"
)

// DAG Workflows实现

func (s *sqliteStore) CreateWorkflowDAG(dag store.WorkflowDAG) (string, error) {
	if dag.WorkflowID == "" {
		dag.WorkflowID = newID()
	}
	if dag.CreatedAt == 0 {
		dag.CreatedAt = time.Now().Unix()
	}
	dag.Version = 2 // DAG版本

	// 序列化nodes和edges
	nodesJSON, err := json.Marshal(dag.Nodes)
	if err != nil {
		return "", err
	}

	edgesJSON, err := json.Marshal(dag.Edges)
	if err != nil {
		return "", err
	}

	startNodesJSON, err := json.Marshal(dag.StartNodes)
	if err != nil {
		return "", err
	}

	_, err = s.db.Exec(`
		INSERT INTO workflow_dags(workflow_id, name, version, nodes, edges, start_nodes, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)
	`, dag.WorkflowID, dag.Name, dag.Version, string(nodesJSON), string(edgesJSON), string(startNodesJSON), dag.CreatedAt)

	if err != nil {
		return "", err
	}
	return dag.WorkflowID, nil
}

func (s *sqliteStore) GetWorkflowDAG(id string) (store.WorkflowDAG, bool, error) {
	row := s.db.QueryRow(`
		SELECT workflow_id, name, version, nodes, edges, start_nodes, created_at
		FROM workflow_dags WHERE workflow_id=?
	`, id)

	var dag store.WorkflowDAG
	var nodesStr, edgesStr, startNodesStr string

	err := row.Scan(&dag.WorkflowID, &dag.Name, &dag.Version, &nodesStr, &edgesStr, &startNodesStr, &dag.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return store.WorkflowDAG{}, false, nil
	}
	if err != nil {
		return store.WorkflowDAG{}, false, err
	}

	// 反序列化
	if err := json.Unmarshal([]byte(nodesStr), &dag.Nodes); err != nil {
		return store.WorkflowDAG{}, false, err
	}
	if err := json.Unmarshal([]byte(edgesStr), &dag.Edges); err != nil {
		return store.WorkflowDAG{}, false, err
	}
	if err := json.Unmarshal([]byte(startNodesStr), &dag.StartNodes); err != nil {
		return store.WorkflowDAG{}, false, err
	}

	return dag, true, nil
}

func (s *sqliteStore) ListWorkflowDAGs() ([]store.WorkflowDAG, error) {
	rows, err := s.db.Query(`
		SELECT workflow_id, name, version, nodes, edges, start_nodes, created_at
		FROM workflow_dags ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dags []store.WorkflowDAG
	for rows.Next() {
		var dag store.WorkflowDAG
		var nodesStr, edgesStr, startNodesStr string

		if err := rows.Scan(&dag.WorkflowID, &dag.Name, &dag.Version, &nodesStr, &edgesStr, &startNodesStr, &dag.CreatedAt); err != nil {
			return nil, err
		}

		_ = json.Unmarshal([]byte(nodesStr), &dag.Nodes)
		_ = json.Unmarshal([]byte(edgesStr), &dag.Edges)
		_ = json.Unmarshal([]byte(startNodesStr), &dag.StartNodes)

		dags = append(dags, dag)
	}

	return dags, rows.Err()
}

func (s *sqliteStore) DeleteWorkflowDAG(id string) error {
	_, err := s.db.Exec(`DELETE FROM workflow_dags WHERE workflow_id=?`, id)
	return err
}

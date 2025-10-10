package sqlitestore

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/manxisuo/plum/controller/internal/store"
)

// ---- Resources ----
func (s *sqliteStore) RegisterResource(r store.Resource) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`INSERT OR REPLACE INTO resources(resource_id, node_id, type, url, last_seen, created_at) VALUES(?,?,?,?,?,?)`,
		r.ResourceID, r.NodeID, r.Type, r.URL, r.LastSeen, r.CreatedAt)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM resource_state_desc WHERE resource_id=?`, r.ResourceID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM resource_op_desc WHERE resource_id=?`, r.ResourceID)
	if err != nil {
		return err
	}

	for _, state := range r.StateDesc {
		_, err = tx.Exec(`INSERT INTO resource_state_desc(resource_id, type, name, value, unit) VALUES(?,?,?,?,?)`,
			r.ResourceID, state.Type, state.Name, state.Value, state.Unit)
		if err != nil {
			return err
		}
	}

	for _, op := range r.OpDesc {
		_, err = tx.Exec(`INSERT INTO resource_op_desc(resource_id, type, name, value, unit, min_val, max_val) VALUES(?,?,?,?,?,?,?)`,
			r.ResourceID, op.Type, op.Name, op.Value, op.Unit, op.Min, op.Max)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *sqliteStore) HeartbeatResource(resourceID string, lastSeen int64) error {
	_, err := s.db.Exec(`UPDATE resources SET last_seen=? WHERE resource_id=?`, lastSeen, resourceID)
	return err
}

func (s *sqliteStore) ListResources() ([]store.Resource, error) {
	rows, err := s.db.Query(`SELECT resource_id, node_id, type, url, last_seen, created_at FROM resources ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []store.Resource
	for rows.Next() {
		var r store.Resource
		if err := rows.Scan(&r.ResourceID, &r.NodeID, &r.Type, &r.URL, &r.LastSeen, &r.CreatedAt); err != nil {
			return nil, err
		}

		stateRows, err := s.db.Query(`SELECT type, name, value, unit FROM resource_state_desc WHERE resource_id=?`, r.ResourceID)
		if err != nil {
			return nil, err
		}
		for stateRows.Next() {
			var state store.ResourceStateDesc
			if err := stateRows.Scan(&state.Type, &state.Name, &state.Value, &state.Unit); err != nil {
				stateRows.Close()
				return nil, err
			}
			r.StateDesc = append(r.StateDesc, state)
		}
		stateRows.Close()

		opRows, err := s.db.Query(`SELECT type, name, value, unit, min_val, max_val FROM resource_op_desc WHERE resource_id=?`, r.ResourceID)
		if err != nil {
			return nil, err
		}
		for opRows.Next() {
			var op store.ResourceOpDesc
			if err := opRows.Scan(&op.Type, &op.Name, &op.Value, &op.Unit, &op.Min, &op.Max); err != nil {
				opRows.Close()
				return nil, err
			}
			r.OpDesc = append(r.OpDesc, op)
		}
		opRows.Close()

		list = append(list, r)
	}
	return list, nil
}

func (s *sqliteStore) GetResource(id string) (store.Resource, bool, error) {
	row := s.db.QueryRow(`SELECT resource_id, node_id, type, url, last_seen, created_at FROM resources WHERE resource_id=?`, id)
	var r store.Resource
	if err := row.Scan(&r.ResourceID, &r.NodeID, &r.Type, &r.URL, &r.LastSeen, &r.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.Resource{}, false, nil
		}
		return store.Resource{}, false, err
	}

	stateRows, err := s.db.Query(`SELECT type, name, value, unit FROM resource_state_desc WHERE resource_id=?`, id)
	if err != nil {
		return store.Resource{}, false, err
	}
	defer stateRows.Close()
	for stateRows.Next() {
		var state store.ResourceStateDesc
		if err := stateRows.Scan(&state.Type, &state.Name, &state.Value, &state.Unit); err != nil {
			return store.Resource{}, false, err
		}
		r.StateDesc = append(r.StateDesc, state)
	}

	opRows, err := s.db.Query(`SELECT type, name, value, unit, min_val, max_val FROM resource_op_desc WHERE resource_id=?`, id)
	if err != nil {
		return store.Resource{}, false, err
	}
	defer opRows.Close()
	for opRows.Next() {
		var op store.ResourceOpDesc
		if err := opRows.Scan(&op.Type, &op.Name, &op.Value, &op.Unit, &op.Min, &op.Max); err != nil {
			return store.Resource{}, false, err
		}
		r.OpDesc = append(r.OpDesc, op)
	}

	return r, true, nil
}

func (s *sqliteStore) DeleteResource(id string) error {
	_, err := s.db.Exec(`DELETE FROM resources WHERE resource_id=?`, id)
	return err
}

func (s *sqliteStore) SubmitResourceState(rs store.ResourceState) error {
	statesJSON, err := json.Marshal(rs.States)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO resource_states(resource_id, timestamp, states_json) VALUES(?,?,?)`,
		rs.ResourceID, rs.Timestamp, string(statesJSON))
	return err
}

func (s *sqliteStore) ListResourceStates(resourceID string, limit int) ([]store.ResourceState, error) {
	query := `SELECT resource_id, timestamp, states_json FROM resource_states WHERE resource_id=? ORDER BY timestamp DESC`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []store.ResourceState
	for rows.Next() {
		var rs store.ResourceState
		var statesJSON string
		if err := rows.Scan(&rs.ResourceID, &rs.Timestamp, &statesJSON); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(statesJSON), &rs.States); err != nil {
			return nil, err
		}
		list = append(list, rs)
	}
	return list, nil
}

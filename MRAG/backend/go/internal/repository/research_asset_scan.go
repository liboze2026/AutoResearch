package repository

import (
	"database/sql"
	"encoding/json"
)

type researchAssetScanner interface {
	Scan(dest ...interface{}) error
}

func decodeJSON(raw []byte, dest interface{}) {
	if len(raw) == 0 {
		return
	}
	_ = json.Unmarshal(raw, dest)
}

func nullableString(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return v.String
}

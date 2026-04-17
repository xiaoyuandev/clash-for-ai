package storage

type SQLite struct {
	Path string
}

func NewSQLite(path string) *SQLite {
	return &SQLite{Path: path}
}

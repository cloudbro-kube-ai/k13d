package db

import (
	"strings"
)

// dialect.go centralizes the SQL differences between the supported backends
// (SQLite, PostgreSQL, MariaDB/MySQL) so per-table code can build portable DDL
// and DML instead of hard-coding SQLite-only syntax.
//
// Placeholder note: SQLite and MySQL/MariaDB use positional `?` placeholders,
// but the lib/pq PostgreSQL driver requires ordinal `$1, $2, ...` placeholders.
// Queries in this package are written with `?` and passed through Rebind before
// execution so the same query string works on every backend.

// Rebind converts a query written with `?` placeholders into the placeholder
// style required by the current backend. For SQLite and MySQL/MariaDB the query
// is returned unchanged; for PostgreSQL each `?` becomes `$1`, `$2`, ... in
// order. `?` characters inside single-quoted string literals are left alone.
//
// It is a no-op for the default (SQLite) backend, so existing `?`-based queries
// keep identical behavior there.
func Rebind(query string) string {
	return rebindFor(GetDBType(), query)
}

// rebindFor is the backend-parameterized core of Rebind, exposed for testing
// without mutating the global currentDBType.
func rebindFor(dbType DBType, query string) string {
	if dbType != DBTypePostgres {
		return query
	}

	var b strings.Builder
	b.Grow(len(query) + 8)
	n := 0
	inSingle := false
	for i := 0; i < len(query); i++ {
		c := query[i]
		switch {
		case c == '\'':
			// Toggle string-literal state, honoring the SQL '' escape by
			// treating a doubled quote as staying inside the literal.
			if inSingle && i+1 < len(query) && query[i+1] == '\'' {
				b.WriteByte(c)
				b.WriteByte(query[i+1])
				i++
				continue
			}
			inSingle = !inSingle
			b.WriteByte(c)
		case c == '?' && !inSingle:
			n++
			b.WriteByte('$')
			b.WriteString(itoa(n))
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// itoa is a tiny allocation-light integer formatter for placeholder indices.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// autoIncrementPK returns the primary-key column definition for an
// auto-incrementing integer id on the current backend.
func autoIncrementPK() string {
	switch currentDBType {
	case DBTypePostgres:
		return "id SERIAL PRIMARY KEY"
	case DBTypeMariaDB, DBTypeMySQL:
		return "id BIGINT AUTO_INCREMENT PRIMARY KEY"
	default: // SQLite
		return "id INTEGER PRIMARY KEY AUTOINCREMENT"
	}
}

// timestampDefault returns the column type + default for a creation/update
// timestamp on the current backend.
func timestampDefault() string {
	switch currentDBType {
	case DBTypePostgres:
		return "TIMESTAMP DEFAULT CURRENT_TIMESTAMP"
	default: // SQLite, MySQL/MariaDB both accept DATETIME
		return "DATETIME DEFAULT CURRENT_TIMESTAMP"
	}
}

// tableSuffix returns any trailing table options required by the backend
// (MySQL needs an explicit engine/charset; the others need nothing).
func tableSuffix() string {
	switch currentDBType {
	case DBTypeMariaDB, DBTypeMySQL:
		return " ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"
	default:
		return ""
	}
}

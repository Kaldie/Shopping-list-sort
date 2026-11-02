package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
)

// One jsonb document per row. Each entity kind has its own table, so rows can
// be inspected and queried, while the domain structs stay the single source
// of truth for the shape of a document.
const schema = `
CREATE TABLE IF NOT EXISTS stores    (id text PRIMARY KEY, doc jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS lists     (id text PRIMARY KEY, doc jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS templates (id text PRIMARY KEY, doc jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS users     (id text PRIMARY KEY, doc jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS sessions  (id text PRIMARY KEY, doc jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS trips     (id bigserial PRIMARY KEY, doc jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS meta      (key text PRIMARY KEY, value bigint NOT NULL);
`

type digest = [sha256.Size]byte

// pgStore writes only the rows that changed since the last save: each row's
// document hash is remembered, so a save upserts changed or new rows, deletes
// vanished ones, and appends new trips (trips are append-only).
type pgStore struct {
	db         *sql.DB
	known      map[string]map[string]digest // table -> row id -> hash of the doc last written
	tripsSaved int
	nextID     int
}

// OpenPostgres loads the data set from Postgres, creating the schema if
// needed. If the database holds no data yet and importPath names an existing
// JSON file, that file is imported once.
func OpenPostgres(dsn, importPath string) (*Repository, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	if _, err := db.ExecContext(ctx, schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	p := &pgStore{db: db, known: map[string]map[string]digest{}}
	r := &Repository{p: p}
	if err := p.load(ctx, &r.d); err != nil {
		db.Close()
		return nil, err
	}

	if p.empty(&r.d) && importPath != "" {
		var imported data
		switch err := readJSON(importPath, &imported); {
		case err == nil:
			r.d = imported
			r.normalize()
			p.reset() // nothing is in the database yet: write every row
			if err := r.save(); err != nil {
				db.Close()
				return nil, fmt.Errorf("import %s: %w", importPath, err)
			}
			log.Printf("imported %s into postgres (%d stores, %d lists, %d users, %d trips)",
				importPath, len(r.d.Stores), len(r.d.Lists), len(r.d.Users), len(r.d.Trips))
			return r, nil
		case !errors.Is(err, os.ErrNotExist):
			db.Close()
			return nil, err
		}
	}
	r.normalize()
	return r, nil
}

func (p *pgStore) empty(d *data) bool {
	return len(d.Stores) == 0 && len(d.Lists) == 0 && len(d.Templates) == 0 &&
		len(d.Users) == 0 && len(d.Trips) == 0 && d.NextID == 0
}

func (p *pgStore) reset() {
	p.known = map[string]map[string]digest{}
	p.tripsSaved = 0
	p.nextID = 0
}

func (p *pgStore) load(ctx context.Context, d *data) error {
	initMaps(d)
	if err := loadTable(ctx, p, "stores", d.Stores); err != nil {
		return err
	}
	if err := loadTable(ctx, p, "lists", d.Lists); err != nil {
		return err
	}
	if err := loadTable(ctx, p, "templates", d.Templates); err != nil {
		return err
	}
	if err := loadTable(ctx, p, "users", d.Users); err != nil {
		return err
	}
	if err := loadTable(ctx, p, "sessions", d.Sessions); err != nil {
		return err
	}

	rows, err := p.db.QueryContext(ctx, `SELECT doc FROM trips ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var b []byte
		var t Trip
		if err := rows.Scan(&b); err != nil {
			return err
		}
		if err := json.Unmarshal(b, &t); err != nil {
			return fmt.Errorf("decode trip: %w", err)
		}
		d.Trips = append(d.Trips, t)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	p.tripsSaved = len(d.Trips)

	err = p.db.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = 'next_id'`).Scan(&d.NextID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	p.nextID = d.NextID
	return nil
}

func loadTable[T any](ctx context.Context, p *pgStore, table string, into map[string]*T) error {
	rows, err := p.db.QueryContext(ctx, `SELECT id, doc FROM `+table)
	if err != nil {
		return err
	}
	defer rows.Close()
	known := map[string]digest{}
	for rows.Next() {
		var id string
		var b []byte
		if err := rows.Scan(&id, &b); err != nil {
			return err
		}
		v := new(T)
		if err := json.Unmarshal(b, v); err != nil {
			return fmt.Errorf("decode %s/%s: %w", table, id, err)
		}
		// Hash the re-marshalled value, not the stored bytes: jsonb
		// normalizes formatting, the Go encoding is deterministic.
		h, err := hashOf(v)
		if err != nil {
			return err
		}
		into[id] = v
		known[id] = h
	}
	p.known[table] = known
	return rows.Err()
}

func hashOf(v any) (digest, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return digest{}, err
	}
	return sha256.Sum256(b), nil
}

func (p *pgStore) save(d *data) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	next := map[string]map[string]digest{}
	if next["stores"], err = syncTable(ctx, tx, "stores", d.Stores, p.known["stores"]); err != nil {
		return err
	}
	if next["lists"], err = syncTable(ctx, tx, "lists", d.Lists, p.known["lists"]); err != nil {
		return err
	}
	if next["templates"], err = syncTable(ctx, tx, "templates", d.Templates, p.known["templates"]); err != nil {
		return err
	}
	if next["users"], err = syncTable(ctx, tx, "users", d.Users, p.known["users"]); err != nil {
		return err
	}
	if next["sessions"], err = syncTable(ctx, tx, "sessions", d.Sessions, p.known["sessions"]); err != nil {
		return err
	}

	for _, t := range d.Trips[p.tripsSaved:] {
		b, err := json.Marshal(t)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO trips(doc) VALUES ($1)`, b); err != nil {
			return err
		}
	}
	if d.NextID != p.nextID {
		if _, err := tx.ExecContext(ctx, `INSERT INTO meta(key, value) VALUES ('next_id', $1)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, d.NextID); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	// Only now is it safe to remember what the database holds.
	p.known = next
	p.tripsSaved = len(d.Trips)
	p.nextID = d.NextID
	return nil
}

// syncTable upserts rows whose document changed, deletes rows that no longer
// exist in memory, and returns the new row hashes.
func syncTable[T any](ctx context.Context, tx *sql.Tx, table string, cur map[string]*T, known map[string]digest) (map[string]digest, error) {
	next := make(map[string]digest, len(cur))
	upsert := fmt.Sprintf(`INSERT INTO %s(id, doc) VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET doc = EXCLUDED.doc`, table)
	for id, v := range cur {
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		h := sha256.Sum256(b)
		next[id] = h
		if old, ok := known[id]; ok && old == h {
			continue
		}
		if _, err := tx.ExecContext(ctx, upsert, id, b); err != nil {
			return nil, fmt.Errorf("write %s/%s: %w", table, id, err)
		}
	}
	del := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, table)
	for id := range known {
		if _, ok := cur[id]; !ok {
			if _, err := tx.ExecContext(ctx, del, id); err != nil {
				return nil, fmt.Errorf("delete %s/%s: %w", table, id, err)
			}
		}
	}
	return next, nil
}

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

type Employee struct {
	ID   int64
	Name string
	Team string
}

type Request struct {
	ID         int64
	EmployeeID int64
	Employee   string
	Team       string
	Kind       string
	StartDate  time.Time
	EndDate    time.Time
	Note       string
	Status     string
	DecidedBy  sql.NullString
	CreatedAt  time.Time
}

func (r Request) Days() int {
	return int(r.EndDate.Sub(r.StartDate).Hours()/24) + 1
}

var requestKinds = []string{"vacation", "sick", "parental", "bereavement", "other"}

type Store struct{ db *sql.DB }

// openStore accepts mysql:// or mariadb:// URLs so the platform can hand the app
// one DATABASE_URL assembled from the connected resource instead of five variables.
func openStore(ctx context.Context, rawURL string) (*Store, error) {
	dsn, err := dsnFromURL(rawURL)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func dsnFromURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	if u.Scheme != "mysql" && u.Scheme != "mariadb" {
		return "", fmt.Errorf("DATABASE_URL scheme must be mysql:// or mariadb://, got %q", u.Scheme)
	}
	host := u.Host
	if _, _, err := net.SplitHostPort(host); err != nil {
		host = net.JoinHostPort(host, "3306")
	}
	cfg := mysql.NewConfig()
	cfg.User = u.User.Username()
	cfg.Passwd, _ = u.User.Password()
	cfg.Net = "tcp"
	cfg.Addr = host
	cfg.DBName = strings.TrimPrefix(u.Path, "/")
	cfg.ParseTime = true
	cfg.Timeout = 10 * time.Second
	// tls=true verifies the server chain. A managed database signs with a
	// private CA, so the platform hands the bundle over in DATABASE_CA_BUNDLE and
	// we verify against that instead of the system roots.
	mode := u.Query().Get("tls")
	if mode == "true" || mode == "verify" {
		if pem := os.Getenv("DATABASE_CA_BUNDLE"); pem != "" {
			if err := registerCABundle(cfg.Addr, pem); err != nil {
				return "", err
			}
			mode = "landingzone"
		}
	}
	if mode != "" {
		cfg.TLSConfig = mode
	}
	return cfg.FormatDSN(), nil
}

func registerCABundle(addr, pem string) error {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM([]byte(pem)) {
		return errors.New("DATABASE_CA_BUNDLE contains no certificates")
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	return mysql.RegisterTLSConfig("landingzone", &tls.Config{RootCAs: pool, ServerName: host, MinVersion: tls.VersionTLS12})
}

func (s *Store) migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS employees (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(120) NOT NULL,
			team VARCHAR(80) NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS timeoff_requests (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			employee_id BIGINT NOT NULL,
			kind VARCHAR(32) NOT NULL,
			start_date DATE NOT NULL,
			end_date DATE NOT NULL,
			note VARCHAR(500) NOT NULL DEFAULT '',
			status VARCHAR(16) NOT NULL DEFAULT 'pending',
			decided_by VARCHAR(120) NULL,
			decided_at TIMESTAMP NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT fk_employee FOREIGN KEY (employee_id) REFERENCES employees(id)
		)`,
	}
	for _, q := range stmts {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return s.seed(ctx)
}

func (s *Store) seed(ctx context.Context) error {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM employees`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	seed := [][2]string{
		{"Ada Okafor", "Engineering"},
		{"Luis Ortega", "Engineering"},
		{"Priya Natarajan", "Customer Success"},
		{"Sam Whitaker", "Finance"},
		{"Mei Tanaka", "People Ops"},
	}
	for _, e := range seed {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO employees (name, team) VALUES (?, ?)`, e[0], e[1]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ListEmployees(ctx context.Context) ([]Employee, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, team FROM employees ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Employee
	for rows.Next() {
		var e Employee
		if err := rows.Scan(&e.ID, &e.Name, &e.Team); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) GetEmployee(ctx context.Context, id int64) (*Employee, error) {
	var e Employee
	err := s.db.QueryRowContext(ctx, `SELECT id, name, team FROM employees WHERE id = ?`, id).Scan(&e.ID, &e.Name, &e.Team)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &e, err
}

func (s *Store) ListRequests(ctx context.Context) ([]Request, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id, e.id, e.name, e.team, r.kind, r.start_date, r.end_date, r.note, r.status, r.decided_by, r.created_at
		FROM timeoff_requests r JOIN employees e ON e.id = r.employee_id
		ORDER BY r.created_at DESC, r.id DESC
		LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Request
	for rows.Next() {
		var r Request
		if err := rows.Scan(&r.ID, &r.EmployeeID, &r.Employee, &r.Team, &r.Kind, &r.StartDate, &r.EndDate, &r.Note, &r.Status, &r.DecidedBy, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) CreateRequest(ctx context.Context, employeeID int64, kind string, start, end time.Time, note, status, decidedBy string) (*Request, error) {
	var decided any
	if decidedBy != "" {
		decided = decidedBy
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO timeoff_requests (employee_id, kind, start_date, end_date, note, status, decided_by, decided_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, IF(? IS NULL, NULL, CURRENT_TIMESTAMP))`, employeeID, kind, start, end, note, status, decided, decided)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetRequest(ctx, id)
}

func (s *Store) GetRequest(ctx context.Context, id int64) (*Request, error) {
	var r Request
	err := s.db.QueryRowContext(ctx, `
		SELECT r.id, e.id, e.name, e.team, r.kind, r.start_date, r.end_date, r.note, r.status, r.decided_by, r.created_at
		FROM timeoff_requests r JOIN employees e ON e.id = r.employee_id
		WHERE r.id = ?`, id).Scan(&r.ID, &r.EmployeeID, &r.Employee, &r.Team, &r.Kind, &r.StartDate, &r.EndDate, &r.Note, &r.Status, &r.DecidedBy, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &r, err
}

// Decide returns nil when the request does not exist or was already decided.
func (s *Store) Decide(ctx context.Context, id int64, status, decidedBy string) (*Request, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE timeoff_requests SET status = ?, decided_by = ?, decided_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'pending'`, status, decidedBy, id)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, nil
	}
	return s.GetRequest(ctx, id)
}

func (s *Store) Healthy(ctx context.Context) error { return s.db.PingContext(ctx) }

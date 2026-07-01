// Hand-written (sqlc generate bị chặn bởi legacy reservation.sql). Chính sách web bán hàng.
package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type Policy struct {
	Slug      string             `json:"slug"`
	Title     string             `json:"title"`
	Summary   string             `json:"summary"`
	Body      string             `json:"body"`
	SortOrder int32              `json:"sort_order"`
	Active    bool               `json:"active"`
	UpdatedAt pgtype.Timestamptz `json:"updated_at"`
}

const policyCols = `slug, title, summary, body, sort_order, active, updated_at`

func scanPolicy(row interface{ Scan(...any) error }) (Policy, error) {
	var p Policy
	err := row.Scan(&p.Slug, &p.Title, &p.Summary, &p.Body, &p.SortOrder, &p.Active, &p.UpdatedAt)
	return p, err
}

const listPolicies = `SELECT ` + policyCols + ` FROM policies ORDER BY sort_order, title`

func (q *Queries) ListPolicies(ctx context.Context) ([]Policy, error) {
	rows, err := q.db.Query(ctx, listPolicies)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Policy{}
	for rows.Next() {
		p, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

const listActivePolicies = `SELECT ` + policyCols + ` FROM policies WHERE active = true ORDER BY sort_order, title`

func (q *Queries) ListActivePolicies(ctx context.Context) ([]Policy, error) {
	rows, err := q.db.Query(ctx, listActivePolicies)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Policy{}
	for rows.Next() {
		p, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

const getPolicy = `SELECT ` + policyCols + ` FROM policies WHERE slug = $1`

func (q *Queries) GetPolicy(ctx context.Context, slug string) (Policy, error) {
	return scanPolicy(q.db.QueryRow(ctx, getPolicy, slug))
}

const upsertPolicy = `
INSERT INTO policies (slug, title, summary, body, sort_order, active, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, now())
ON CONFLICT (slug) DO UPDATE SET title=$2, summary=$3, body=$4, sort_order=$5, active=$6, updated_at=now()
RETURNING ` + policyCols

type UpsertPolicyParams struct {
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	Body      string `json:"body"`
	SortOrder int32  `json:"sort_order"`
	Active    bool   `json:"active"`
}

func (q *Queries) UpsertPolicy(ctx context.Context, arg UpsertPolicyParams) (Policy, error) {
	return scanPolicy(q.db.QueryRow(ctx, upsertPolicy, arg.Slug, arg.Title, arg.Summary, arg.Body, arg.SortOrder, arg.Active))
}

const deletePolicy = `DELETE FROM policies WHERE slug = $1`

func (q *Queries) DeletePolicy(ctx context.Context, slug string) error {
	_, err := q.db.Exec(ctx, deletePolicy, slug)
	return err
}

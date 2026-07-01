-- name: AdminStats :one
SELECT
    (SELECT COALESCE(SUM(amount_vnd),0)::bigint FROM investments WHERE status = 'approved') AS total_capital_reconciled,
    (SELECT count(*) FROM shareholdings WHERE shares > 0)                                   AS shareholder_count,
    (SELECT shares_sold FROM offering WHERE status = 'open' ORDER BY created_at LIMIT 1)    AS shares_sold,
    (SELECT shares_for_sale FROM offering WHERE status = 'open' ORDER BY created_at LIMIT 1) AS shares_for_sale;

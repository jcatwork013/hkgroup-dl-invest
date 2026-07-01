-- name: CreateRevenueDistribution :one
INSERT INTO revenue_distributions
    (period, total_revenue, pool_rate, investor_share_rate, investor_pool, dividend_id, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListRevenueDistributions :many
SELECT * FROM revenue_distributions ORDER BY created_at DESC;

-- name: GetRevenueDistribution :one
SELECT * FROM revenue_distributions WHERE id = $1;

-- name: DeleteRevenueDistribution :exec
DELETE FROM revenue_distributions WHERE id = $1;

-- DeleteRevenueDistributionsByDividend: gỡ bản ghi phân bổ trỏ tới 1 đợt cổ tức (FK NO ACTION nên
-- phải xoá trước khi xoá dividends).
-- name: DeleteRevenueDistributionsByDividend :exec
DELETE FROM revenue_distributions WHERE dividend_id = $1;

-- name: SumRevenue :one
SELECT
    COALESCE(SUM(total_revenue), 0)::bigint AS total_revenue,
    COALESCE(SUM(investor_pool), 0)::bigint AS total_investor_pool
FROM revenue_distributions;

-- name: CreateWithdrawal :one
INSERT INTO withdrawals (user_id, amount, note) VALUES ($1, $2, $3)
RETURNING *;

-- name: ListWithdrawalsByUser :many
SELECT * FROM withdrawals WHERE user_id = $1 ORDER BY requested_at DESC;

-- name: ListWithdrawals :many
SELECT w.*, u.full_name, u.email FROM withdrawals w
JOIN users u ON u.id = w.user_id
ORDER BY w.requested_at DESC;

-- name: SetWithdrawalStatus :one
UPDATE withdrawals
SET status = $2, processed_by = $3, processed_at = now()
WHERE id = $1
RETURNING *;

-- name: SumWithdrawalsByUser :one
-- tiền đã rút hoặc đang chờ (không tính rejected) — để tính số dư khả dụng
SELECT COALESCE(SUM(amount), 0)::bigint AS total
FROM withdrawals WHERE user_id = $1 AND status <> 'rejected';

-- name: ListWalletBalances :many
-- Số dư ví hoa hồng của MỌI tài khoản có phát sinh hoa hồng (đầu tư + bán hàng),
-- để admin "lập lệnh rút dùm". Công thức KHỚP 100% với WalletService.Balance():
--   earned    = SUM(net_amount) commissions + sales_commissions  (status <> rejected)
--   withdrawn = SUM(amount) withdrawals                          (status <> rejected)
--   available = earned - withdrawn ; pending = phần status='pending'
SELECT
    u.id,
    u.email,
    u.full_name,
    (COALESCE(ic.earned, 0) + COALESCE(sc.earned, 0))::bigint                              AS earned,
    COALESCE(wd.withdrawn, 0)::bigint                                                      AS withdrawn,
    (COALESCE(ic.earned, 0) + COALESCE(sc.earned, 0) - COALESCE(wd.withdrawn, 0))::bigint  AS available,
    (COALESCE(ic.pending, 0) + COALESCE(sc.pending, 0))::bigint                            AS pending
FROM users u
JOIN (
    SELECT beneficiary_id FROM commissions       WHERE status <> 'rejected'
    UNION
    SELECT beneficiary_id FROM sales_commissions WHERE status <> 'rejected'
) b ON b.beneficiary_id = u.id
LEFT JOIN (
    SELECT beneficiary_id,
           SUM(net_amount)                                   AS earned,
           SUM(net_amount) FILTER (WHERE status = 'pending') AS pending
    FROM commissions WHERE status <> 'rejected' GROUP BY beneficiary_id
) ic ON ic.beneficiary_id = u.id
LEFT JOIN (
    SELECT beneficiary_id,
           SUM(net_amount)                                   AS earned,
           SUM(net_amount) FILTER (WHERE status = 'pending') AS pending
    FROM sales_commissions WHERE status <> 'rejected' GROUP BY beneficiary_id
) sc ON sc.beneficiary_id = u.id
LEFT JOIN (
    SELECT user_id, SUM(amount) AS withdrawn
    FROM withdrawals WHERE status <> 'rejected' GROUP BY user_id
) wd ON wd.user_id = u.id
ORDER BY available DESC, u.full_name;

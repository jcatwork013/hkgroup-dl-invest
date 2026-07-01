-- name: GetProfile :one
SELECT * FROM investor_profiles WHERE user_id = $1;

-- name: UpsertProfile :one
INSERT INTO investor_profiles (
    user_id, date_of_birth, gender, nationality, cccd_number, cccd_issue_date,
    cccd_issue_place, permanent_address, contact_address, occupation, tax_code,
    bank_name, bank_account_number, bank_account_name, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14, now())
ON CONFLICT (user_id) DO UPDATE SET
    date_of_birth = EXCLUDED.date_of_birth,
    gender = EXCLUDED.gender,
    nationality = EXCLUDED.nationality,
    cccd_number = EXCLUDED.cccd_number,
    cccd_issue_date = EXCLUDED.cccd_issue_date,
    cccd_issue_place = EXCLUDED.cccd_issue_place,
    permanent_address = EXCLUDED.permanent_address,
    contact_address = EXCLUDED.contact_address,
    occupation = EXCLUDED.occupation,
    tax_code = EXCLUDED.tax_code,
    bank_name = EXCLUDED.bank_name,
    bank_account_number = EXCLUDED.bank_account_number,
    bank_account_name = EXCLUDED.bank_account_name,
    updated_at = now()
RETURNING *;

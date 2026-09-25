-- name: ListVerifiedLaws :many
-- Только проверенные вручную. Законы без даты вступления в силу (внесены/приняты) попадают всегда.
SELECT
    id, title, what_changed, who_affected, actions, audience_tags, region_code, status,
    introduced_at, passed_at, signed_at, effective_at,
    official_url, bill_url, act_number, verified_at
FROM law_changes
WHERE verified
  AND (effective_at IS NULL OR (effective_at >= @from_date::date AND effective_at <= @to_date::date))
ORDER BY effective_at NULLS LAST, id;

-- name: LawSeen :one
SELECT EXISTS (SELECT 1 FROM law_changes WHERE eo_number = $1);

-- name: InsertLawDraft :one
INSERT INTO law_changes (
    eo_number, title, what_changed, who_affected, actions, audience_tags, region_code, status,
    passed_at, signed_at, effective_at, official_url, source_url, act_number, quotes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
)
ON CONFLICT (eo_number) DO NOTHING
RETURNING id;

-- name: InsertRejectedLaw :exec
-- Отклонённые сохраняются, чтобы не обрабатывать один и тот же закон повторно.
INSERT INTO law_changes (eo_number, title, what_changed, who_affected, status, signed_at, act_number, rejected, reject_reason)
VALUES ($1, $2, '', '', 'signed', $3, $4, true, $5)
ON CONFLICT (eo_number) DO NOTHING;

-- name: ListLawDrafts :many
SELECT id, eo_number, title, what_changed, who_affected, actions, audience_tags, region_code, status,
       passed_at, signed_at, effective_at, official_url, source_url, act_number, quotes
FROM law_changes
WHERE NOT verified AND NOT rejected
ORDER BY effective_at NULLS LAST, id;

-- name: UpdateLawDraft :exec
UPDATE law_changes
SET title = $2, what_changed = $3, who_affected = $4, actions = $5, audience_tags = $6,
    region_code = $7, status = $8, effective_at = $9
WHERE id = $1 AND NOT verified;

-- name: VerifyLaw :exec
UPDATE law_changes SET verified = true, verified_at = now() WHERE id = $1 AND NOT rejected;

-- name: RejectLaw :exec
UPDATE law_changes SET rejected = true, reject_reason = $2 WHERE id = $1 AND NOT verified;

-- name: CountLaws :one
SELECT
    count(*) FILTER (WHERE verified)                       AS verified,
    count(*) FILTER (WHERE NOT verified AND NOT rejected)  AS drafts,
    count(*) FILTER (WHERE rejected)                       AS rejected
FROM law_changes;

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

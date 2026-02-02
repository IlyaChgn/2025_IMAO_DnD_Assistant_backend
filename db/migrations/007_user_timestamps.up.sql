ALTER TABLE public."user"
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ NULL;

-- Backfill: ensure existing rows have created_at/updated_at set.
-- With DEFAULT now() on NOT NULL, Postgres fills existing rows automatically
-- during ADD COLUMN. This UPDATE is a safety net for edge cases.
UPDATE public."user"
SET created_at  = COALESCE(created_at, now()),
    updated_at  = COALESCE(updated_at, now())
WHERE created_at IS NULL
   OR updated_at IS NULL;

ALTER TABLE public."user"
    DROP COLUMN IF EXISTS last_login_at,
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS created_at;

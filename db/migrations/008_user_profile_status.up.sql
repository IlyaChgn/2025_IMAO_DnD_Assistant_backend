-- Rename existing columns.
ALTER TABLE public."user" RENAME COLUMN name TO display_name;
ALTER TABLE public."user" RENAME COLUMN avatar TO avatar_url;

-- Add new profile / status columns.
ALTER TABLE public."user"
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'user',
    ADD COLUMN IF NOT EXISTS email TEXT NULL,
    ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT false;

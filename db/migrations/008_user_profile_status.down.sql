ALTER TABLE public."user"
    DROP COLUMN IF EXISTS email_verified,
    DROP COLUMN IF EXISTS email,
    DROP COLUMN IF EXISTS role,
    DROP COLUMN IF EXISTS status;

ALTER TABLE public."user" RENAME COLUMN display_name TO name;
ALTER TABLE public."user" RENAME COLUMN avatar_url TO avatar;

-- PR5: Drop vkid column from user table.
-- All provider-specific IDs are now in user_identity.
ALTER TABLE public."user" DROP COLUMN IF EXISTS vkid;

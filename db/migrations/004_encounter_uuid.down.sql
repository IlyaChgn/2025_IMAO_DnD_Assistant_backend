DROP INDEX IF EXISTS encounter_uuid_idx;

ALTER TABLE public.encounter_store
DROP COLUMN IF EXISTS uuid;

DROP EXTENSION IF EXISTS "uuid-ossp";

ALTER TABLE public.encounter_store
ADD COLUMN IF NOT EXISTS uuid UUID
    UNIQUE;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

UPDATE public.encounter_store
SET uuid = uuid_generate_v4()
WHERE uuid IS NULL;

ALTER TABLE public.encounter_store
    ALTER COLUMN uuid SET NOT NULL;

CREATE UNIQUE INDEX encounter_uuid_idx
ON public.encounter_store (uuid);


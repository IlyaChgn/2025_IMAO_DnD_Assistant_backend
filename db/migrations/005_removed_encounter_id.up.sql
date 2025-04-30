ALTER TABLE public.encounter_store
DROP CONSTRAINT IF EXISTS encounter_store_uuid_key;

DROP INDEX IF EXISTS encounter_uuid_idx;

ALTER TABLE public.encounter_store
DROP CONSTRAINT encounter_store_pkey;

ALTER TABLE public.encounter_store
DROP COLUMN id;

ALTER TABLE public.encounter_store
ADD PRIMARY KEY (uuid);

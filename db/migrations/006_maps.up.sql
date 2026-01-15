CREATE TABLE IF NOT EXISTS public.maps
(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL
        REFERENCES public.user(id),
    name VARCHAR(255) NOT NULL
        CHECK(name <> '')
        CONSTRAINT max_len_map_name CHECK(LENGTH(name) <= 255),
    data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX maps_user_id_idx ON public.maps (user_id);
CREATE INDEX maps_user_id_updated_at_idx ON public.maps (user_id, updated_at DESC);

-- Trigger to auto-update updated_at on row update
CREATE OR REPLACE FUNCTION update_maps_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER maps_updated_at_trigger
    BEFORE UPDATE ON public.maps
    FOR EACH ROW
    EXECUTE FUNCTION update_maps_updated_at();

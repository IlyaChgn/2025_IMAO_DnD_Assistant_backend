DROP TRIGGER IF EXISTS maps_updated_at_trigger ON public.maps;
DROP FUNCTION IF EXISTS update_maps_updated_at();
DROP INDEX IF EXISTS maps_user_id_updated_at_idx;
DROP INDEX IF EXISTS maps_user_id_idx;
DROP TABLE IF EXISTS public.maps;

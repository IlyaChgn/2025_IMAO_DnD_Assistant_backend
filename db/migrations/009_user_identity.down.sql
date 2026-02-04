-- Restore vkid NOT NULL (backfill from identity if needed)
UPDATE public."user" u
SET vkid = ui.provider_user_id
FROM public.user_identity ui
WHERE ui.user_id = u.id AND ui.provider = 'vk' AND u.vkid IS NULL;

ALTER TABLE public."user" ALTER COLUMN vkid SET NOT NULL;

DROP INDEX IF EXISTS idx_identity_user_id;
DROP TABLE IF EXISTS public.user_identity;

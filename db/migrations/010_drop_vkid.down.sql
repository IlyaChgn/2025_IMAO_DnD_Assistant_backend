-- Restore vkid column and backfill from user_identity.
ALTER TABLE public."user" ADD COLUMN IF NOT EXISTS vkid TEXT;

UPDATE public."user" u
SET vkid = ui.provider_user_id
FROM public.user_identity ui
WHERE ui.user_id = u.id AND ui.provider = 'vk';

ALTER TABLE public."user" ALTER COLUMN vkid SET NOT NULL;
ALTER TABLE public."user" ADD CONSTRAINT user_vkid_unique UNIQUE (vkid);
ALTER TABLE public."user" ADD CONSTRAINT user_vkid_nonempty CHECK (vkid <> '');

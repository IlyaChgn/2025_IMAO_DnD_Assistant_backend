package repository

const (
	FindIdentityByProviderQuery = `
		SELECT id, user_id, provider, provider_user_id, email
		FROM public.user_identity
		WHERE provider = $1 AND provider_user_id = $2;
	`

	CreateIdentityQuery = `
		INSERT INTO public.user_identity (user_id, provider, provider_user_id, email)
		VALUES ($1, $2, $3, $4);
	`

	UpdateIdentityLastUsedQuery = `
		UPDATE public.user_identity
		SET last_used_at = $1
		WHERE id = $2;
	`

	ListIdentitiesByUserIDQuery = `
		SELECT id, user_id, provider, provider_user_id, email, created_at
		FROM public.user_identity
		WHERE user_id = $1
		ORDER BY created_at;
	`

	DeleteIdentityByUserAndProviderQuery = `
		DELETE FROM public.user_identity
		WHERE user_id = $1 AND provider = $2;
	`

	FindUserByIdentityQuery = `
		SELECT u.id, u.display_name, u.avatar_url, u.status
		FROM public."user" u
		JOIN public.user_identity ui ON ui.user_id = u.id
		WHERE ui.provider = $1 AND ui.provider_user_id = $2;
	`
)

package repository

const (
	CheckUserQuery = `
		SELECT id, vkid, display_name, avatar_url, status
		FROM public."user"
		WHERE vkid = $1;
	`

	CreateUserQuery = `
		INSERT INTO public."user" (vkid, display_name, avatar_url)
		VALUES ($1, $2, $3)
		RETURNING id, vkid, display_name, avatar_url, status;
	`

	UpdateUserQuery = `
		UPDATE public."user"
		SET display_name = $2, avatar_url = $3, updated_at = now()
		WHERE vkid = $1
		RETURNING id, vkid, display_name, avatar_url, status;
	`

	UpdateLastLoginAtQuery = `
		UPDATE public."user"
		SET last_login_at = $1, updated_at = now()
		WHERE id = $2;
	`
)

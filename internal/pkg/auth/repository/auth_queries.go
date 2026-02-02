package repository

const (
	GetUserByIDQuery = `
		SELECT id, display_name, avatar_url, status
		FROM public."user"
		WHERE id = $1;
	`

	CreateUserQuery = `
		INSERT INTO public."user" (display_name, avatar_url)
		VALUES ($1, $2)
		RETURNING id, display_name, avatar_url, status;
	`

	UpdateUserQuery = `
		UPDATE public."user"
		SET display_name = $2, avatar_url = $3, updated_at = now()
		WHERE id = $1
		RETURNING id, display_name, avatar_url, status;
	`

	UpdateLastLoginAtQuery = `
		UPDATE public."user"
		SET last_login_at = $1, updated_at = now()
		WHERE id = $2;
	`
)

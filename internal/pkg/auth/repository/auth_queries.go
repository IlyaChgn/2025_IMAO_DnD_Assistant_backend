package repository

const (
	CheckUserQuery = `
		SELECT id, vkid, name, avatar 
		FROM public.user
		WHERE vkid = $1;
	`

	CreateUserQuery = `
		INSERT INTO public.user (vkid, name, avatar)
		VALUES ($1, $2, $3)
		RETURNING id, vkid, name, avatar;
	`

	UpdateUserQuery = `
		UPDATE public.user
		SET name = $2, avatar = $3
		WHERE vkid = $1
		RETURNING id, vkid, name, avatar;
	`
)

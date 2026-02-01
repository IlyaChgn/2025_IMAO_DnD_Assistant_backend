package repository

const (
	CheckMapPermissionQuery = `
		SELECT EXISTS(
			SELECT 1
			FROM public.maps
			WHERE id = $1 AND user_id = $2
		);
	`

	CreateMapQuery = `
		INSERT INTO public.maps (user_id, name, data)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, name, data, created_at, updated_at;
	`

	GetMapByIDQuery = `
		SELECT id, user_id, name, data, created_at, updated_at
		FROM public.maps
		WHERE id = $1 AND user_id = $2;
	`

	UpdateMapQuery = `
		UPDATE public.maps
		SET name = $3, data = $4
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, name, data, created_at, updated_at;
	`

	DeleteMapQuery = `
		DELETE FROM public.maps
		WHERE id = $1 AND user_id = $2;
	`

	ListMapsQuery = `
		SELECT id, user_id, name, created_at, updated_at
		FROM public.maps
		WHERE user_id = $1
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3;
	`

	CountMapsQuery = `
		SELECT COUNT(*)
		FROM public.maps
		WHERE user_id = $1;
	`
)

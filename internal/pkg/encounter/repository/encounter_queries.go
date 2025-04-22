package repository

const (
	CheckPermissionQuery = `
		SELECT EXISTS(
		    SELECT 1
			FROM public.encounter_store
			WHERE id = $1 AND user_id = $2 AND NOT(is_deleted)
		);
	`

	GetEncounterByIDQuery = `
		SELECT id, user_id, name, data
		FROM public.encounter_store
		WHERE id = $1 AND NOT(is_deleted);
	`

	GetEncountersListQuery = `
		SELECT id, user_id, name
		FROM public.encounter_store
		WHERE user_id = $1 AND NOT(is_deleted)
		LIMIT $2 OFFSET $3;
	`

	GetEncountersListWithSearchQuery = `
		SELECT id, user_id, name
		FROM public.encounter_store
		WHERE user_id = $1
			AND NOT is_deleted
			AND (
				to_tsvector('russian', name) || to_tsvector('english', name)
			) @@ (
				to_tsquery('russian', $2) || to_tsquery('english', $2)
			)
		LIMIT $3 OFFSET $4;
	`

	SaveEncounterQuery = `
		INSERT INTO public.encounter_store (user_id, name, data) 
		VALUES ($1, $2, $3)
		RETURNING id, user_id, name, data;
	`

	UpdateEncounterQuery = `
		UPDATE public.encounter_store
		SET data = $2
		WHERE id = $1;
	`

	DeleteEncounterQuery = `
		UPDATE public.encounter_store
		SET is_deleted = TRUE
		WHERE id = $1;
	`
)

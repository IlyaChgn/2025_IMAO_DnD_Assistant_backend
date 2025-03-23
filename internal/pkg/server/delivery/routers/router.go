package router

import (
	bestiaryinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	bestiarydel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/delivery"
	characterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character"
	characterdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character/delivery"
	creatureinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/creature"
	creaturedel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/creature/delivery"
	descriptioninterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description"
	descriptiondel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description/delivery"
	encounterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter"
	encounterdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter/delivery"
	myrecovery "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/middleware/recover"
	"github.com/gorilla/mux"
)

func NewRouter(creatureInterface creatureinterfaces.CreatureUsecases,
	bestiaryInterface bestiaryinterfaces.BestiaryUsecases,
	descriptionInterface descriptioninterfaces.DescriptionUsecases,
	characterInterface characterinterfaces.CharacterUsecases,
	encounterInterface encounterinterfaces.EncounterUsecases) *mux.Router {

	creatureHandler := creaturedel.NewCreatureHandler(creatureInterface)
	bestiaryHandler := bestiarydel.NewBestiaryHandler(bestiaryInterface)
	descriptionHandler := descriptiondel.NewDescriptionHandler(descriptionInterface)
	characterHandler := characterdel.NewCharacterHandler(characterInterface)
	encounterHandler := encounterdel.NewEncounterHandler(encounterInterface)

	router := mux.NewRouter()

	router.Use(myrecovery.RecoveryMiddleware)

	rootRouter := router.PathPrefix("/api").Subrouter()

	ServeBestiaryRouter(rootRouter, creatureHandler, bestiaryHandler)
	ServeBattleRouter(rootRouter, descriptionHandler)
	ServeCharacterRouter(rootRouter, characterHandler)
	ServeEncounteRouter(rootRouter, encounterHandler)

	return router
}

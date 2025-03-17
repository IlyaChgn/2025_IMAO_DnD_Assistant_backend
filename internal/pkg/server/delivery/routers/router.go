package router

import (
	bestiaryinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	bestiarydel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/delivery"
	creatureinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/creature"
	creaturedel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/creature/delivery"
	descriptioninterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description"
	descriptiondel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description/delivery"
	myrecovery "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/middleware/recover"
	"github.com/gorilla/mux"
)

func NewRouter(creatureInterface creatureinterfaces.CreatureUsecases,
	bestiaryInterface bestiaryinterfaces.BestiaryUsecases,
	descriptionInterface descriptioninterfaces.DescriptionUsecases) *mux.Router {

	creatureHandler := creaturedel.NewCreatureHandler(creatureInterface)
	bestiaryHandler := bestiarydel.NewBestiaryHandler(bestiaryInterface)
	descriptionHandler := descriptiondel.NewDescriptionHandler(descriptionInterface)

	router := mux.NewRouter()

	router.Use(myrecovery.RecoveryMiddleware)

	rootRouter := router.PathPrefix("/api").Subrouter()

	ServeBestiaryRouter(rootRouter, creatureHandler, bestiaryHandler)
	ServeBattleRouter(rootRouter, descriptionHandler)

	return router
}

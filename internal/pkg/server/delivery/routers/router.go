package router

import (
	creatureinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/creature"
	creaturedel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/creature/delivery"
	myrecovery "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/middleware/recover"
	"github.com/gorilla/mux"
)

func NewRouter(creatureinterfaces creatureinterfaces.CreatureUsecases) *mux.Router {
	creatureHandler := creaturedel.NewCreatureHandler(creatureinterfaces)

	router := mux.NewRouter()

	router.Use(myrecovery.RecoveryMiddleware)

	rootRouter := router.PathPrefix("/api").Subrouter()
	ServeCreatureRouter(rootRouter, creatureHandler)

	return router
}

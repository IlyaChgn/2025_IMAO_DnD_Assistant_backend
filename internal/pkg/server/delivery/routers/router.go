package router

import (
	authinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth"
	authdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth/delivery"
	bestiaryinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	bestiarydel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/delivery"
	characterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character"
	characterdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character/delivery"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/config"
	descriptioninterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description"
	descriptiondel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description/delivery"
	encounterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter"
	encounterdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter/delivery"
	myauth "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/middleware/auth"
	myrecovery "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/middleware/recover"
	"github.com/gorilla/mux"
)

func NewRouter(cfg *config.Config,
	bestiaryInterface bestiaryinterfaces.BestiaryUsecases,
	descriptionInterface descriptioninterfaces.DescriptionUsecases,
	characterInterface characterinterfaces.CharacterUsecases,
	encounterInterface encounterinterfaces.EncounterUsecases,
	authInterface authinterface.AuthUsecases) *mux.Router {

	bestiaryHandler := bestiarydel.NewBestiaryHandler(bestiaryInterface)
	descriptionHandler := descriptiondel.NewDescriptionHandler(descriptionInterface)
	characterHandler := characterdel.NewCharacterHandler(characterInterface)
	encounterHandler := encounterdel.NewEncounterHandler(encounterInterface, authInterface)
	authHandler := authdel.NewAuthHandler(authInterface, &cfg.VKApi)

	loginRequiredMiddleware := myauth.LoginRequiredMiddleware(authInterface)

	router := mux.NewRouter()

	router.Use(myrecovery.RecoveryMiddleware)

	rootRouter := router.PathPrefix("/api").Subrouter()

	ServeBestiaryRouter(rootRouter, bestiaryHandler)
	ServeBattleRouter(rootRouter, descriptionHandler)
	ServeCharacterRouter(rootRouter, characterHandler)
	ServeEncounteRouter(rootRouter, encounterHandler, loginRequiredMiddleware)
	ServeAuthRouter(rootRouter, authHandler, loginRequiredMiddleware)

	return router
}

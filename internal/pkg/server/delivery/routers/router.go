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
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	myauth "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/middleware/auth"
	mymetrics "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/middleware/metrics"
	myrecovery "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/middleware/recover"
	tableinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/table"
	tabledel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/table/delivery"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
)

func NewRouter(cfg *config.Config,
	bestiaryInterface bestiaryinterfaces.BestiaryUsecases,
	descriptionInterface descriptioninterfaces.DescriptionUsecases,
	characterInterface characterinterfaces.CharacterUsecases,
	encounterInterface encounterinterfaces.EncounterUsecases,
	authInterface authinterface.AuthUsecases,
	tableInterface tableinterfaces.TableUsecases,
	llmInterface bestiaryinterfaces.GenerationUsecases) *mux.Router {

	bestiaryHandler := bestiarydel.NewBestiaryHandler(bestiaryInterface, authInterface)
	descriptionHandler := descriptiondel.NewDescriptionHandler(descriptionInterface)
	characterHandler := characterdel.NewCharacterHandler(characterInterface, authInterface)
	encounterHandler := encounterdel.NewEncounterHandler(encounterInterface, authInterface)
	authHandler := authdel.NewAuthHandler(authInterface, &cfg.VKApi)
	tableHandler := tabledel.NewTableHandler(tableInterface, authInterface)
	llmHandler := bestiarydel.NewLLMHandler(llmInterface, authInterface)

	loginRequiredMiddleware := myauth.LoginRequiredMiddleware(authInterface)

	router := mux.NewRouter()

	router.Use(myrecovery.RecoveryMiddleware)

	m, err := metrics.NewHTTPMetrics()
	if err != nil {
		log.Fatal("Something went wrong initializing prometheus app metrics, ", err)
	}

	metricsMiddleware := mymetrics.CreateMetricsMiddleware(m)

	router.Use(metricsMiddleware)

	router.PathPrefix("/metrics").Handler(promhttp.Handler())

	rootRouter := router.PathPrefix("/api").Subrouter()

	ServeBestiaryRouter(rootRouter, bestiaryHandler, loginRequiredMiddleware)
	ServeBattleRouter(rootRouter, descriptionHandler)
	ServeCharacterRouter(rootRouter, characterHandler, loginRequiredMiddleware)
	ServeEncounteRouter(rootRouter, encounterHandler, loginRequiredMiddleware)
	ServeAuthRouter(rootRouter, authHandler, loginRequiredMiddleware)
	ServeTableRouter(rootRouter, tableHandler, loginRequiredMiddleware)
	ServeLLMRouter(rootRouter, llmHandler, loginRequiredMiddleware)

	return router
}

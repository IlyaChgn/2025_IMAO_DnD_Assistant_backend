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
	mylogger "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	mapsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maps"
	mapsdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maps/delivery"
	maptilesinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maptiles"
	maptilesdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maptiles/delivery"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	myauth "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/middleware/auth"
	mylog "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/middleware/log"
	mymetrics "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/middleware/metrics"
	myrecovery "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/middleware/recover"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/middleware/reqdata"
	tableinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/table"
	tabledel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/table/delivery"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(cfg *config.Config,
	logger mylogger.Logger,
	m metrics.HTTPMetrics,
	bestiaryInterface bestiaryinterfaces.BestiaryUsecases,
	descriptionInterface descriptioninterfaces.DescriptionUsecases,
	characterInterface characterinterfaces.CharacterUsecases,
	encounterInterface encounterinterfaces.EncounterUsecases,
	authInterface authinterface.AuthUsecases,
	tableInterface tableinterfaces.TableUsecases,
	llmInterface bestiaryinterfaces.GenerationUsecases,
	maptilesInterface maptilesinterfaces.MapTilesUsecases,
	mapsInterface mapsinterfaces.MapsUsecases) *mux.Router {

	bestiaryHandler := bestiarydel.NewBestiaryHandler(bestiaryInterface, cfg.CtxUserKey)
	descriptionHandler := descriptiondel.NewDescriptionHandler(descriptionInterface)
	characterHandler := characterdel.NewCharacterHandler(characterInterface, cfg.CtxUserKey)
	encounterHandler := encounterdel.NewEncounterHandler(encounterInterface, cfg.CtxUserKey)
	authHandler := authdel.NewAuthHandler(authInterface, cfg.Session.Duration, cfg.IsProd)
	tableHandler := tabledel.NewTableHandler(tableInterface, cfg.CtxUserKey)
	llmHandler := bestiarydel.NewLLMHandler(llmInterface)
	mapTilesHandler := maptilesdel.NewMapTilesHandler(maptilesInterface, cfg.CtxUserKey)
	mapsHandler := mapsdel.NewMapsHandler(mapsInterface, cfg.CtxUserKey)

	loginRequiredMiddleware := myauth.LoginRequiredMiddleware(authInterface, cfg.CtxUserKey)

	router := mux.NewRouter()

	logMiddleware := mylog.CreateLogMiddleware(logger)
	metricsMiddleware := mymetrics.CreateMetricsMiddleware(m)

	router.Use(logMiddleware)
	router.Use(reqdata.RequestDataMiddleware)
	router.Use(myrecovery.RecoveryMiddleware)
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
	ServeMapTilesRouter(rootRouter, mapTilesHandler, loginRequiredMiddleware)
	ServeMapsRouter(rootRouter, mapsHandler, loginRequiredMiddleware)

	return router
}

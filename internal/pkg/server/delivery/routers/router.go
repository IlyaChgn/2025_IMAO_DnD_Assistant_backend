package router

import (
	authinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth"
	actionsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/actions"
	actionsdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/actions/delivery"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/combatai"
	combataideliv "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/combatai/delivery"
	authdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth/delivery"
	backgroundsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/backgrounds"
	backgroundsdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/backgrounds/delivery"
	bestiaryinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	bestiarydel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/delivery"
	classesinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/classes"
	classesdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/classes/delivery"
	characterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character"
	characterdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character/delivery"
	conditionsdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/conditions/delivery"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/config"
	descriptioninterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description"
	descriptiondel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description/delivery"
	encounterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter"
	encounterdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter/delivery"
	featsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/feats"
	featsdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/feats/delivery"
	featuresinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/features"
	featuresdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/features/delivery"
	itemsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/items"
	itemsdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/items/delivery"
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
	racesinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/races"
	racesdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/races/delivery"
	spellsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/spells"
	spellsdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/spells/delivery"
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
	characterBaseInterface characterinterfaces.CharacterBaseUsecases,
	encounterInterface encounterinterfaces.EncounterUsecases,
	authInterface authinterface.AuthUsecases,
	tableInterface tableinterfaces.TableUsecases,
	llmInterface bestiaryinterfaces.GenerationUsecases,
	maptilesInterface maptilesinterfaces.MapTilesUsecases,
	mapsInterface mapsinterfaces.MapsUsecases,
	spellsInterface spellsinterfaces.SpellsUsecases,
	featuresInterface featuresinterfaces.FeaturesUsecases,
	itemsInterface itemsinterfaces.ItemUsecases,
	inventoryInterface itemsinterfaces.InventoryUsecases,
	broadcaster itemsinterfaces.SessionBroadcaster,
	actionsInterface actionsinterfaces.ActionsUsecases,
	classesInterface classesinterfaces.ClassesUsecases,
	racesInterface racesinterfaces.RacesUsecases,
	backgroundsInterface backgroundsinterfaces.BackgroundsUsecases,
	featsInterface featsinterfaces.FeatsUsecases,
	combatAIInterface combatai.CombatAIUsecases) *mux.Router {

	bestiaryHandler := bestiarydel.NewBestiaryHandler(bestiaryInterface, cfg.CtxUserKey)
	descriptionHandler := descriptiondel.NewDescriptionHandler(descriptionInterface)
	characterHandler := characterdel.NewCharacterHandler(characterInterface, cfg.CtxUserKey)
	characterBaseHandler := characterdel.NewCharacterBaseHandler(characterBaseInterface, cfg.CtxUserKey)
	encounterHandler := encounterdel.NewEncounterHandler(encounterInterface, cfg.CtxUserKey)
	authHandler := authdel.NewAuthHandler(authInterface, cfg.Session.Duration, cfg.IsProd, cfg.CtxUserKey)
	tableHandler := tabledel.NewTableHandler(tableInterface, cfg.CtxUserKey)
	llmHandler := bestiarydel.NewLLMHandler(llmInterface)
	mapTilesHandler := maptilesdel.NewMapTilesHandler(maptilesInterface, cfg.CtxUserKey)
	mapsHandler := mapsdel.NewMapsHandler(mapsInterface, cfg.CtxUserKey)
	spellsHandler := spellsdel.NewSpellsHandler(spellsInterface)
	featuresHandler := featuresdel.NewFeaturesHandler(featuresInterface)
	conditionsHandler := conditionsdel.NewConditionsHandler()
	itemsHandler := itemsdel.NewItemsHandler(itemsInterface, cfg.CtxUserKey)
	inventoryHandler := itemsdel.NewInventoryHandler(inventoryInterface, cfg.CtxUserKey, broadcaster)
	actionsHandler := actionsdel.NewActionsHandler(actionsInterface, cfg.CtxUserKey)
	classesHandler := classesdel.NewClassesHandler(classesInterface)
	racesHandler := racesdel.NewRacesHandler(racesInterface)
	backgroundsHandler := backgroundsdel.NewBackgroundsHandler(backgroundsInterface)
	featsHandler := featsdel.NewFeatsHandler(featsInterface)
	combatAIHandler := combataideliv.NewCombatAIHandler(combatAIInterface, cfg.CtxUserKey)

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
	ServeCharacterBaseRouter(rootRouter, characterBaseHandler, loginRequiredMiddleware)
	ServeEncounteRouter(rootRouter, encounterHandler, loginRequiredMiddleware)
	ServeAuthRouter(rootRouter, authHandler, loginRequiredMiddleware)
	ServeTableRouter(rootRouter, tableHandler, loginRequiredMiddleware)
	ServeLLMRouter(rootRouter, llmHandler, loginRequiredMiddleware)
	ServeMapTilesRouter(rootRouter, mapTilesHandler, loginRequiredMiddleware)
	ServeMapsRouter(rootRouter, mapsHandler, loginRequiredMiddleware)
	ServeSpellsRouter(rootRouter, spellsHandler)
	ServeFeaturesRouter(rootRouter, featuresHandler)
	ServeConditionsRouter(rootRouter, conditionsHandler)
	ServeItemsRouter(rootRouter, itemsHandler, loginRequiredMiddleware)
	ServeInventoryRouter(rootRouter, inventoryHandler, loginRequiredMiddleware)
	ServeActionsRouter(rootRouter, actionsHandler, loginRequiredMiddleware)
	ServeClassesRouter(rootRouter, classesHandler)
	ServeRacesRouter(rootRouter, racesHandler)
	ServeBackgroundsRouter(rootRouter, backgroundsHandler)
	ServeFeatsRouter(rootRouter, featsHandler)
	ServeCombatAIRouter(rootRouter, combatAIHandler, loginRequiredMiddleware)

	return router
}

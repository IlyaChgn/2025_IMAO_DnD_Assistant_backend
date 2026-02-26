package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	authinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth"
	authext "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth/external"
	mylogger "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbinit"

	tablerepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/table/repository"
	tableuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/table/usecases"

	authrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth/repository"
	authuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth/usecases"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/config"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/grpcconnection"
	myrouter "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/routers"
	"github.com/gorilla/handlers"

	bestiaryinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	bestiarydlv "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/delivery"
	bestiaryproto "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/delivery/protobuf"
	bestiaryext "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/external"
	bestiaryrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/repository"
	bestiaryuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/usecases"
	characterrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character/repository"
	characteruc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character/usecases"
	descriptioninterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description"
	descriptiondlv "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description/delivery"
	descriptionproto "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description/delivery/protobuf"
	descriptionuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description/usecases"
	encounterrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter/repository"
	encounteruc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter/usecases"
	mapsrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maps/repository"
	mapsuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maps/usecases"
	maptilerepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maptiles/repository"
	maptileuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maptiles/usecases"
	backgroundsrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/backgrounds/repository"
	backgroundsseed "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/backgrounds/seed"
	backgroundsuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/backgrounds/usecases"
	classesrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/classes/repository"
	classesseed "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/classes/seed"
	classesuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/classes/usecases"
	featsrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/feats/repository"
	featsseed "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/feats/seed"
	featsuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/feats/usecases"
	featuresrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/features/repository"
	featuresseed "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/features/seed"
	featuresuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/features/usecases"
	racesrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/races/repository"
	racesseed "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/races/seed"
	racesuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/races/usecases"
	spellsrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/spells/repository"
	spellsseed "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/spells/seed"
	spellsuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/spells/usecases"

	actionsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/actions"
	auditlogrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/actions/repository"
	actionsuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/actions/usecases"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/combatai"
	combataiuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/combatai/usecases"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/dungeongen"
	dungeongenrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/dungeongen/repository"
	dungeongenuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/dungeongen/usecases"
	itemsrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/items/repository"
	itemsseed "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/items/seed"
	itemsuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/items/usecases"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/socks5proxy"
)

type Server struct {
	server *http.Server
}

type serverConfig struct {
	Address string
	Timeout time.Duration
	Handler http.Handler
}

func createServerConfig(addr string, timeout int, handler *http.Handler) serverConfig {
	return serverConfig{
		Address: addr,
		Timeout: time.Second * time.Duration(timeout),
		Handler: *handler,
	}
}

func createServer(config serverConfig) *http.Server {
	return &http.Server{
		Addr:         config.Address,
		ReadTimeout:  config.Timeout,
		WriteTimeout: config.Timeout,
		Handler:      config.Handler,
	}
}

func (srv *Server) Run() error {
	cfgPath := os.Getenv("CONFIG_PATH")

	serverMode := os.Getenv("SERVER_MODE")
	isProduction := serverMode == "production"
	isTestMode := serverMode == "test"

	cfg := config.ReadConfig(cfgPath)
	if cfg == nil {
		log.Fatal("The config wasn`t opened")
	}

	cfg.IsProd = isProduction

	logger, err := mylogger.New(cfg.Logger.OutputPath, cfg.Logger.ErrPath)
	if err != nil {
		log.Fatalf("Failed to initialize log: %v", err)
	}
	defer logger.Sync()

	m, err := metrics.NewHTTPMetrics()
	if err != nil {
		log.Fatal("Something went wrong initializing prometheus app metrics, ", err)
	}

	var descriptionGateway descriptioninterface.DescriptionGateway
	var actionProcessorGateway bestiaryinterface.ActionProcessorGateway
	var geminiAPI bestiaryinterface.GeminiAPI

	if isTestMode {
		descriptionGateway = stubDescriptionGateway{}
		actionProcessorGateway = stubActionProcessorGateway{}
		geminiAPI = stubGeminiAPI{}
	} else {
		descriptionAddr := fmt.Sprintf("%s:%s", cfg.Services.Description.Host, cfg.Services.Description.Port)

		grpcConnDescription, grpcErr := grpc.NewClient(
			descriptionAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if grpcErr != nil {
			log.Fatalf("Error occurred while starting grpc connection on description service, %v", grpcErr)
		}
		defer grpcConnDescription.Close()

		descriptionClient := descriptionproto.NewDescriptionServiceClient(grpcConnDescription)
		descriptionGateway = descriptiondlv.NewDescriptionGatewayAdapter(descriptionClient)

		grpcConnActionProcessor, grpcErr := grpcconnection.CreateGRPCConn(
			cfg.Services.ActionProcessor.Host,
			cfg.Services.ActionProcessor.Port,
		)
		if grpcErr != nil {
			log.Fatalf("Failed to connect to ActionProcessorService: %v", grpcErr)
		}
		defer grpcConnActionProcessor.Close()

		actionProcessorClient := bestiaryproto.NewActionProcessorServiceClient(grpcConnActionProcessor)
		actionProcessorGateway = bestiarydlv.NewActionProcessorAdapter(actionProcessorClient)

		proxieAddr := fmt.Sprintf("%s:%s", cfg.Proxies.Socks5Proxie.IP, cfg.Proxies.Socks5Proxie.Port)

		proxyClient, proxyErr := socks5proxy.NewProxiedHttpClient(proxieAddr)
		if proxyErr != nil {
			log.Fatalf("failed to create proxy client: %v", proxyErr)
		}

		geminiURL := fmt.Sprintf("%s:%s", cfg.Gemini.Host, cfg.Gemini.Port)
		geminiAPI = bestiaryext.NewGeminiClient(geminiURL, cfg.Gemini.ExternalVM1, proxyClient)
	}

	vkClient := authext.NewVKApi(cfg.VKApi.RedirectURI, cfg.VKApi.ClientID, cfg.VKApi.SecretKey, cfg.VKApi.ServiceKey,
		cfg.VKApi.Exchange, cfg.VKApi.PublicInfo)

	mongoURI := dbinit.NewMongoConnectionURI(cfg.Mongo.Username, cfg.Mongo.Password, cfg.Mongo.Host,
		cfg.Mongo.Port, !isProduction && !isTestMode)
	mongoDatabase := dbinit.ConnectToMongoDatabase(context.Background(), mongoURI, cfg.Mongo.DBName)
	logger.DBInfo(cfg.Mongo.Host, cfg.Mongo.Port, "mongodb", cfg.Mongo.DBName, !isProduction)

	mongoMetrics, err := metrics.NewDBMetrics("mongo")
	if err != nil {
		log.Fatal("Something went wrong initializing prometheus mongo metrics, ", err)
	}

	minioURI := dbinit.NewMinioEndpoint(cfg.Minio.Host)
	if isTestMode && cfg.Minio.Port != "" {
		minioURI = dbinit.NewMinioEndpoint(cfg.Minio.Host, cfg.Minio.Port)
	}
	minioClient, err := dbinit.ConnectToMinio(minioURI, cfg.Minio.AccessKey, cfg.Minio.SecretKey, !isTestMode)
	if err != nil {
		logger.DBFatal(cfg.Minio.Host, cfg.Minio.Port, "minio", "", true,
			"Failed to initialize MinIO client", err)
	}

	_, err = minioClient.ListBuckets(context.Background())
	if err != nil {
		logger.DBFatal(cfg.Minio.Host, cfg.Minio.Port, "minio", "", true,
			"Failed to connect to MinIO server", err)
	}

	logger.DBInfo(cfg.Minio.Host, cfg.Minio.Port, "minio", "", true)

	postgresURL := dbinit.NewConnectionString(cfg.Postgres.Username, cfg.Postgres.Password,
		cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.DBName)
	postgresPool, err := dbinit.NewPostgresPool(postgresURL)
	if err != nil {
		logger.DBFatal(cfg.Postgres.Host, cfg.Postgres.Port, "postgres", cfg.Postgres.DBName, false,
			"Something went wrong while creating postgres pool", err)
	}

	err = postgresPool.Ping(context.Background())
	if err != nil {
		logger.DBFatal(cfg.Postgres.Host, cfg.Postgres.Port, "postgres", cfg.Postgres.DBName, false,
			"Cannot ping postgres database", err)
	}
	logger.DBInfo(cfg.Postgres.Host, cfg.Postgres.Port, "postgres", cfg.Postgres.DBName, false)

	postgresMetrics, err := metrics.NewDBMetrics("postgres")
	if err != nil {
		log.Fatal("Something went wrong initializing prometheus postgres metrics, ", err)
	}

	redisClient := dbinit.NewRedisClient(cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.Password, cfg.Redis.DB)

	err = redisClient.Ping(context.Background()).Err()
	if err != nil {
		logger.DBFatal(cfg.Redis.Host, cfg.Redis.Port, "redis", cfg.Redis.DB, false, "Cannot ping Redis", err)
	}
	logger.DBInfo(cfg.Redis.Host, cfg.Redis.Port, "redis", cfg.Redis.DB, false)

	redisMetrics, err := metrics.NewDBMetrics("redis")
	if err != nil {
		log.Fatal("Something went wrong initializing prometheus redis metrics, ", err)
	}

	wsMetrics, err := metrics.NewWSMetrics()
	if err != nil {
		log.Fatal("Something went wrong initializing websocket metrics, ", err)
	}

	wsSessionMetrics, err := metrics.NewWSSessionMetrics()
	if err != nil {
		log.Fatal("Something went wrong initializing websocket session metrics, ", err)
	}

	bestiaryRepository := bestiaryrepo.NewBestiaryStorage(mongoDatabase, mongoMetrics)
	bestiaryS3Manager := bestiaryrepo.NewMinioManager(minioClient, "creature-images")
	llmInmemoryStorage := bestiaryrepo.NewInMemoryLLMRepo()
	characterRepository := characterrepo.NewCharacterStorage(mongoDatabase, mongoMetrics)
	characterBaseRepository := characterrepo.NewCharacterBaseStorage(mongoDatabase, mongoMetrics)
	characterAvatarS3Manager := characterrepo.NewAvatarS3Manager(minioClient, "character-avatars")
	encounterRepository := encounterrepo.NewEncounterStorage(postgresPool, postgresMetrics)
	maptileRepository := maptilerepo.NewMapTilesStorage(mongoDatabase, mongoMetrics)
	mapsRepository := mapsrepo.NewMapsStorage(postgresPool, postgresMetrics)
	authRepository := authrepo.NewAuthStorage(postgresPool, postgresMetrics)
	identityRepository := authrepo.NewIdentityStorage(postgresPool, postgresMetrics)
	sessionManager := authrepo.NewSessionManager(redisClient, redisMetrics)
	tableManager := tablerepo.NewTableManager(wsMetrics, wsSessionMetrics)

	bestiaryUsecases := bestiaryuc.NewBestiaryUsecases(bestiaryRepository, bestiaryS3Manager, geminiAPI)
	actionProcessorUsecase := bestiaryuc.NewActionProcessorUsecase(actionProcessorGateway)
	generatedCreatureProcessor := bestiaryuc.NewGeneratedCreatureProcessor(actionProcessorUsecase)
	llmUsecases := bestiaryuc.NewLLMUsecase(llmInmemoryStorage, geminiAPI, generatedCreatureProcessor,
		bestiaryuc.NewGoRunner(), bestiaryuc.NewUUIDGenerator())
	descriptionUsecases := descriptionuc.NewDescriptionUsecase(descriptionGateway)
	characterUsecases := characteruc.NewCharacterUsecases(characterRepository)
	characterBaseUsecases := characteruc.NewCharacterBaseUsecases(characterBaseRepository, characterAvatarS3Manager)
	encounterUsecases := encounteruc.NewEncounterUsecases(encounterRepository)
	googleClient := authext.NewGoogleOAuth(cfg.GoogleOAuth.ClientID, cfg.GoogleOAuth.ClientSecret,
		cfg.GoogleOAuth.RedirectURI)
	yandexClient := authext.NewYandexOAuth(cfg.YandexOAuth.ClientID, cfg.YandexOAuth.ClientSecret)

	oauthProviders := map[string]authinterface.OAuthProvider{
		vkClient.Name():     vkClient,
		googleClient.Name(): googleClient,
		yandexClient.Name(): yandexClient,
	}
	authUsecases := authuc.NewAuthUsecases(authRepository, identityRepository, oauthProviders, sessionManager)
	tableUsecases := tableuc.NewTableUsecases(encounterRepository, tableManager,
		tableuc.NewRandSessionIDGen(), tableuc.NewRealTimerFactory())
	maptilesUsecases := maptileuc.NewMapTilesUsecases(maptileRepository)
	mapsUsecases := mapsuc.NewMapsUsecases(mapsRepository)

	spellsRepository := spellsrepo.NewSpellsStorage(mongoDatabase, mongoMetrics)
	if err := spellsRepository.EnsureIndexes(context.Background()); err != nil {
		log.Printf("Warning: failed to ensure spell indexes: %v", err)
	}
	if n, err := spellsseed.SeedSpellDefinitions(context.Background(), spellsRepository); err != nil {
		log.Printf("Warning: failed to seed spell definitions: %v", err)
	} else if n > 0 {
		log.Printf("Seeded %d spell definitions", n)
	}
	spellsUsecases := spellsuc.NewSpellsUsecases(spellsRepository)

	featuresRepository := featuresrepo.NewFeaturesStorage(mongoDatabase, mongoMetrics)
	if err := featuresRepository.EnsureIndexes(context.Background()); err != nil {
		log.Printf("Warning: failed to ensure feature indexes: %v", err)
	}
	if n, err := featuresseed.SeedFeatureDefinitions(context.Background(), featuresRepository); err != nil {
		log.Printf("Warning: failed to seed feature definitions: %v", err)
	} else if n > 0 {
		log.Printf("Seeded %d feature definitions", n)
	}
	featuresUsecases := featuresuc.NewFeaturesUsecases(featuresRepository)

	classesRepository := classesrepo.NewClassesStorage(mongoDatabase, mongoMetrics)
	if err := classesRepository.EnsureIndexes(context.Background()); err != nil {
		log.Printf("Warning: failed to ensure class indexes: %v", err)
	}
	if n, err := classesseed.SeedClassDefinitions(context.Background(), classesRepository); err != nil {
		log.Printf("Warning: failed to seed class definitions: %v", err)
	} else if n > 0 {
		log.Printf("Seeded %d class definitions", n)
	}
	classesUsecases := classesuc.NewClassesUsecases(classesRepository)

	racesRepository := racesrepo.NewRacesStorage(mongoDatabase, mongoMetrics)
	if err := racesRepository.EnsureIndexes(context.Background()); err != nil {
		log.Printf("Warning: failed to ensure race indexes: %v", err)
	}
	if n, err := racesseed.SeedRaceDefinitions(context.Background(), racesRepository); err != nil {
		log.Printf("Warning: failed to seed race definitions: %v", err)
	} else if n > 0 {
		log.Printf("Seeded %d race definitions", n)
	}
	racesUsecases := racesuc.NewRacesUsecases(racesRepository)

	backgroundsRepository := backgroundsrepo.NewBackgroundsStorage(mongoDatabase, mongoMetrics)
	if err := backgroundsRepository.EnsureIndexes(context.Background()); err != nil {
		log.Printf("Warning: failed to ensure background indexes: %v", err)
	}
	if n, err := backgroundsseed.SeedBackgroundDefinitions(context.Background(), backgroundsRepository); err != nil {
		log.Printf("Warning: failed to seed background definitions: %v", err)
	} else if n > 0 {
		log.Printf("Seeded %d background definitions", n)
	}
	backgroundsUsecases := backgroundsuc.NewBackgroundsUsecases(backgroundsRepository)

	featsRepository := featsrepo.NewFeatsStorage(mongoDatabase, mongoMetrics)
	if err := featsRepository.EnsureIndexes(context.Background()); err != nil {
		log.Printf("Warning: failed to ensure feat indexes: %v", err)
	}
	if n, err := featsseed.SeedFeatDefinitions(context.Background(), featsRepository); err != nil {
		log.Printf("Warning: failed to seed feat definitions: %v", err)
	} else if n > 0 {
		log.Printf("Seeded %d feat definitions", n)
	}
	featsUsecases := featsuc.NewFeatsUsecases(featsRepository)

	itemsRepository := itemsrepo.NewItemsStorage(mongoDatabase, mongoMetrics)
	if err := itemsRepository.EnsureItemDefinitionIndexes(context.Background()); err != nil {
		log.Printf("Warning: failed to ensure item definition indexes: %v", err)
	}
	if n, err := itemsseed.SeedItemDefinitions(context.Background(), itemsRepository); err != nil {
		log.Printf("Warning: failed to seed item definitions: %v", err)
	} else if n > 0 {
		log.Printf("Seeded %d item definitions", n)
	}
	inventoryRepository := itemsrepo.NewInventoryStorage(mongoDatabase, mongoMetrics)
	if err := inventoryRepository.EnsureInventoryContainerIndexes(context.Background()); err != nil {
		log.Printf("Warning: failed to ensure inventory container indexes: %v", err)
	}
	itemsUsecases := itemsuc.NewItemUsecases(itemsRepository)
	inventoryUsecases := itemsuc.NewInventoryUsecases(inventoryRepository, itemsRepository)

	auditLogRepository := auditlogrepo.NewAuditLogStorage(mongoDatabase, mongoMetrics)
	if err := auditLogRepository.EnsureIndexes(context.Background()); err != nil {
		log.Printf("Warning: failed to ensure audit log indexes: %v", err)
	}

	tileMetadataRepository := dungeongenrepo.NewTileMetadataStorage(mongoDatabase, mongoMetrics)
	if err := tileMetadataRepository.EnsureIndexes(context.Background()); err != nil {
		log.Printf("Warning: failed to ensure tile metadata indexes: %v", err)
	}
	if n, err := dungeongen.BatchClassifyTiles(context.Background(), maptileRepository, tileMetadataRepository); err != nil {
		log.Printf("Warning: failed to batch-classify tiles: %v", err)
	} else if n > 0 {
		log.Printf("Classified %d tiles into tile_metadata", n)
	}

	dungeonGenUsecases := dungeongenuc.NewDungeonGenUsecases(
		tileMetadataRepository, maptileRepository, bestiaryRepository, "default",
	)

	actionsUsecases := actionsuc.NewActionsUsecases(encounterRepository, characterBaseRepository, spellsRepository, bestiaryRepository, auditLogRepository)

	combatAIEngine := combatai.NewRuleBasedAI()
	combatAIUsecases := combataiuc.NewCombatAIUsecases(
		combatAIEngine, encounterRepository, bestiaryRepository,
		characterBaseRepository, actionsUsecases, auditLogRepository, tableManager,
	)

	// Inject reaction evaluator into actions usecases (breaks circular DI).
	reactionEval := combataiuc.NewReactionEvaluator(bestiaryRepository)
	if setter, ok := actionsUsecases.(actionsinterfaces.ReactionEvaluatorSetter); ok {
		setter.SetReactionEvaluator(reactionEval)
	} else {
		log.Fatal("actionsUsecases does not implement ReactionEvaluatorSetter — DI wiring is broken")
	}

	credentials := handlers.AllowCredentials()
	headersOk := handlers.AllowedHeaders(cfg.Server.Headers)
	originsOk := handlers.AllowedOrigins(cfg.Server.Origins)
	methodsOk := handlers.AllowedMethods(cfg.Server.Methods)

	router := myrouter.NewRouter(
		cfg,
		logger,
		m,
		bestiaryUsecases,
		descriptionUsecases,
		characterUsecases,
		characterBaseUsecases,
		encounterUsecases,
		authUsecases,
		tableUsecases,
		llmUsecases,
		maptilesUsecases,
		mapsUsecases,
		spellsUsecases,
		featuresUsecases,
		itemsUsecases,
		inventoryUsecases,
		tableManager,
		actionsUsecases,
		classesUsecases,
		racesUsecases,
		backgroundsUsecases,
		featsUsecases,
		combatAIUsecases,
		dungeonGenUsecases,
	)
	muxWithCORS := handlers.CORS(credentials, originsOk, headersOk, methodsOk)(router)

	serverURL := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)

	serverCfg := createServerConfig(serverURL, cfg.Server.Timeout, &muxWithCORS)
	srv.server = createServer(serverCfg)

	logger.ServerInfo(cfg.Server.Host, cfg.Server.Port, isProduction)

	// Graceful shutdown: listen for SIGINT/SIGTERM.
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() { errCh <- srv.server.ListenAndServe() }()

	select {
	case <-sigCtx.Done():
		log.Println("Shutting down server...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := srv.server.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	}

	// Close database connections.
	postgresPool.Close()
	if err := redisClient.Close(); err != nil {
		log.Printf("Redis close error: %v", err)
	}
	if err := mongoDatabase.Client().Disconnect(context.Background()); err != nil {
		log.Printf("MongoDB disconnect error: %v", err)
	}

	log.Println("Server stopped gracefully")
	return nil
}

package server

import (
	"context"
	"fmt"
	mylogger "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbinit"
	"log"
	"net/http"
	"os"
	"time"

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

	bestiaryproto "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/delivery/protobuf"
	bestiaryext "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/external"
	bestiaryrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/repository"
	bestiaryuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/usecases"
	characterrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character/repository"
	characteruc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character/usecases"
	descriptionproto "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description/delivery/protobuf"
	descriptionuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description/usecases"
	encounterrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter/repository"
	encounteruc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter/usecases"
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

	var isProduction bool

	if os.Getenv("SERVER_MODE") == "production" {
		isProduction = true
	}

	cfg := config.ReadConfig(cfgPath)
	if cfg == nil {
		log.Fatal("The config wasn`t opened")
	}

	logger, err := mylogger.New(cfg.Logger.Key, cfg.Logger.OutputPath, cfg.Logger.ErrPath)
	if err != nil {
		log.Fatalf("Failed to initialize log: %v", err)
	}
	defer logger.Sync()

	m, err := metrics.NewHTTPMetrics()
	if err != nil {
		log.Fatal("Something went wrong initializing prometheus app metrics, ", err)
	}

	descriptionAddr := fmt.Sprintf("%s:%s", cfg.Services.Description.Host, cfg.Services.Description.Port)

	grpcConnDescription, err := grpc.NewClient(
		descriptionAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Error occurred while starting grpc connection on description service, %v", err)
	}
	defer grpcConnDescription.Close()

	descriptionClient := descriptionproto.NewDescriptionServiceClient(grpcConnDescription)

	grpcConnActionProcessor, err := grpcconnection.CreateGRPCConn(
		cfg.Services.ActionProcessor.Host,
		cfg.Services.ActionProcessor.Port,
	)
	if err != nil {
		log.Fatalf("Failed to connect to ActionProcessorService: %v", err)
	}
	defer grpcConnActionProcessor.Close()

	actionProcessorClient := bestiaryproto.NewActionProcessorServiceClient(grpcConnActionProcessor)

	proxieAddr := fmt.Sprintf("%s:%s", cfg.Proxies.Socks5Proxie.IP, cfg.Proxies.Socks5Proxie.Port)

	proxyClient, err := socks5proxy.NewProxiedHttpClient(proxieAddr)
	if err != nil {
		log.Fatalf("failed to create proxy client: %v", err)
	}

	geminiURL := fmt.Sprintf("%s:%s", cfg.Gemini.Host, cfg.Gemini.Port)
	geminiClient := bestiaryext.NewGeminiClient(geminiURL, cfg.Gemini.ExternalVM1, proxyClient)

	mongoURI := dbinit.NewMongoConnectionURI(cfg.Mongo.Username, cfg.Mongo.Password, cfg.Mongo.Host,
		cfg.Mongo.Port, !isProduction)
	mongoDatabase := dbinit.ConnectToMongoDatabase(context.Background(), mongoURI, cfg.Mongo.DBName)
	logger.DBInfo(cfg.Mongo.Host, cfg.Mongo.Port, "mongodb", cfg.Mongo.DBName, !isProduction)

	mongoMetrics, err := metrics.NewDBMetrics("mongo")
	if err != nil {
		log.Fatal("Something went wrong initializing prometheus mongo metrics, ", err)
	}

	minioURI := dbinit.NewMinioEndpoint(cfg.Minio.Host)
	minioClient := dbinit.ConnectToMinio(context.Background(), minioURI, cfg.Minio.AccessKey,
		cfg.Minio.SecretKey, true)
	logger.DBInfo(cfg.Minio.Host, cfg.Minio.Port, "minio", "", true)

	postgresURL := dbinit.NewConnectionString(cfg.Postgres.Username, cfg.Postgres.Password,
		cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.DBName)
	postgresPool, err := dbinit.NewPostgresPool(postgresURL)
	if err != nil {
		logger.DBFatal(cfg.Postgres.Host, cfg.Postgres.Port, "postgres", cfg.Postgres.DBName, false,
			"Something went wrong while creating postgres pool", err)
	}
	logger.DBInfo(cfg.Postgres.Host, cfg.Postgres.Port, "postgres", cfg.Postgres.DBName, false)

	err = postgresPool.Ping(context.Background())
	if err != nil {
		logger.DBFatal(cfg.Postgres.Host, cfg.Postgres.Port, "postgres", cfg.Postgres.DBName, false,
			"Cannot ping postgres database", err)
	}

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
	encounterRepository := encounterrepo.NewEncounterStorage(postgresPool, postgresMetrics)
	authRepository := authrepo.NewAuthStorage(postgresPool, postgresMetrics)
	sessionManager := authrepo.NewSessionManager(redisClient, redisMetrics)
	tableManager := tablerepo.NewTableManager(wsMetrics, wsSessionMetrics)

	bestiaryUsecases := bestiaryuc.NewBestiaryUsecases(bestiaryRepository, bestiaryS3Manager, geminiClient)
	actionProcessorUsecase := bestiaryuc.NewActionProcessorUsecase(actionProcessorClient)
	generatedCreatureProcessor := bestiaryuc.NewGeneratedCreatureProcessor(actionProcessorUsecase)
	llmUsecases := bestiaryuc.NewLLMUsecase(llmInmemoryStorage, geminiClient, generatedCreatureProcessor)
	descriptionUsecases := descriptionuc.NewDescriptionUsecase(descriptionClient)
	characterUsecases := characteruc.NewCharacterUsecases(characterRepository)
	encounterUsecases := encounteruc.NewEncounterUsecases(encounterRepository)
	authUsecases := authuc.NewAuthUsecases(authRepository, sessionManager)
	tableUsecases := tableuc.NewTableUsecases(encounterRepository, tableManager)

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
		encounterUsecases,
		authUsecases,
		tableUsecases,
		llmUsecases,
	)
	muxWithCORS := handlers.CORS(credentials, originsOk, headersOk, methodsOk)(router)

	serverURL := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)

	serverCfg := createServerConfig(serverURL, cfg.Server.Timeout, &muxWithCORS)
	srv.server = createServer(serverCfg)

	logger.ServerInfo(cfg.Server.Host, cfg.Server.Port, isProduction)

	return srv.server.ListenAndServe()
}

package server

import (
	"context"
	"fmt"
	authrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth/repository"
	authuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth/usecases"
	"log"
	"net/http"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/config"
	myrouter "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/routers"
	"github.com/gorilla/handlers"

	bestiaryrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/repository"
	bestiaryuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/usecases"
	characterrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character/repository"
	characteruc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character/usecases"
	descriptionproto "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description/delivery/protobuf"
	descriptionuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description/usecases"
	encounterrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter/repository"
	encounteruc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter/usecases"
	serverrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository"
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

	cfg := config.ReadConfig(cfgPath)
	if cfg == nil {
		log.Fatal("The config wasn`t opened")
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

	mongoURI := serverrepo.NewMongoConnectionURI(cfg.Mongo.Username, cfg.Mongo.Password, cfg.Mongo.Host, cfg.Mongo.Port)

	mongoDatabase := serverrepo.ConnectToMongoDatabase(context.Background(), mongoURI, cfg.Mongo.DBName)

	postgresURL := serverrepo.NewConnectionString(cfg.Postgres.Username, cfg.Postgres.Password,
		cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.DBName)

	postgresPool, err := serverrepo.NewPostgresPool(postgresURL)
	if err != nil {
		log.Fatal("Something went wrong while creating postgres pool ", err)
	}

	err = postgresPool.Ping(context.Background())
	if err != nil {
		log.Fatal("Cannot ping postgres database ", err)
	}

	redisClient := serverrepo.NewRedisClient(cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.Password, cfg.Redis.DB)

	err = redisClient.Ping(context.Background()).Err()
	if err != nil {
		log.Fatalf("Cannot ping Redis: %v", err)
	}

	bestiaryRepository := bestiaryrepo.NewBestiaryStorage(mongoDatabase)
	characterRepository := characterrepo.NewCharacterStorage(mongoDatabase)
	encounterRepository := encounterrepo.NewEncounterStorage(mongoDatabase)
	authRepository := authrepo.NewAuthStorage()

	bestiaryUsecases := bestiaryuc.NewBestiaryUsecases(bestiaryRepository)
	descriptionUsecases := descriptionuc.NewDescriptionUsecase(descriptionClient)
	characterUsecases := characteruc.NewCharacterUsecases(characterRepository)
	encounterUsecases := encounteruc.NewEncounterUsecases(encounterRepository)
	authUsecases := authuc.NewAuthUsecases(authRepository)

	credentials := handlers.AllowCredentials()
	headersOk := handlers.AllowedHeaders(cfg.Server.Headers)
	originsOk := handlers.AllowedOrigins(cfg.Server.Origins)
	methodsOk := handlers.AllowedMethods(cfg.Server.Methods)

	router := myrouter.NewRouter(
		cfg,
		bestiaryUsecases,
		descriptionUsecases,
		characterUsecases,
		encounterUsecases,
		authUsecases,
	)
	muxWithCORS := handlers.CORS(credentials, originsOk, headersOk, methodsOk)(router)

	serverURL := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)

	serverCfg := createServerConfig(serverURL, cfg.Server.Timeout, &muxWithCORS)
	srv.server = createServer(serverCfg)

	log.Printf("Server is listening on %s\n", serverURL)

	return srv.server.ListenAndServe()
}

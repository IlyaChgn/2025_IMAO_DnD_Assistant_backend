package server

import (
	"context"
	"fmt"
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

	bestiaryRepository := bestiaryrepo.NewBestiaryStorage(mongoDatabase)
	characterRepository := characterrepo.NewCharacterStorage(mongoDatabase)
	encounterRepository := encounterrepo.NewEncounterStorage(mongoDatabase)

	bestiaryUsecases := bestiaryuc.NewBestiaryUsecases(bestiaryRepository)
	descriptionUsecases := descriptionuc.NewDescriptionUsecase(descriptionClient)
	characterUsecases := characteruc.NewCharacterUsecases(characterRepository)
	encounterUsecases := encounteruc.NewEncounterUsecases(encounterRepository)

	credentials := handlers.AllowCredentials()
	headersOk := handlers.AllowedHeaders(cfg.Server.Headers)
	originsOk := handlers.AllowedOrigins(cfg.Server.Origins)
	methodsOk := handlers.AllowedMethods(cfg.Server.Methods)

	router := myrouter.NewRouter(bestiaryUsecases, descriptionUsecases,
		characterUsecases, encounterUsecases)
	muxWithCORS := handlers.CORS(credentials, originsOk, headersOk, methodsOk)(router)

	serverURL := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)

	serverCfg := createServerConfig(serverURL, cfg.Server.Timeout, &muxWithCORS)
	srv.server = createServer(serverCfg)

	log.Printf("Server is listening on %s\n", serverURL)

	return srv.server.ListenAndServe()
}

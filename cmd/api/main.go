// @title DevLab API
// @version 1.0
// @description DevLab - Cloud-Based Coding Environment Provisioner
// @termsOfService http://swagger.io/terms/

// @contact.name DevLab Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8000
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter the token with the `Bearer ` prefix, e.g. "Bearer abcde12345". Do NOT include the quotes around the entire value.

package main

import (
	"context"
	_ "devlab/docs/api"
	"devlab/internal/api"
	"devlab/internal/config"
	"devlab/internal/docker"
	"devlab/internal/scenario"
	"devlab/internal/storage"
	pb "devlab/proto"
	"net"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	zerologlog "github.com/rs/zerolog/log"
	ginSwaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	otelgin "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	otelgrpc "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"google.golang.org/grpc"
)

func initTracer() func() {
	exp, _ := stdouttrace.New(stdouttrace.WithPrettyPrint())
	provider := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("devlab-api"),
		)),
	)
	otel.SetTracerProvider(provider)
	return func() { _ = provider.Shutdown(context.Background()) }
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerologlog.Logger = zerologlog.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	shutdown := initTracer()
	defer shutdown()

	cfg := config.Load()
	mongoClient, err := storage.GetMongoClient(context.Background(), cfg.MongoURI)
	if err != nil {
		zerologlog.Fatal().Err(err).Msg("failed to connect to MongoDB")
	}
	db := mongoClient.Database(cfg.DBName)
	dockerClient := docker.RealClient{}
	scenarioManager := scenario.NewManager(cfg, db, dockerClient)
	handler := &api.Handler{Scenario: scenarioManager}

	// REST API
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(otelgin.Middleware("devlab-api"))

	// Swagger docs endpoint
	r.GET("/swagger/*any", ginSwagger.WrapHandler(ginSwaggerFiles.Handler))
	// Health endpoint (no auth)
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Protected scenario endpoints
	scenarioGroup := r.Group("/")
	scenarioGroup.Use(api.JWTAuthMiddleware())
	scenarioGroup.POST("/scenarios/start", handler.StartScenarioREST)
	scenarioGroup.GET("/scenarios/types", handler.GetScenarioTypesREST)
	scenarioGroup.GET("/scenarios/:id/status", handler.GetScenarioStatusREST)
	scenarioGroup.GET("/scenarios/:id/terminal", handler.GetTerminalURLREST)
	scenarioGroup.GET("/scenarios/:id/directory", handler.GetDirectoryStructureREST)
	scenarioGroup.DELETE("/scenarios/:id", handler.StopScenarioREST)
	go func() {
		zerologlog.Info().Msg("API server running on :8000")
		r.Run(":8000")
	}()

	// gRPC server
	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
	pb.RegisterScenarioServiceServer(grpcServer, &api.GRPCServer{Scenario: scenarioManager})
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		zerologlog.Fatal().Err(err).Msg("failed to listen")
	}
	zerologlog.Info().Msg("gRPC server running on :9090")
	if err := grpcServer.Serve(lis); err != nil {
		zerologlog.Fatal().Err(err).Msg("failed to serve")
	}
}

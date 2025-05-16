package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/teresa-solution/tenant-management-service/internal/monitoring"
	"github.com/teresa-solution/tenant-management-service/internal/service"
	"github.com/teresa-solution/tenant-management-service/internal/store"
	tenantpb "github.com/teresa-solution/tenant-management-service/proto/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	var (
		port   = flag.Int("port", 50051, "Port gRPC server")
		dbHost = flag.String("db-host", "localhost", "Database host")
		dbPort = flag.Int("db-port", 5432, "Database port")
		dbUser = flag.String("db-user", "admin", "Database user")
		dbPass = flag.String("db-pass", "securepassword", "Database password")
		dbName = flag.String("db-name", "tenant_registry", "Database name")
	)
	flag.Parse()

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		*dbHost, *dbPort, *dbUser, *dbPass, *dbName)

	repo, err := store.NewTenantRepository(dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer repo.Close()

	tenantService := service.NewTenantService(repo)

	// Initialize metrics
	monitoring.InitMetrics()

	log.Info().Msgf("Starting Tenant Management Service on port %d", *port)

	// Load TLS credentials for gRPC
	certFile := "certs/cert.pem"
	keyFile := "certs/key.pem"
	creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load TLS credentials")
	}

	// Create gRPC server with TLS
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to listen")
	}

	server := grpc.NewServer(grpc.Creds(creds))
	tenantpb.RegisterTenantServiceServer(server, tenantService)
	reflection.Register(server)

	go func() {
		log.Info().Msgf("gRPC server listening at %v with TLS", lis.Addr())
		if err := server.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("Failed to start gRPC server")
		}
	}()

	// Update HTTP server to use HTTPS for metrics and health
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.Handle("/metrics", promhttp.Handler())

	go func() {
		httpServer := &http.Server{
			Addr:    ":8081",
			Handler: mux,
		}
		log.Info().Msg("HTTPS server for health checks and metrics started on port 8081")
		if err := httpServer.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("HTTPS server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down server...")

	server.GracefulStop()
	log.Info().Msg("Server exiting")
}

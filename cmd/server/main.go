package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/teresa-solution/tenant-management-service/internal/store"
	"google.golang.org/grpc"
)

func main() {
	// Configure logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Parse command line flags
	var (
		port   = flag.Int("port", 50051, "Port gRPC server")
		dbHost = flag.String("db-host", "localhost", "Database host")
		dbPort = flag.Int("db-port", 5432, "Database port")
		dbUser = flag.String("db-user", "admin", "Database user")
		dbPass = flag.String("db-pass", "securepassword", "Database password")
		dbName = flag.String("db-name", "tenant_registry", "Database name")
	)
	flag.Parse()

	// Construct DSN
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		*dbHost, *dbPort, *dbUser, *dbPass, *dbName)

	// Initialize database repository
	repo, err := store.NewTenantRepository(dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer repo.Close()

	log.Info().Msgf("Starting Tenant Management Service on port %d", *port)

	// Setup gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to listen")
	}

	server := grpc.NewServer()

	// Register service gRPC (will be implemented later)
	// pb.RegisterTenantServiceServer(server, &service.TenantServiceServer{})

	// Start server in goroutine
	go func() {
		log.Info().Msgf("gRPC server listening at %v", lis.Addr())
		if err := server.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("Failed to start gRPC server")
		}
	}()

	// Setup HTTP server for health checks and metrics
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		httpServer := &http.Server{
			Addr:    ":8081",
			Handler: mux,
		}

		log.Info().Msg("HTTP server for health checks started on port 8081")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("HTTP server error")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down server...")

	server.GracefulStop()
	log.Info().Msg("Server exiting")
}

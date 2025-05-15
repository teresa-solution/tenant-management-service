package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configure logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Parse command line flags
	var (
		dbHost  = flag.String("db-host", "localhost", "Database host")
		dbPort  = flag.Int("db-port", 5432, "Database port")
		dbUser  = flag.String("db-user", "admin", "Database user")
		dbPass  = flag.String("db-pass", "securepassword", "Database password")
		dbName  = flag.String("db-name", "tenant_registry", "Database name")
		command = flag.String("command", "up", "Migration command (up, down, force)")
	)
	flag.Parse()

	// Construct DSN
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		*dbHost, *dbPort, *dbUser, *dbPass, *dbName)

	// Connect to the database
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse DSN")
	}
	db := stdlib.OpenDB(*config)
	defer db.Close()

	// Set up the migration driver
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create migration driver")
	}

	// Create the migrator
	m, err := migrate.NewWithDatabaseInstance(
		"file://scripts/migrations",
		"postgres", driver,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create migrator")
	}

	// Run the migration command
	switch *command {
	case "up":
		log.Info().Msg("Applying migrations...")
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatal().Err(err).Msg("Failed to apply migrations")
		}
		log.Info().Msg("Migrations applied successfully")
	case "down":
		log.Info().Msg("Reverting migrations...")
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatal().Err(err).Msg("Failed to revert migrations")
		}
		log.Info().Msg("Migrations reverted successfully")
	case "force":
		log.Info().Msg("Forcing migration version...")
		if err := m.Force(1); err != nil {
			log.Fatal().Err(err).Msg("Failed to force migration version")
		}
		log.Info().Msg("Migration version forced successfully")
	default:
		log.Fatal().Msgf("Unknown command: %s", *command)
	}
}

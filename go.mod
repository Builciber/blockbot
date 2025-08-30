module github.com/Builciber/blockbot

go 1.23.6

toolchain go1.23.11

replace internal/database v0.0.0 => ./internal/database

require github.com/go-telegram/bot v1.8.2

require (
	github.com/jackc/pgx/v5 v5.7.5
	github.com/joho/godotenv v1.5.1
)

require (
	github.com/go-chi/chi/v5 v5.2.3 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/crypto v0.40.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/text v0.27.0 // indirect
)

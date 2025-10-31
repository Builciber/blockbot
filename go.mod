module github.com/Builciber/blockbot

go 1.24.0

toolchain go1.24.7

replace internal/database v0.0.0 => ./internal/database

require github.com/go-telegram/bot v1.8.2

require (
	github.com/fogleman/gg v1.3.0
	github.com/go-chi/chi/v5 v5.2.3
	github.com/jackc/pgx/v5 v5.7.5
	github.com/joho/godotenv v1.5.1
)

require (
	github.com/boombuler/barcode v1.1.0 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/crypto v0.40.0 // indirect
	golang.org/x/image v0.31.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/text v0.29.0 // indirect
)

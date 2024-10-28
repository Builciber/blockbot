module github.com/Builciber/blockbot

go 1.22.6

replace internal/database v0.0.0 => ./internal/database

require internal/database v0.0.0

require (
	github.com/go-telegram/bot v1.8.2
)

require (
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9
)

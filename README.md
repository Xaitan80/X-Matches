![Build & Publish Docker](https://github.com/xaitan80/X-Matches/actions/workflows/docker-publish.yml/badge.svg)

# X‑Matches

En liten app för att lägga in matcher, se dem i en enkel vy och uppdatera resultat. Backend är Go + SQLite, migrationer körs automatiskt vid start.

## Köra lokalt

- Krav: Go 1.23+ (CGO påslaget), SQLite C‑toolchain (macOS: Xcode CLT; Linux: build‑essential)
- Standardport: `:8080`
- Standarddatabas: `xmatches.db` i aktuell katalog

Snabbstart:

```
go run .
# Öppna http://localhost:8080
```

Valfria miljövariabler:

- `ADDR`: adress/port (default `:8080`)
- `DB_PATH`: sökväg till SQLite‑fil (default `xmatches.db`)

Exempel:

```
ADDR=127.0.0.1:9000 DB_PATH=/tmp/xmatches.db go run .
```

Migrationer: Tabellen skapas/uppgraderas automatiskt vid start (inbakade Goose‑migrationer).

## Köra med Docker (lokal build)

Bygg och kör med beständigt volym‑lagring för databasen:

```
make docker-build            # bygger bild (default tag: xmatches:local)
make docker-run              # mappar 8080 och volymen xmatches-data
# Öppna http://localhost:8080
```

## Köra med Docker Compose (förbyggd bild)

`docker-compose.yml` använder publicerade bilden `xaitan/x-matches:latest` och monterar en named volume för `/data` där `xmatches.db` sparas.

```
make compose-up              # bygger/bootar via docker compose
make compose-down            # stoppar och tar bort containrar
```

Miljö i compose:

- `ADDR=:8080`
- `DB_PATH=/data/xmatches.db` (ligger på volymen `xmatches-data`)

## API‑snabbguide

- Bas‑URL: `http://localhost:8080`
- UI: `GET /` (enkel webbsida)
- Lista matcher: `GET /api/matches`
- Exportera CSV: `GET /api/matches.csv` (laddar ner `matches_YYYY-MM-DD.csv`)
- Hämta match: `GET /api/matches/:id`
- Skapa match: `POST /api/matches`
- Uppdatera match: `PATCH /api/matches/:id`
- Radera match: `DELETE /api/matches/:id`

Minimal `POST`‑exempel:

```
curl -X POST http://localhost:8080/api/matches \
  -H 'Content-Type: application/json' \
  -d '{
    "date_raw":"2025-09-20",
    "time_raw":"14:30",
    "team":"IFK X F11",
    "opponent":"BK Y",
    "top_scorer_team":"A. Svensson (3)",
    "top_scorer_opponent":"K. Karlsson (2)"
  }'
```

Markera som spelad (PATCH‑exempel):

```
curl -X PATCH http://localhost:8080/api/matches/1 \
  -H 'Content-Type: application/json' \
  -d '{"played": true, "goals_for": 3, "goals_against": 1}'
```

## Utveckling

- Formattering: `make fmt`
- Rensa databasen: stoppa appen och radera `xmatches.db` (eller byt `DB_PATH`).

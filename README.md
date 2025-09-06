![Build & Publish Docker](https://github.com/xaitan80/X-Matches/actions/workflows/docker-publish.yml/badge.svg)
[![Go Tests](https://github.com/xaitan80/X-Matches/actions/workflows/go-test.yml/badge.svg)](https://github.com/xaitan80/X-Matches/actions/workflows/go-test.yml)

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
- `TRUSTED_PROXIES`: kommaseparerade CIDR/IP för proxys att lita på (default `127.0.0.1,::1`)

Exempel:

```
ADDR=127.0.0.1:9000 DB_PATH=/tmp/xmatches.db TRUSTED_PROXIES="127.0.0.1,::1" go run .
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

## Köra från Docker Hub (rekommenderat)

1) Hämta senaste bild:

```
docker pull xaitan/x-matches:latest
```

2) Sätt rättigheter på volymen en gång (icke-root runtime använder UID 65532):

```
docker run --rm -v xmatches-data:/data busybox:1.36 sh -c 'chown -R 65532:65532 /data'
```

3) Starta appen:

```
docker run --rm \
  -p 8080:8080 \
  -v xmatches-data:/data \
  -e ADDR=:8080 \
  -e DB_PATH=/data/xmatches.db \
  xaitan/x-matches:latest


  docker run --rm -p 8080:8080 -v xmatches-data:/data -e ADDR=:8080 -e DB_PATH=/data/xmatches.db xaitan/x-matches:latest
```

Öppna: http://localhost:8080

Prenumerera i kalender (iCal):

- Ladda ner: `http://localhost:8080/api/matches.ics`
- Lägg till i din kalender som fil eller via URL (om exponerad).

Snabb felsökning:

- Testa utan volym (temporär DB):

```
docker run --rm -p 8080:8080 -e DB_PATH=/tmp/xmatches.db xaitan/x-matches:latest
```

- Om du absolut vill köra som root (ej rekommenderat):

```
docker run --rm --user 0:0 -p 8080:8080 -v xmatches-data:/data -e DB_PATH=/data/xmatches.db xaitan/x-matches:latest
```

## API‑snabbguide

- Bas‑URL: `http://localhost:8080`
- UI: `GET /` (enkel webbsida)
- Lista matcher: `GET /api/matches`
- Exportera CSV: `GET /api/matches.csv` (laddar ner `matches_YYYY-MM-DD.csv`)
- Exportera iCal: `GET /api/matches.ics` (prenumerera i kalender)
- Importera: `POST /api/matches/import` (multipart med `file` – `.csv` eller `.xlsx`)
  - Valfri query: `our_team=H43%20Lund%20HF` för att sätta vilket lag som ska tolkas som "vårt" vid import (hemma/borta mappas till team/opponent utifrån detta)
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

Hälsa/Status:

```
curl http://localhost:8080/healthz
```

## Utveckling

- Formattering: `make fmt`
- Rensa databasen: stoppa appen och radera `xmatches.db` (eller byt `DB_PATH`).
- Ha kul
Import‑exempel:

CSV:

```
curl -X POST http://localhost:8080/api/matches/import \
  -F file=@matches.csv
```

XLSX:

```
curl -X POST http://localhost:8080/api/matches/import \
  -F file=@matches.xlsx

Med valt "vårt lag":

```
curl -X POST "http://localhost:8080/api/matches/import?our_team=H43%20Lund%20HF" \
  -F file=@matches_2025-09-05\ 19_50_44.csv
```

Radera alla matcher:

```
curl -X DELETE http://localhost:8080/api/matches
```
```

Stödda kolumnnamn (skiftlägesokänsliga, mellanslag/underscore ignoreras; svenska alias stöds):
- date_raw (alias: datum)
- time_raw (alias: starttid/tid), end_time_raw (alias: sluttid)
- team, opponent, home_team, away_team
- venue (alias: hall/plats), city (alias: stad)
- league (serie), court (plan)
- match_number, referees, notes (noteringar)
- played (true/false/1/ja)
- goals_for, goals_against
- player_notes
- top_scorer_team, top_scorer_opponent
- start_iso, end_iso (ISO8601)

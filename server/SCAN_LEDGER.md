# Scan ledger

What the scan feature is made of. One line per file.

## Flow

Tap → `/scan/preview` (street photos) + `/scan` (stories). Table first: lanes
only scrape a spot nobody has scanned recently; everything found is kept.

## Files

| File | Feature |
|---|---|
| `cmd/api/scan.go` | `GET /scan`: run stale lanes, read finds, group into incidents |
| `cmd/api/preview.go` | `GET /scan/preview`: street photos (table, else Mapillary → R2) |
| `cmd/api/enrich.go` | Background: article real link + photo → R2 thumbnail |
| `internal/scan/region.go` | Countries, news languages, crime words |
| `internal/scan/targets.go` | OSM features → names worth searching |
| `internal/scan/newslane.go` | Names → Google News → filtered stories |
| `internal/scan/cluster.go` | Many articles → one incident |
| `internal/scan/lexicon.go` | Malay–English word glue + anchors (RM, Op, ages) |
| `internal/scan/geohash.go` | Map squares for the sweep ledger |
| `internal/osm/overpass.go` | Police stations, streets, areas near a point |
| `internal/news/google.go` | Google News RSS search |
| `internal/news/resolve.go` | Google link → publisher link; og:image |
| `internal/mapillary/` | Street photos near a point, best-photo pick |
| `internal/blob/` | R2 upload + news thumbnails |
| `internal/store/finds.go` | Stories + police stations table |
| `internal/store/incidents.go` | Incidents table |
| `internal/store/street_images.go` | Street photos table |
| `internal/store/migrations/0006–0010` | finds, sweeps, scans, incidents, photos |
| `app/lib/api/scan_client.dart` | App calls `/scan` + `/scan/preview` |

## Env

`MAPILLARY_TOKEN`, `R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`,
`R2_BUCKET`, `R2_PUBLIC_BASE_URL` (photos are off without them).

## Rules

- Sweeps: news 24 h, police stations 30 d, street photos 90 d per square.
- We only ever show our own copies of photos.

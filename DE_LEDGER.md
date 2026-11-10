# DE ledger

Everything Claude wrote into the repo (DE). Newest first. File → what it does.

## 2026-11-10 · Scan: photos

| File | Feature |
|---|---|
| `server/cmd/api/preview.go` | `GET /scan/preview`: street photos (table, else Mapillary → R2) |
| `server/cmd/api/enrich.go` | Background: article real link + photo → R2 thumbnail |
| `server/internal/news/resolve.go` | Google link → publisher link; og:image |
| `server/internal/mapillary/` | Street photos near a point, best-photo pick |
| `server/internal/blob/` | R2 upload + news thumbnails |
| `server/internal/store/street_images.go` | Street photos table |
| `server/internal/store/migrations/0008–0010` | Photo columns + street photos table |
| `app/lib/api/scan_client.dart` | `preview()` call |

Env: `MAPILLARY_TOKEN`, `R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`,
`R2_SECRET_ACCESS_KEY`, `R2_BUCKET`, `R2_PUBLIC_BASE_URL` (photos off without them).

## 2026-11-10 · Scan: stories

Tap → `/scan`. Table first: lanes only scrape a spot nobody scanned recently
(news 24 h, police stations 30 d, street photos 90 d per map square).

| File | Feature |
|---|---|
| `server/cmd/api/scan.go` | `GET /scan`: run stale lanes, read finds, group into incidents |
| `server/internal/scan/region.go` | Countries, news languages, crime words |
| `server/internal/scan/targets.go` | OSM features → names worth searching |
| `server/internal/scan/newslane.go` | Names → Google News → filtered stories |
| `server/internal/scan/cluster.go` | Many articles → one incident |
| `server/internal/scan/lexicon.go` | Malay–English word glue + anchors (RM, Op, ages) |
| `server/internal/scan/geohash.go` | Map squares for the sweep ledger |
| `server/internal/osm/overpass.go` | Police stations, streets, areas near a point |
| `server/internal/news/google.go` | `SearchIn`: news editions per country |
| `server/internal/store/finds.go` | Stories + police stations table |
| `server/internal/store/incidents.go` | Incidents table |
| `server/internal/store/migrations/0006–0007` | finds, sweeps, scans, incidents |
| `server/internal/tiktok/` | TikTok place videos (Apify), not called yet |

## 2026-11-09 · Radar panel

| File | Feature |
|---|---|
| `app/lib/screens/map_screen.dart` | Panel tabs (`_PanelTab`), tab fade, radar-settings key fix |
| `app/lib/widgets/radar_view.dart` | Radar colour params (`gridColor`, `sweepColor`, `dotColor`) |

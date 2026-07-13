# Care — implementation plan

Care is GrowRig's **manual-action journal** for a grow: watering, feeding,
inspecting, training, trimming, transplanting, treating, flushing, harvesting,
and custom actions. It is one unified logging flow (feeding = watering + nutrients),
targeting the whole grow, a subset of plants, or a single plant.

This document is the agreed plan before any code is written. Decisions locked in:

- **Species-driven care actions from day one** — the available actions and their
  form fields come from the grow's species template (`species/<id>/species.yaml`),
  layered over GrowRig defaults. No hardcoded per-grow action list.
- A care action is a **session** (one event → many per-plant applications), so
  "mixed 5 L and fed all four plants" is one action that still records the exact
  amount each plant got.
- Care reuses the **existing Activity Log** for the human-readable timeline and
  adds structured `care_events` tables for detail, summaries and per-plant history.

---

## 1. What already exists (build on, don't duplicate)

| Concept | Where | Role in Care |
|---|---|---|
| `Grow`, `PlantUnit` (individual/group) | [`domain/grow.go`](../growcore/internal/domain/grow.go) | The care target set; plant selection. |
| `FeedingPreset` (brand nutrient charts) | [`domain/feeding.go`](../growcore/internal/domain/feeding.go) | The **nutrient recipe** referenced by a feeding event (`recipeId`). Not a logged event. |
| `Activity` + `activity_log` table | [`store.go:233`](../growcore/internal/store/store.go), [`ActivityLog.svelte`](../web/src/lib/components/ActivityLog.svelte) | The journal timeline. Care writes one grouped row per event; a new `Care` filter surfaces them. |
| Species YAML (stages, cultivar attrs) | [`species/cannabis/species.yaml`](../species/cannabis/species.yaml), [`species/species.go`](../growcore/internal/species/species.go) | Gains a `careActions:` block defining actions + fields per crop family. |
| Vertical-slice pattern | domain → `store.go` → `internal/api/*.go` → `web/src/lib/api.ts` + `types.ts` → Svelte | The path each new artifact below follows. |
| `growActivity(...)` helper | [`api/api.go:43`](../growcore/internal/api/api.go) | Reused to write the grouped care journal row. |

---

## 2. Data model

### 2.1 Care action templates (species-driven)

A **care action** declares an action key, label, icon, whether it's a quick
action, and which form fields it shows. GrowRig ships a **default set**; each
species may override or extend it in YAML. Inheritance (Phase 2 adds the last):

```
GrowRig defaults  →  species template  →  (later) grow customization
```

**Go** (`species/species.go`, extend `Species`):

```go
type CareField string // "amount" | "runoff" | "recipe" | "ph" | "ec" |
                       // "note" | "photos" | "potSize" | "product" | "trainType"

type CareAction struct {
    Key    string      `json:"key"    yaml:"key"`
    Label  string      `json:"label"  yaml:"label"`
    Icon   string      `json:"icon"   yaml:"icon,omitempty"`  // lucide name
    Fields []CareField `json:"fields" yaml:"fields"`
    Quick  bool        `json:"quick"  yaml:"quick,omitempty"` // show as a quick action
}

type Species struct {
    // ...existing...
    CareActions []CareAction `json:"careActions,omitempty" yaml:"careActions,omitempty"`
}
```

`DefaultCareActions` (a package var in `species`) is the fallback when a species
declares none, and provides the crop-neutral base:

| key | label | fields | quick |
|---|---|---|---|
| `water` | Water | amount, runoff, note | ✓ |
| `feed` | Feed | recipe, amount, ph, ec, runoff, note | ✓ |
| `inspect` | Inspect | note, photos | ✓ |
| `train` | Train | trainType, note, photos | |
| `trim` | Trim / Prune | note, photos | |
| `transplant` | Transplant | potSize, note | |
| `treat` | Spray / Treat | product, note | |
| `flush` | Flush | amount, runoff, note | |
| `harvest` | Harvest | note | |
| `custom` | Custom | note, photos | |

`species.CareActionsFor(id)` returns the species' list, or `DefaultCareActions`
if unset. The `GET /api/species` response (already consumed by the web app)
starts including `careActions`, so the client renders the action menu from it.

**YAML** (add to each `species/<id>/species.yaml`; example cannabis):

```yaml
careActions:
  - { key: water,      label: Water,         fields: [amount, runoff, note],           quick: true }
  - { key: feed,       label: Feed,          fields: [recipe, amount, ph, ec, runoff, note], quick: true }
  - { key: inspect,    label: Inspect,       fields: [note, photos],                    quick: true }
  - { key: train,      label: Train,         fields: [trainType, note, photos] }
  - { key: trim,       label: Trim / prune,  fields: [note, photos] }
  - { key: transplant, label: Transplant,    fields: [potSize, note] }
  - { key: treat,      label: Spray / treat, fields: [product, note] }
  - { key: flush,      label: Flush,         fields: [amount, runoff, note] }
  - { key: harvest,    label: Harvest,       fields: [note] }
```

Tomato/basil get their own lists (e.g. tomato swaps `train`→`stake`, adds
`pollinate`, drops `flush`). Photos in the field list are a UI affordance; image
persistence is deferred (see §7, Phase 3+).

### 2.2 Care event + applications

**Go** (`domain/care.go`, new file):

```go
type CareSource string
const ( CareManual CareSource = "manual"; CareAutomation CareSource = "automation" )

// CareEvent is one care action performed at a moment in time, targeting one or
// more plants of a grow. Solution fields apply to watering/feeding.
type CareEvent struct {
    ID         string     `json:"id"`
    GrowID     string     `json:"growId"`
    Type       string     `json:"type"`        // a CareAction key
    OccurredAt time.Time  `json:"occurredAt"`
    Source     CareSource `json:"source"`      // manual | automation
    Notes      string     `json:"notes,omitempty"`
    RecipeID   string     `json:"recipeId,omitempty"` // FeedingPreset id, feed only
    PH         float64    `json:"ph,omitempty"`
    EC         float64    `json:"ec,omitempty"`
    RunoffML   float64    `json:"runoffMl,omitempty"`
    RunoffPH   float64    `json:"runoffPh,omitempty"`
    CreatedAt  time.Time  `json:"createdAt"`

    Applications []CareApplication `json:"applications"`
}

// CareApplication is what one plant received in a care event.
type CareApplication struct {
    ID          string  `json:"id"`
    CareEventID string  `json:"careEventId"`
    PlantUnitID string  `json:"plantUnitId"`
    AmountML    float64 `json:"amountMl,omitempty"`
    Note        string  `json:"note,omitempty"`
}
```

### 2.3 Storage (`store.go` schema block + `store/care.go` methods)

```sql
CREATE TABLE IF NOT EXISTS care_events (
    id          TEXT PRIMARY KEY,
    grow_id     TEXT NOT NULL,
    type        TEXT NOT NULL,
    occurred_at INTEGER NOT NULL,
    source      TEXT NOT NULL DEFAULT 'manual',
    notes       TEXT NOT NULL DEFAULT '',
    recipe_id   TEXT NOT NULL DEFAULT '',
    ph          REAL NOT NULL DEFAULT 0,
    ec          REAL NOT NULL DEFAULT 0,
    runoff_ml   REAL NOT NULL DEFAULT 0,
    runoff_ph   REAL NOT NULL DEFAULT 0,
    created_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_care_grow_ts ON care_events (grow_id, occurred_at DESC);

CREATE TABLE IF NOT EXISTS care_applications (
    id            TEXT PRIMARY KEY,
    care_event_id TEXT NOT NULL,
    plant_unit_id TEXT NOT NULL,
    amount_ml     REAL NOT NULL DEFAULT 0,
    note          TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_care_app_event ON care_applications (care_event_id);
CREATE INDEX IF NOT EXISTS idx_care_app_plant ON care_applications (plant_unit_id);
```

Store methods (mirroring `SaveGrow`/`BulkCreatePlants`/`Activities` style):

- `SaveCareEvent(e domain.CareEvent) error` — one transaction: insert event +
  all applications.
- `CareEvents(growID string, limit, offset int) ([]domain.CareEvent, error)` —
  newest first, applications attached.
- `CareEvent(id string) (domain.CareEvent, bool, error)`.
- `LastCareByType(growID string) (map[string]domain.CareEvent, error)` — most
  recent event per action key (drives the summary's "last watered/fed/…").
- `LastCarePerPlant(growID string) (map[string]time.Time, error)` — last
  application time per plant unit (drives skipped-plant detection).
- `DeleteCareEvent(id string) error` — cascades applications.

Migration note: pure additive `CREATE TABLE IF NOT EXISTS` — no data migration,
consistent with how the other cultivation tables were introduced.

---

## 3. API (`internal/api/care.go` + routes in `api.go`)

Routes (admin-managed, matching the existing grow/plant handlers which use
`requireAdmin`):

```go
mux.HandleFunc("GET  /api/grows/{id}/care", s.requireAuth(s.getCare))
mux.HandleFunc("POST /api/grows/{id}/care", s.requireAdmin(s.logCare))
mux.HandleFunc("DELETE /api/care/{id}",     s.requireAdmin(s.deleteCare))
```

`GET /api/species` already exists — extend its payload to carry `careActions`.

### POST /api/grows/{id}/care — log a care event

Request body:

```jsonc
{
  "type": "feed",
  "occurredAt": "2026-07-13T18:30:00Z",   // optional, defaults to now
  "source": "manual",                      // optional, defaults to manual
  "notes": "slight runoff",
  "recipeId": "biobizz-lightmix-4a1c65",   // feed only, optional
  "ph": 6.2, "ec": 1.4, "runoffMl": 120, "runoffPh": 6.4,  // optional
  "amountMl": 900,                          // shorthand: same amount for every plant
  "applications": [                         // OR explicit per-plant list (wins over amountMl)
    { "plantUnitId": "oreoz-1", "amountMl": 900, "note": "" }
  ]
}
```

Handler contract:
1. Load grow (404 if missing).
2. Resolve the action: `type` must be a key in `species.CareActionsFor(grow.Species)`
   (400 otherwise).
3. Resolve target plants: explicit `applications[]` if present, else `amountMl`
   broadcast to **all active** `PlantUnits(grow.ID)`. Reject plant ids not in
   this grow (400).
4. Build `CareEvent` (server-assigned `id(...)`, `createdAt = now`), persist via
   `SaveCareEvent`.
5. Write **one grouped** journal row via `growActivity(grow.ID, "", "info",
   "care", summary)` — e.g. `💧 Watered 4 plants · 3.2 L total` /
   `🧪 Fed all plants · Veg Base · EC 1.4`. A `formatCareSummary(event, plants,
   recipeName)` helper builds the message.
6. Return the created event (applications + resolved plant labels).

### GET /api/grows/{id}/care — history + summary

```jsonc
{
  "summary": {
    "lastByType": { "water": {CareEvent}, "feed": {CareEvent} },
    "skipped": [ { "plantUnitId": "oreoz-4", "lastCareAt": "..." } ] // vs. grow's last batch
  },
  "events": [ {CareEvent, with per-application plant labels} ]
}
```

Plant-label resolution reuses the pattern in `getGrow` (map unit id → label).
The activity-log timeline stays one-line; **per-plant expansion and the Care
summary read `care_events` directly** on the grow page (no need to overload the
`activity_log` row).

---

## 4. Web (SvelteKit)

### 4.1 Types & client (`lib/types.ts`, `lib/api.ts`)

- `types.ts`: `CareType`, `CareSource`, `CareField`, `CareAction` (and add
  `careActions?: CareAction[]` to `Species`), `CareApplication`, `CareEvent`,
  `CareSummary`, `CareHistory`.
- `api.ts`:
  ```ts
  export const getCare = (growId) => json<CareHistory>(`/api/grows/${growId}/care`);
  export const logCare = (growId, body: LogCareInput) =>
      json<CareEvent>(`/api/grows/${growId}/care`, { method: 'POST', body: JSON.stringify(body) });
  export const deleteCare = (id) => req(`/api/care/${id}`, { method: 'DELETE' });
  ```

### 4.2 Components

**`LogCareModal.svelte`** — the one shared flow (three steps), built on the
existing `ui/Dialog`, `Select`, `Button`, `Switch`, `Slider`:

- Props: `grow`, `plants: PlantDetail[]`, `careActions: CareAction[]`,
  `preselectedPlantIds?: string[]`.
- **Step 1 — Action:** grid of large buttons from `careActions` (quick ones
  first, then "More").
- **Step 2 — Plants:** multiselect, **default = all active plants**. When opened
  from a plant row, that plant is preselected; user can still change it. → the
  grow-level and plant-level entry points are the *same* component.
- **Step 3 — Details:** render only the fields in `action.fields`.
  - `amount`: one "same amount for all" number + optional per-plant override.
  - `recipe`: `Select` populated from `getFeedingPresets()` + `getFeedingTemplates(species)`.
  - `ph`/`ec`/`runoff`/`note`/`potSize`/`product`/`trainType`: contextual inputs.
- Submit label: `Log care for N plants` → `logCare(...)`, then emit a change
  event so the grow page + activity log refresh.

**`CareSummary.svelte`** — compact section placed on the grow detail page between
the plants list and the Activity Log:

```
CARE
💧 Last watered 2d ago   🧪 Last fed 5d ago   ✂️ Last trim 12d ago
⚠️ Oreoz #4 last watered 3d ago (skipped in the last batch)
[ Water all ]  [ Feed all ]  [ Log care ]
```

Driven by `getCare(growId).summary`. "Water all"/"Feed all" open `LogCareModal`
prefilled to that action with all plants selected.

### 4.3 Wiring into existing pages

- **Grow detail** ([`routes/grows/[id]/+page.svelte`](../web/src/routes/grows/[id]/+page.svelte)):
  add `+ Log care` to the `PLANTS · N ACTIVE` header, mount `CareSummary`, and add
  a per-row care action opening `LogCareModal` with that plant preselected.
- **Activity Log** ([`ActivityLog.svelte`](../web/src/lib/components/ActivityLog.svelte)):
  add `Care` to the filter set (maps to `type = "care"`), and give care types
  their icons (💧 water, 🧪 feed, ✂️ trim, 🔍 inspect, …) alongside the existing
  `Sprout`/`Box` source icons.
- **Environment / GrowBox page** (later): a compact "Plant care" card with
  `Water` / `Feed` shortcuts that reuse `LogCareModal`.

---

## 5. Phasing

**Phase 1 — Care logging MVP** *(the core deliverable)*
- `species.CareAction` + `DefaultCareActions` + `careActions:` YAML for the three
  species; `careActions` on the species API payload.
- `domain/care.go`, `care_events`/`care_applications` tables + `store/care.go`.
- `api/care.go`: `GET`/`POST /api/grows/{id}/care`, `DELETE /api/care/{id}`;
  grouped journal row.
- Web: `LogCareModal`, `+ Log care` on the grow page, per-plant care action,
  `Care` filter in the Activity Log.

**Phase 2 — Grow-level care customization**
- Per-grow enable/disable, reorder, rename, add custom actions, configure default
  fields; stage-sensitive ordering (e.g. `train`/`trim` up during veg). Persist a
  per-grow care config (new `grow_care_config` table or a JSON column on `grows`).

**Phase 3 — Presets & richer summaries**
- Lightweight **care preset** = recipe + target volume + pH/EC target + stage
  (extend `FeedingPreset` or a small `care_presets` table). Photo attachments for
  inspect/trim. Sharper skipped-plant and per-plant last-action detection.

**Phase 4 — Planning & reminders**
- `planned` flag on care events → an "Upcoming care" area; overdue indicators;
  recurring inspections/treatments.

**Phase 5 — Automation-sourced care**
- Auto-log completed irrigation/dosing with `source: "automation"`; user confirm/
  correct; `Manual` vs `Automated` badges (`source` already in the model).

---

## 6. Testing

- `store/care_test.go` — save/read round-trip, application cascade on delete,
  `LastCareByType` / `LastCarePerPlant` (mirrors `store_test.go`, `grow_test.go`).
- `species` — `CareActionsFor` falls back to defaults; YAML parses.
- `api` — POST validates unknown `type`, rejects foreign plant ids, broadcasts
  `amountMl` to active plants, writes exactly one activity row.
- Manual: log water/feed for all plants and one plant, confirm the grouped
  Activity Log entry, the Care summary, and the `Care` filter (use dev ports
  8791/5250 per project memory).

---

## 7. Open questions

1. **Groups** (`TrackGroup`, quantity > 1): does `amountMl` mean per-plant or
   per-group total? Proposed: `amountML` is **per application row** (i.e. per
   unit), and the UI shows "× quantity" context for groups.
2. **Photos** for inspect/trim — store as BLOBs like cultivar images, or defer to
   Phase 3? Proposed: field appears in Phase 1 UI but persistence lands in Phase 3.
3. **Access control** — writes as `requireAdmin` (like grows/plants) or
   `requireEnvWrite` so a write-access user can log care? Proposed: start with
   `requireAdmin` for consistency, revisit in Phase 2.
4. **Journal ↔ detail link** — keep the grouped activity row purely textual
   (grow page reads `care_events` for detail), or add a `care_event_id` column to
   `activity_log` for direct expansion in the global log? Proposed: textual for
   Phase 1.

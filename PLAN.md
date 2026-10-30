# Shopping List Sort — Plan

Goal: given a shopping list (typed, or a photo of one), output the items in
the order you walk past them in a specific supermarket — without ever having
been given the supermarket's floor plan.

## 1. The core problem

We don't know the layout. So we **learn** it from cheap inputs the user
produces anyway, and refine it every time they shop.

## 2. Inputs that build a layout

| Input | What it gives us | Strength |
|-------|------------------|----------|
| **Aisle-sign photos**, taken in walking order | OCR → ordered list of section names (e.g. "Bread", "Dairy") | Strong, one-off bootstrap |
| **Manual section editor** | User drags sections into order | Strong, explicit |
| **Shopping trips** (check items off in the app while shopping) | The *order of check-offs* is an observed walk through the store | Weak per trip, strong in aggregate |
| **Self-scan receipts** (hand scanner) | Receipt lines are in scan = pick order → a trip for free. Each line has a dot to exclude "went back for it" items | Strong, effortless |
| **Shopping-list photo** | OCR → list of items (input to sorting, not layout) | — |
| **Item → section assignment** | Each item learned to belong to a section; seeded by a keyword dictionary | Lets unseen items be placed |

## 2b. Making the list: lists, templates, statistics

- **Build a list**: type items (autocomplete from everything you've bought
  before) with a quantity stepper, or scan a handwritten list (OCR → one item
  per line).
- **Templates**: save any list as a template ("Weekly"). Each week, "New list"
  copies it; you tweak that copy for the week. The template stays untouched.
- **Completing a list** (finish a Shop trip, or "Mark as bought") stores what
  was *actually* bought. That history drives statistics:
  - per item: % of lists it was bought in, average quantity, last bought;
  - per template, over the last N lists:
    - **add**: not on the template but bought in ≥50% of lists (≥2 times);
    - **adjust quantity**: usual quantity differs from the template's;
    - **remove**: on the template but bought in ≤25% of lists (after ≥4 lists).
  Each suggestion is one tap to apply to the template.

## 3. Data model

```
Store      { id, name, sections[] (ordered), itemSection{item→section},
             pairs{ "a|b" → count of trips where a was picked before b } }
Trip       { storeId, items[] (ordered as checked off) }
List       { id, name, items[{name, qty}], templateId?, createdAt, completedAt? }
Template   { id, name, items[{name, qty}] }
Category   built-in keyword → section-name dictionary (fallback)
```

Persistence: a single JSON file (`data.json`) — trivial to back up, enough
for a personal app. Swappable behind the `store.Repository` interface.

## 4. Generating a layout (the algorithm)

1. **Pairwise precedence.** Every trip of N items adds N·(N−1)/2
   observations `a before b`. These accumulate in `pairs`.
2. **Item ranking.** For an item set, score each item with a Copeland/Borda
   style score: `score(a) = Σ_b P(a before b) − P(b before a)`, where P is
   the smoothed fraction of trips that saw that order. Sort by score. This
   is robust to noisy trips (going back for forgotten milk).
3. **Section order.** Sections get the mean rank of their items across
   trips, blended with any explicit order (photo/manual), which wins ties
   and acts as a prior.
4. `GET /api/stores/{id}/layout` returns the generated section order plus
   the learned item order — so the user can inspect and correct it.

## 5. Sorting a list

For each list item, in priority order:
1. Item seen in trips → use its learned rank (pairwise score vs. the other
   list items).
2. Item has a known section (learned, or keyword dictionary) → section's
   position.
3. Unknown → end of list, under "Unsorted"; when the user checks it off on
   a trip, it gets learned.

Items are returned grouped by section.

## 6. Architecture

```
frontend/ (Vue 3 + Vite)        backend/ (Go, stdlib only)
  Stores view                      /api/stores            CRUD
  Layout editor (drag sections)    /api/stores/{id}/layout  generate layout
  Sort view (text or photo)        /api/stores/{id}/sort    sort a list
  Shop mode (check-off = trip)     /api/stores/{id}/trips   record a trip
  List builder + templates         /api/lists, /api/templates   CRUD
                                   /api/lists/{id}/complete, /save-as-template
                                   /api/templates/{id}/new-list
  Stats view                       /api/stats?templateId=&window=
                                   /api/receipt/parse     receipt text → items
                                   /api/ocr               image → lines
```

OCR: backend shells out to `tesseract` if installed (`apt install
tesseract-ocr`); otherwise `/api/ocr` returns 501 and the UI falls back
to text input. Aisle-sign photos are OCR'd one by one in order; each
photo's first line becomes the next section.

## 7. Milestones

1. ✅ Backend: model, JSON store, ranking, sort, trips, OCR endpoint, tests.
2. ✅ Receipt input: OCR/paste → item lines (prices, totals, payment, headers
   stripped) → user taps the dot on out-of-order items → recorded as a trip.
   Excluded items add no pairs at all, so a go-back can't corrupt the order.
3. ✅ Frontend: stores, layout editor, sort, shop mode with trip recording.
4. ✅ Lists, templates and buying statistics with template suggestions.
5. Next: multi-store per user/auth; Vision-LLM OCR for handwriting; per-store
   item synonyms ("melk" = "milk"); export sorted list to share.

## 8. Running

```
cd backend && go run ./cmd/server          # :8080, data in ./data.json
cd frontend && npm install && npm run dev  # :5173, proxies /api → :8080
```

---

# Roadmap v2 (decisions from review, 30 Oct 2026)

## Decisions

| Topic | Decision |
|-------|----------|
| Users | Sign-in required. **Google Sign-In** (OpenID Connect ID token, verified by the backend). Android/Apple sign-in later. |
| Ownership | Lists, templates, stats and trips belong to a user. |
| Supermarket layouts | **Public by default, forked on edit.** A store's layout is shared; the first user to create it publishes it. When a user changes the section order (their route differs), they get a **personal fork**; everyone else keeps the shared one. *(Open: see question A.)* |
| Receipts | A scanned receipt **means bought**: it records the trip *and* completes the list for stats. Each line gets a second toggle: **"didn't buy this"** (excluded from stats and ordering), next to the existing "don't use for ordering" dot. |
| Matching names | **Keyword match, no product catalogue.** A receipt line matches a list item if the line contains the item's word (`AH HALFV MELK 1L` ⊇ `melk`). Longest keyword wins; no match → item added as-is. Good enough, YOLO. |

## Status

- ✅ 1 Auth & ownership · ✅ 2 Voted shared layouts with forks · ✅ 3 Receipt = bought (PDF via `pdftotext`, photo via `tesseract`, asks which list) · ✅ 5 Dockerfile
- ⏳ 4 SQLite: deferred — the JSON file is fine for a personal tool.
- Voting: each user casts **one vote per item pair** (their own majority), so one user's
  nonsense counts once however many trips they record. Forks learn only from their owner.

## Steps

1. **Auth & ownership**: Google Sign-In in the frontend, `Authorization: Bearer <id token>` → backend verifies against Google's keys, issues a session cookie; `ownerId` on lists/templates/trips.
2. **Shared layouts with forks**: `Store.ownerId = ""` (public) or a user id (fork). Reading a layout prefers the user's fork, else the public one. Trips from all users feed the public layout; a fork learns only from its owner's trips.
3. **Receipt = bought**: receipt upload → pick the open list (or create one) → keyword match → trip + complete list. Two toggles per line.
4. **Storage**: move JSON file → SQLite (single file, still easy to back up) once there are multiple users.
5. **Deploy**: one container serving API + built frontend over HTTPS (needed for Google Sign-In and camera access).

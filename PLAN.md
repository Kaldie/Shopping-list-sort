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
| **Shopping-list photo** | OCR → list of items (input to sorting, not layout) | — |
| **Item → section assignment** | Each item learned to belong to a section; seeded by a keyword dictionary | Lets unseen items be placed |

## 3. Data model

```
Store      { id, name, sections[] (ordered), itemSection{item→section},
             pairs{ "a|b" → count of trips where a was picked before b } }
Trip       { storeId, items[] (ordered as checked off) }
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
                                   /api/ocr               image → lines
```

OCR: backend shells out to `tesseract` if installed (`apt install
tesseract-ocr`); otherwise `/api/ocr` returns 501 and the UI falls back
to text input. Aisle-sign photos are OCR'd one by one in order; each
photo's first line becomes the next section.

## 7. Milestones

1. ✅ Backend: model, JSON store, ranking, sort, trips, OCR endpoint, tests.
2. ✅ Frontend: stores, layout editor, sort, shop mode with trip recording.
3. Next: multi-store per user/auth; Vision-LLM OCR for handwriting; per-store
   item synonyms ("melk" = "milk"); export sorted list to share.

## 8. Running

```
cd backend && go run ./cmd/server          # :8080, data in ./data.json
cd frontend && npm install && npm run dev  # :5173, proxies /api → :8080
```

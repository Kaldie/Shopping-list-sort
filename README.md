# Shopping List Sort

Sorts your shopping list into walking order for a supermarket whose layout
you don't know up front — the layout is learned from aisle-sign photos,
a manual section list, and the order you tick items off while shopping.

See [PLAN.md](PLAN.md) for the design and algorithm.

```
cd backend && go run ./cmd/server          # http://localhost:8080
cd frontend && npm install && npm run dev  # http://localhost:5173
```

Photo input (shopping list / aisle signs) needs `tesseract-ocr` on the server.

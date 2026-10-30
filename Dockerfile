# Build the frontend
FROM node:22-alpine AS web
WORKDIR /web
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Build the backend
FROM golang:1.24-alpine AS api
WORKDIR /src
COPY backend/go.mod ./
COPY backend/ ./
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

# Runtime: OCR for receipt photos (tesseract) and digital receipts (pdftotext)
FROM alpine:3.20
RUN apk add --no-cache tesseract-ocr tesseract-ocr-data-nld poppler-utils ca-certificates
COPY --from=api /server /server
COPY --from=web /web/dist /web
VOLUME /data
EXPOSE 8080
ENV SECURE_COOKIES=true
ENTRYPOINT ["/server", "-addr", ":8080", "-data", "/data/data.json", "-static", "/web"]

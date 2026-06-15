# --- Étape build ---
FROM golang:1.26-alpine AS build
WORKDIR /src

# Cache des dépendances
COPY go.mod go.sum ./
RUN go mod download

# Code + compilation statique
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /opale ./cmd/api

# --- Étape run ---
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /opale /opale
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/opale"]

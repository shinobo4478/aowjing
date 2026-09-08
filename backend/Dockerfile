# Backend image — builds both binaries (cmd/api and cmd/worker) from the
# shared module. The two compose services run the same image with a
# different entrypoint.

FROM golang:1.24-alpine AS build
WORKDIR /src

# Cache deps: only re-download when go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO off → a fully static binary that runs on distroless/static.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api    ./cmd/api
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

# distroless/static ships ca-certificates + tzdata and nothing else — no shell,
# no package manager. The worker's HTTPS calls to fal.ai need the CA bundle.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api    /usr/local/bin/api
COPY --from=build /out/worker /usr/local/bin/worker
USER nonroot:nonroot
EXPOSE 8080
# Default to the API; the worker service overrides `entrypoint` in compose.
ENTRYPOINT ["/usr/local/bin/api"]

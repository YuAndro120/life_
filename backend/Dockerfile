# syntax=docker/dockerfile:1
# Сборка на Mac кросс-компиляцией под linux/amd64 (сервер: Ubuntu x86_64): docker buildx build --platform linux/amd64 --target api .
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
WORKDIR /src
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && go build -trimpath -ldflags="-s -w" -o /out/collector ./cmd/collector \
 && go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed
# goose для миграций (та же версия, что и на разработке)
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.28.0 && cp /go/bin/${TARGETOS}_${TARGETARCH}/goose /out/goose 2>/dev/null || cp /go/bin/goose /out/goose

FROM gcr.io/distroless/static-debian12:nonroot AS api
COPY --from=build /out/api /app/api
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/api"]

FROM gcr.io/distroless/static-debian12:nonroot AS collector
WORKDIR /app
COPY --from=build /out/collector /app/collector
COPY seeds/sources.yaml /app/seeds/sources.yaml
USER nonroot:nonroot
ENTRYPOINT ["/app/collector"]

FROM gcr.io/distroless/static-debian12:nonroot AS migrate
COPY --from=build /out/goose /app/goose
COPY internal/db/migrations /migrations
USER nonroot:nonroot
ENV GOOSE_DRIVER=postgres
ENTRYPOINT ["/app/goose", "-dir", "/migrations"]
CMD ["up"]

# Только для ручного запуска на dev/demo (очищает таблицы данных); в постоянные сервисы не входит.
FROM gcr.io/distroless/static-debian12:nonroot AS seed
WORKDIR /app
COPY --from=build /out/seed /app/seed
COPY seeds/dev.json /app/seeds/dev.json
USER nonroot:nonroot
ENTRYPOINT ["/app/seed"]

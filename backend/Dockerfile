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
 && go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker \
 && go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed \
 && go build -trimpath -ldflags="-s -w" -o /out/lawtool ./cmd/lawtool
# goose для миграций (та же версия, что и на разработке)
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.28.0 && cp /go/bin/${TARGETOS}_${TARGETARCH}/goose /out/goose 2>/dev/null || cp /go/bin/goose /out/goose

FROM gcr.io/distroless/static-debian12:nonroot AS api
COPY --from=build /out/api /app/api
# Веб-приложение (PWA) раздаёт тот же сервис. Каталог web/ лежит вне контекста сборки и подключается именованным контекстом:
#   docker buildx build --build-context web=../web --target api …
COPY --from=web . /app/web
ENV WEB_DIR=/app/web
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/api"]

FROM gcr.io/distroless/static-debian12:nonroot AS collector
WORKDIR /app
COPY --from=build /out/collector /app/collector
COPY seeds/sources.yaml /app/seeds/sources.yaml
USER nonroot:nonroot
ENTRYPOINT ["/app/collector"]

FROM gcr.io/distroless/static-debian12:nonroot AS worker
COPY --from=build /out/worker /app/worker
USER nonroot:nonroot
ENTRYPOINT ["/app/worker"]

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

# Юридический конвейер: запускается вручную (fetch, review), в постоянные сервисы не входит.
FROM gcr.io/distroless/static-debian12:nonroot AS lawtool
COPY --from=build /out/lawtool /app/lawtool
USER nonroot:nonroot
ENTRYPOINT ["/app/lawtool"]

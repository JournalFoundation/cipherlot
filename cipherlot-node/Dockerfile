# build
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/node ./cmd/node

# run (distroless base for smaller image)
FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/node /app/node
EXPOSE 8080
VOLUME ["/data"]
ENV DATA_ROOT=/data
ENTRYPOINT ["/app/node"]

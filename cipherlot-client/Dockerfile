# build
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/clt ./cmd/clt

# run
FROM gcr.io/distroless/base-debian12
COPY --from=build /out/clt /usr/local/bin/clt
ENTRYPOINT ["/usr/local/bin/clt"]

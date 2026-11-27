# GitaiHub API and demo seeder. Both are static Go binaries.
FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/...

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/gitaihub-api /out/gitaihub-seed /usr/local/bin/
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/gitaihub-api"]

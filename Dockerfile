FROM golang:1.26 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /progames ./cmd/progames

FROM scratch
COPY --from=builder /progames /progames

ENV PROGAMES_ADDR=:8080 \
    PROGAMES_DB=/data/progames.db \
    PROGAMES_ARTIFACTS=/data/artifacts

EXPOSE 8080
VOLUME ["/data"]
CMD ["/progames"]

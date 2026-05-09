FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
    -o /out/smtp-dev-server ./cmd/smtp-dev-server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/smtp-dev-server /usr/local/bin/smtp-dev-server
EXPOSE 2525 5050
ENTRYPOINT ["/usr/local/bin/smtp-dev-server"]
CMD ["-smtp", "0.0.0.0:2525", "-http", "0.0.0.0:5050"]

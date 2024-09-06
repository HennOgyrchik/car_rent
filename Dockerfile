FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY . .
RUN  mkdir /app
RUN go build -o /app/car_rent ./cmd/

FROM alpine:latest
COPY --from=builder /app /app
COPY --from=builder /build/pkg/common/db/migration /app/migration
COPY --from=builder /build/startup.sh /app
RUN chmod +x /app/startup.sh
CMD ["/app/startup.sh"]
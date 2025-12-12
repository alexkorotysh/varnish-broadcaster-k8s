FROM golang:1.23-alpine AS build
WORKDIR /src
COPY . .
RUN go mod init broadcaster || true
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /broadcaster main.go

FROM alpine:3.20
RUN adduser -D -H broadcaster
USER broadcaster
COPY --from=build /broadcaster /usr/local/bin/broadcaster
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/broadcaster"]
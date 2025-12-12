FROM golang:1.22-alpine AS build
WORKDIR /src
COPY . .
RUN apk add --no-cache git build-base ca-certificates
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/server ./cmd/api

FROM scratch
COPY --from=build /bin/server /bin/server
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
EXPOSE 8080
ENTRYPOINT ["/bin/server"]

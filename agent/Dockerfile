FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /eits-agent ./cmd/eits-agent
FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /eits-agent /usr/local/bin/eits-agent
ENTRYPOINT ["/usr/local/bin/eits-agent"]

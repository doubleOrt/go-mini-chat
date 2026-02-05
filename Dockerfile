FROM golang:1.25.3 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags="-s -w" -o /out/app .

FROM gcr.io/distroless/static:nonroot
WORKDIR /app

COPY --from=build /out/app /app/app
COPY --from=build /src/public /app/public

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/app"]

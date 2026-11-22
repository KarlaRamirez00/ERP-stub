FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/rex-stub ./cmd

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/rex-stub /rex-stub
EXPOSE 8090
ENTRYPOINT ["/rex-stub"]

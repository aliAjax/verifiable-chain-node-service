FROM golang:1.22 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/node ./cmd/node
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/node /node
EXPOSE 8080
ENTRYPOINT ["/node"]

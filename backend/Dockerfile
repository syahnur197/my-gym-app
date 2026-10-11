FROM golang:1.23 AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN go mod download && CGO_ENABLED=0 go build -o /gym-server .
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /gym-server /gym-server
WORKDIR /data
ENV DATABASE_PATH=/data/gym.db PORT=8080
EXPOSE 8080
ENTRYPOINT ["/gym-server"]

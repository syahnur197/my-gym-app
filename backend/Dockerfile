FROM golang:1.23 AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /gym-server .
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /gym-server /gym-server
WORKDIR /data
ENV DATABASE_PATH=/data/gym.db PORT=8080
EXPOSE 8080
ENTRYPOINT ["/gym-server"]

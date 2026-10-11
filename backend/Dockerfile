FROM golang:1.23-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -trimpath -o /gym-server .

FROM debian:bookworm-slim
RUN useradd --system --uid 10001 --create-home gym && mkdir -p /data && chown gym:gym /data
COPY --from=build /gym-server /usr/local/bin/gym-server
USER gym
WORKDIR /data
ENV DATABASE_PATH=/data/gym.db PORT=8080
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/gym-server"]

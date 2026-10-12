FROM golang:1.23-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -trimpath -o /gym-server .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install --no-install-recommends -y curl \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 10001 --create-home gym \
    && mkdir -p /data && chown gym:gym /data
COPY --from=build /gym-server /usr/local/bin/gym-server
USER gym
WORKDIR /data
ENV DATABASE_PATH=/data/gym.db PORT=8080
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD curl --fail --silent http://127.0.0.1:8080/health || exit 1
ENTRYPOINT ["/usr/local/bin/gym-server"]

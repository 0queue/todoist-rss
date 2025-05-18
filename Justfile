run:
    TOKEN=$(cat .token) go run ./cmd/todoist-rss

minifluxdb:
    podman run --rm -d --name minfluxdb -e POSTGRES_USER=miniflux -e POSTGRES_PASSWORD=miniflux -e POSTGRES_DB=miniflux -p 5432:5432 postgres

miniflux:
    podman run --rm -d --name miniflux \
        -e DATABASE_URL=postgres://miniflux:miniflux@localhost:5432/miniflux?sslmode=disable \
        -e RUN_MIGRATIONS=1 \
        -e CREATE_ADMIN=1 \
        -e LISTEN_ADDR=:8081 \
        -e ADMIN_USERNAME=admin \
        -e ADMIN_PASSWORD=password \
        --network host \
        miniflux/miniflux:latest

export KO_DOCKER_REPO := "ghcr.io/0queue/todoist-rss"

publish:
    go run github.com/google/ko@v0.18.0 build --bare --tags=v0.1.0 ./cmd/todoist-rss

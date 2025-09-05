IMAGE ?= xmatches:local
PLATFORM ?= linux/arm64

docker-build:
	@docker buildx build --platform $(PLATFORM) -t $(IMAGE) .

docker-run:
	@docker run --rm -p 8080:8080 -v xmatches-data:/data $(IMAGE)

compose-up:
	@PLATFORM=$(PLATFORM) docker compose up --build

compose-down:
	@docker compose down

# Snabb formattering (valfritt)
fmt:
	@gofmt -s -w .
	@command -v goimports >/dev/null 2>&1 && goimports -w . || true

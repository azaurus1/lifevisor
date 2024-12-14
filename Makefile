LIFEVISOR_BACKEND_BINARY=lifeVisorService

up:
	@echo "Starting Docker images..."
	docker compose up
	@echo "Docker images started!"

up_build: build_backend
	@echo "Stopping docker images if running ..."
	docker compose down
	@echo "Building (when required) and starting docker images..."
	docker compose up --build
	@echo "Docker images built and started!"

down:
	@echo "Stopping docker compose..."
	docker compose down
	@echo "Done!"

build_backend:
	@echo "Building backend binary.."
	cd ./service && env GOOS=linux CGO_ENABLED=0 go build -o ${LIFEVISOR_BACKEND_BINARY} ./cmd
	@echo "Done!"

.PHONY: infra backend web mobile clean

# Run infrastructure (PostgreSQL, Redis, NATS)
infra:
	docker-compose up -d

# Run everything (infrastructure first)
all: infra backend

# Run Backend
backend:
	cd apps/backend && go run .

# Run Web App
web:
	cd apps/web && npm run dev

# Run Mobile App
mobile:
	cd apps/mobile && npx expo start

# Stop infrastructure
stop:
	docker-compose down

# Clean up
clean:
	docker-compose down -v

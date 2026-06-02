# .env ni include qilmaymiz — Make sintaksisini buzadi (URL, ssh qatorlari va h.k.)
MIGRATIONS_PATH = ./cmd/migrate/migrations

# --- DB: eski server (47.236.63.18) -> Mac -> yangi server (mubashshir) ---
OLD_HOST        ?= root@47.236.63.18
OLD_PATH        ?= /root/CurrencyExchange/deploy/server
NEW_HOST        ?= swe
NEW_PATH        ?= /app/backend/currency-exchange
LOCAL_BACKUPS   ?= ./backups

.PHONY: migrate-create
migration:
	@migrate create -seq -ext sql -dir ${MIGRATIONS_PATH} ${filter-out $@,${MAKECMDGOALS}}

.PHONY: migrate-up
migrate-up:
	@set -a && [ -f .env ] && . ./.env && set +a; \
	migrate -path=$(MIGRATIONS_PATH) -database="$$DB_ADDR" up

.PHONY: migrate-down
migrate-down:
	@set -a && [ -f .env ] && . ./.env && set +a; \
	migrate -path=$(MIGRATIONS_PATH) -database="$$DB_ADDR" down $(filter-out $@,$(MAKECMDGOALS))

.PHONY: seed
seed:
	@set -a && [ -f .env ] && . ./.env && set +a; \
	go run cmd/migrate/seed/main.go

.PHONY: db-help db-backup db-push db-restore db-migrate db-sync-scripts

db-help:
	@echo "DB migratsiya (eski -> Mac -> yangi)"
	@echo ""
	@echo "  make db-backup        Eski serverda backup + Mac'ga yuklash"
	@echo "  make db-push          Local dump -> yangi server"
	@echo "  make db-restore       Yangi serverda restore (avval db-push)"
	@echo "  make db-sync-scripts  Yangi serverga scripts/ sync"
	@echo "  make db-migrate       Hammasi ketma-ket"
	@echo ""
	@echo "  DUMP=backups/db-....sql.gz   aniq fayl (ixtiyoriy)"
	@echo "  OLD_HOST=$(OLD_HOST)"
	@echo "  NEW_HOST=$(NEW_HOST)"

db-backup:
	@mkdir -p "$(LOCAL_BACKUPS)"
	@echo ">> Backup: $(OLD_HOST):$(OLD_PATH)"
	ssh "$(OLD_HOST)" 'cd "$(OLD_PATH)" && ./scripts/backup-db.sh'
	@echo ">> Mac'ga yuklanmoqda..."
	@REMOTE=$$(ssh "$(OLD_HOST)" 'ls -t "$(OLD_PATH)"/backups/db-*.sql.gz 2>/dev/null | head -1'); \
	test -n "$$REMOTE" || { echo "Xato: serverda dump topilmadi"; exit 1; }; \
	scp "$(OLD_HOST):$$REMOTE" "$(LOCAL_BACKUPS)/"; \
	ls -lh "$(LOCAL_BACKUPS)/"

db-sync-scripts:
	@echo ">> restore.sh + scripts -> $(NEW_HOST):$(NEW_PATH)"
	rsync -avz deploy/server/scripts/ "$(NEW_HOST):/tmp/ce-db-scripts/"
	rsync -avz deploy/server/restore.sh "$(NEW_HOST):/tmp/ce-restore.sh"
	ssh "$(NEW_HOST)" 'sudo mkdir -p "$(NEW_PATH)/scripts/lib" && \
		sudo cp -a /tmp/ce-db-scripts/. "$(NEW_PATH)/scripts/" && \
		sudo cp /tmp/ce-restore.sh "$(NEW_PATH)/restore.sh" && \
		sudo chmod +x "$(NEW_PATH)/restore.sh" "$(NEW_PATH)/scripts/"*.sh "$(NEW_PATH)/scripts/lib/"*.sh 2>/dev/null || true'

db-push:
	@set -e; \
	DUMP="$${DUMP:-$$(ls -t "$(LOCAL_BACKUPS)"/db-*.sql.gz 2>/dev/null | head -1)}"; \
	test -f "$$DUMP" || { echo "Xato: dump yo'q. Avval: make db-backup"; exit 1; }; \
	F=$$(basename "$$DUMP"); \
	echo ">> $$F -> $(NEW_HOST):$(NEW_PATH)/backups/"; \
	scp "$$DUMP" "$(NEW_HOST):/tmp/$$F"; \
	ssh "$(NEW_HOST)" "sudo mkdir -p '$(NEW_PATH)/backups' && \
		sudo mv /tmp/$$F '$(NEW_PATH)/backups/$$F'"

db-restore:
	@set -e; \
	DUMP="$${DUMP:-$$(ls -t "$(LOCAL_BACKUPS)"/db-*.sql.gz 2>/dev/null | head -1)}"; \
	test -f "$$DUMP" || { echo "Xato: local dump yo'q"; exit 1; }; \
	F=$$(basename "$$DUMP"); \
	echo ">> Restore on $(NEW_HOST): $$F"; \
	ssh "$(NEW_HOST)" "sudo bash -lc 'cd \"$(NEW_PATH)\" && ./restore.sh --fresh backups/$$F'"

db-migrate: db-backup db-sync-scripts db-push db-restore
	@echo ">> Tayyor."
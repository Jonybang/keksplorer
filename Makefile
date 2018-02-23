ENV_VARS := `cat .env`

worker-dev:
	export $(ENV_VARS) && node worker/worker.js

dev-compose-up:
	docker-compose -f dev-stack.yml up

dev-compose-down:
	docker-compose -f dev-stack.yml down

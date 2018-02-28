ENV_VARS := `cat .env`

worker-dev:
	export $(ENV_VARS) && node worker/worker.js

agent-dev:
	export $(ENV_VARS) && node agent/agent.js

dev-compose-up:
	docker-compose -f dev-stack.yml up

dev-compose-down:
	docker-compose -f dev-stack.yml down

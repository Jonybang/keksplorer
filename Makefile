ENV_VARS := `cat .env`
DOCKER_PUSH_MSG = "Enter version using samever convention (like 1.5.2) or just 'latest':"

worker-dev:
	export $(ENV_VARS) && node worker/worker.js

agent-dev:
	export $(ENV_VARS) && node agent/agent.js

dev-compose-up:
	docker-compose -f dev-stack.yml up

dev-compose-down:
	docker-compose -f dev-stack.yml down

docker-build-worker:
	cd worker && npm install && docker build -t keksplorer-worker .

docker-build-agent:
	cd agent && npm install && docker build -t keksplorer-agent .

docker-build-web:
	cd web && npm install && docker build -t keksplorer-web .

docker-push-worker:
	@read -p $(DOCKER_PUSH_MSG) version; \
	docker tag keksplorer-worker:latest chebykin/keksplorer-worker:$$version; \
	docker push chebykin/keksplorer-worker:$$version

docker-push-agent:
	@read -p $(DOCKER_PUSH_MSG) version; \
	docker tag keksplorer-agent:latest chebykin/keksplorer-agent:$$version; \
	docker push chebykin/keksplorer-agent:$$version

docker-push-web:
	@read -p $(DOCKER_PUSH_MSG) version; \
	docker tag keksplorer-web:latest chebykin/keksplorer-web:$$version; \
	docker push chebykin/keksplorer-web:$$version

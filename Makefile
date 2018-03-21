ENV_VARS := `cat .env`
PUSH_MSG = "Enter version using samever convention (like 1.5.2) or just 'latest':"

clean:
	rm -rf ./build

worker-dev:
	export $(ENV_VARS) && node worker/worker.js

agent-dev:
	export $(ENV_VARS) && node agent/agent.js

web-dev:
	export REDIS_URL=redis://127.0.0.1:6379 && cd web && go run server.go models.go

up:
	docker-compose -f prod-stack.yml up

down:
	docker-compose -f prod-stack.yml down --volumes

pull:
	docker-compose -f prod-stack.yml pull

up-daemon:
	docker-compose -f prod-stack.yml up -d

down-daemon:
	docker-compose -f prod-stack.yml stop

up-dev:
	docker-compose -f dev-stack.yml up

down-dev:
	docker-compose -f dev-stack.yml down

install-deps:
	cd agent && npm i && cd ../ && \
	cd worker && npm i && cd ../ && \
	cd web && dep ensure

docker-build-worker:
	cd worker && docker build -t keksplorer-worker .

docker-build-agent:
	cd agent && docker build -t keksplorer-agent .

docker-build-web: clean
	mkdir -p ./build/gopath/src/web
	cp -r ./web/* ./build/gopath/src/web
	export GOPATH=`pwd`/build/gopath; \
	cd ./build/gopath/src/web; \
	dep ensure; \
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server .; \
	docker build -t keksplorer-web .

docker-push-worker:
	@read -p $(PUSH_MSG) version; \
	docker tag keksplorer-worker:latest kenigtech/keksplorer-worker:$$version; \
	docker push kenigtech/keksplorer-worker:$$version

docker-push-agent:
	@read -p $(PUSH_MSG) version; \
	docker tag keksplorer-agent:latest kenigtech/keksplorer-agent:$$version; \
	docker push kenigtech/keksplorer-agent:$$version

docker-push-web:
	@read -p $(PUSH_MSG) version; \
	docker tag keksplorer-web:latest kenigtech/keksplorer-web:$$version; \
	docker push kenigtech/keksplorer-web:$$version

chain-poa-up:
	cd chains/configs/poa && parity --config node.toml

chain-sokol-up:
	cd chains/configs/sokol && parity --config node.toml

chain-kenig54-up:
	cd chains/configs/kenig54 && parity --config node.toml

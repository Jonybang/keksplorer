ENV_VARS := `cat .env`
PUSH_MSG = "Enter version using samever convention (like 1.5.2) or just 'latest':"

clean:
	rm -rf ./build

worker-dev:
	export $(ENV_VARS) && node worker/worker.js

agent-dev:
	export $(ENV_VARS) && node agent/agent.js

up:
	docker-compose -f prod-stack.yml up

down:
	docker-compose -f prod-stack.yml down --volumes

up-daemon:
	docker-compose -f prod-stack.yml up -d

down-daemon:
	docker-compose -f prod-stack.yml stop

up-dev:
	docker-compose -f dev-stack.yml up

down-dev:
	docker-compose -f dev-stack.yml down

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
	docker tag keksplorer-worker:latest chebykin/keksplorer-worker:$$version; \
	docker push chebykin/keksplorer-worker:$$version

docker-push-agent:
	@read -p $(PUSH_MSG) version; \
	docker tag keksplorer-agent:latest chebykin/keksplorer-agent:$$version; \
	docker push chebykin/keksplorer-agent:$$version

docker-push-web:
	@read -p $(PUSH_MSG) version; \
	docker tag keksplorer-web:latest chebykin/keksplorer-web:$$version; \
	docker push chebykin/keksplorer-web:$$version

BIN = openclaw-creator
IMAGE = openclaw-creator
ANSIBLE_REPO = ../openclaw-ansible

.PHONY: run build clean docker deploy sync-ansible frontend-build migrate

frontend-build:
	cd frontend && npm ci && npm run build

run:
	go run .

build: frontend-build
	go build -o $(BIN) .

sync-ansible:
	rm -rf ansible/roles
	cp -r $(ANSIBLE_REPO)/roles ansible/roles

docker: sync-ansible
	docker build -t $(IMAGE) .

deploy: sync-ansible
	fly deploy

migrate:
	go run . --migrate

clean:
	rm -f $(BIN)
	rm -rf ansible/roles
	rm -rf static/assets static/index.html

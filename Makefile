BIN = openclaw-creator
IMAGE = openclaw-creator
ANSIBLE_REPO = ../openclaw-ansible

.PHONY: run build clean docker deploy sync-ansible

run:
	go run .

build:
	go build -o $(BIN) .

sync-ansible:
	rm -rf ansible/roles
	cp -r $(ANSIBLE_REPO)/roles ansible/roles

docker: sync-ansible
	docker build -t $(IMAGE) .

deploy: sync-ansible
	fly deploy

clean:
	rm -f $(BIN)
	rm -rf ansible/roles

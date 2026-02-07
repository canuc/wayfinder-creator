BIN = openclaw-creator
IMAGE = openclaw-creator

.PHONY: run build clean docker

run:
	go run .

build:
	go build -o $(BIN) .

docker:
	docker build -t $(IMAGE) .

clean:
	rm -f $(BIN)

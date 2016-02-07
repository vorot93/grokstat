FILES:="data/..."

clean:
	rm -rf ./bindata/* ./bin/*
build: clean
	go generate
	CGO_ENABLED=0 GOOS=linux go build -o ./bin/grokstat
start: build
	./bin/grokstat $(FLAGS)
docker-build:
	docker build -t grokstat:dev .

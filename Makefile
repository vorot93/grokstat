deps:
	glide i && glide rebuild
clean:
	rm -rf ./bin/*
build: clean
	CGO_ENABLED=0 GOOS=linux go build -o ./bin/grokstat
start: build
	./bin/grokstat $(FLAGS)

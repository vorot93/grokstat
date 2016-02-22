FILES:="data/..."

get-deps:
	go get -u "github.com/BurntSushi/toml" "github.com/imdario/mergo" "github.com/jteeuwen/go-bindata/..."
clean:
	rm -rf ./bindata/* ./bin/*
build: clean
	go generate
	CGO_ENABLED=0 GOOS=linux go build -o ./bin/grokstat
start: build
	./bin/grokstat $(FLAGS)

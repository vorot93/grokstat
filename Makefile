FILES:="data/..."

clean:
	rm -rf ./bindata/* ./bin/*
build: clean
	go generate
	go build -o ./bin/grokstat
start: build
	./bin/grokstat $(FLAGS)

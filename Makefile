FILES:="data/..."

bindata:
	#go get -u github.com/jteeuwen/go-bindata/...
	go-bindata -o "bindata/bindata.go" -pkg "bindata" $(FILES)
build: bindata
	go build -o ./bin/grokstat
start: build
	./bin/grokstat $(FLAGS)
clean:
	rm -rf ./bindata ./bin

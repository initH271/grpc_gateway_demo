clean:
	rm -rf api/helloworld/v1

generate:
	buf generate .\proto\helloworld\v1\hello_world.proto

buf-update:
	buf mod update

run:
	go mod tidy && go run main.go
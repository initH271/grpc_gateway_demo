clean:
	rm -rf api/helloworld/v1

generate:
	buf generate .\proto\helloworld\v1\hello_world.proto

run:
	go run main.go
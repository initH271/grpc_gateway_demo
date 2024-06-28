# gRPC-Gateway使用示例

grpc 相比REST HTTP服务没有默认的速度优势, 但是它提供了一些可选的能提高速度的方式:
- 选择性消息压缩
- 负载均衡
- 等
grpc并不是万能的工具. 比如因为兼容性易维护的等需求, 我想提供一个传统的HTTP/JSON式的API.但是另写一份HTTP/JSON的API服务又是非常耗时的一件事. 社区提出的gRPC-Gateway项目解决了这个问题.

gRPC-Gateway是protoc编译器的一个插件, 它读取protobuf的服务定义并生成一个反向代理服务器, 反向代理会将RESTful HTTP API翻译为gRPC.![image](./Pasted%20image%2020240628213521.png)

gRPC-Gateway帮助一次性同时开发HTTP/JSON和gRPC API. 所以gRPC-Gateway的介绍如下:
- protobuf编译器的插件
- 从protobuf中生成代理代码
- 翻译HTTP/JSON 调用为gRPC

## 使用示例

### 创建项目 安装工具

```sh
mkdir grpc_gateway && cd grpc_gateway 
go mod init grpcgateway

# 安装protoc的插件: go gRPC, gRPC Gateway
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/bufbuild/buf/cmd/buf@v1.28.1

# 安装grpcurl工具
scoop install grpcurl
```

### 使用protocol buffers定义helloworld gRPC服务
编辑proto文件
```protobuf
// proto/hellowrold/v1/hello_world.proto
syntax = "proto3";

package helloworld.v1;

option go_package="grpcgateway/helloworld";

import "google/api/annotations.proto";

// The greeting service definition
service Greeter {
  // Sends a greeting
  rpc SayHello (HelloRequest) returns (HelloReply) {
  }
}

// The request message containing the user's name
message HelloRequest {
  string name = 1;
}

// The response message containing the greetings
message HelloReply {
  string message = 1;
}

```

编写buf配置文件
```yaml
# buf.yaml
version: v1
breaking:
  use:
    - FILE
lint:
  use:
    - DEFAULT
deps:
  - buf.build/googleapis/googleapis
```

编写buf.gen配置文件
```yaml
# buf.gen.yaml
version: v1
plugins:
  - plugin: go
    out: api
    opt: paths=source_relative
  - plugin: go-grpc
    out: api
    opt: paths=source_relative
```

编辑makefile文件
```makefile
clean:
	rm -rf api/helloworld/v1

generate:
	buf generate .\proto\helloworld\v1\hello_world.proto

buf-update:
	buf mod update

run:
	go mod tidy && go run main.go
```
运行命令更新依赖, 生成proto代码
```sh
make buf-update

make generate
```

### 编写server端代码, client测试

```go
// main.go
package main

import (
	"context"
	"log"
	"net"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	helloworldpb "grpcgateway/api/proto/helloworld/v1"
)

type server struct {
	helloworldpb.UnimplementedGreeterServer
}

func NewServer() *server {
	return &server{}
}

func (s *server) SayHello(ctx context.Context, in *helloworldpb.HelloRequest) (*helloworldpb.HelloReply, error) {
	return &helloworldpb.HelloReply{Message: in.Name + " world"}, nil
}

func runGRPCServer() {
	// Create a listener on TCP port
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalln("Failed to listen:", err)
	}

	// Create a gRPC server object
	s := grpc.NewServer()
	// Attach the Greeter service to the server
	helloworldpb.RegisterGreeterServer(s, &server{})

	// Enable reflection to allow clients query the server's services.
	reflection.Register(s)
	// Serve gRPC Server
	log.Println("Serving gRPC on 0.0.0.0:8080")
	log.Fatal(s.Serve(lis))
}

func main() {
	runGRPCServer()
}

```

启动服务, 使用grpcurl工具测试
```sh
grpcurl -plaintext -d '{"name":"Sunce"}' localhost:8080 helloworld.v1.Greeter/SayHello

# ouput: 
# {
#   "message": "Sunce world"
# }
```

### 配置gRPC-Gateway

修改proto文件如下, 表示该HTTP API允许POST方法请求
```protobuf
// proto/hellowrold/v1/hello_world.proto
import "google/api/annotations.proto";

// The greeting service definition
service Greeter {
  // Sends a greeting
  rpc SayHello (HelloRequest) returns (HelloReply) {
    option(google.api.http) = {
      post: "/v1/helloworld"
      body: "*"
    };
  }
}
```

修改buf.gen配置文件
```yaml
# buf.gen.yaml
version: v1
plugins:
  - plugin: go
    out: api
    opt: paths=source_relative
  - plugin: go-grpc
    out: api
    opt: paths=source_relative
    # 添加grpc-gateway插件配置
  - plugin: grpc-gateway
    out: api
    opt: 
      - paths=source_relative
```

重新生成代码
```sh
make generate
```

编辑main.go文件, 添加REST服务端代码
```go
// main.go
package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	helloworldpb "grpcgateway/api/proto/helloworld/v1"
)

type server struct {
	helloworldpb.UnimplementedGreeterServer
}

func NewServer() *server {
	return &server{}
}

func (s *server) SayHello(ctx context.Context, in *helloworldpb.HelloRequest) (*helloworldpb.HelloReply, error) {
	return &helloworldpb.HelloReply{Message: in.Name + " world"}, nil
}

func runGRPCServer(wg *sync.WaitGroup) {
	defer wg.Done()
	// Create a listener on TCP port
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalln("Failed to listen:", err)
	}

	// Create a gRPC server object
	s := grpc.NewServer()
	// Attach the Greeter service to the server
	helloworldpb.RegisterGreeterServer(s, &server{})

	// Enable reflection to allow clients query the server's services.
	reflection.Register(s)
	// Serve gRPC Server
	log.Println("Serving gRPC on 0.0.0.0:8080")
	log.Fatal(s.Serve(lis))
}

// 添加REST服务端
func runRESTServer(wg *sync.WaitGroup) {
	defer wg.Done()

	ctx := context.Background()
	mux := runtime.NewServeMux()

	conn, err := grpc.DialContext(ctx, "localhost:8080", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	if err := helloworldpb.RegisterGreeterHandler(ctx, mux, conn); err != nil {
		log.Fatalf("Failed to register handler: %v", err)
	}

	log.Println("Started a gRPC gateway server on 0.0.0.0:8081")
	log.Fatalln(http.ListenAndServe(":8081", mux))
}


func main() {
	var wg sync.WaitGroup
	wg.Add(2)

	go runGRPCServer(&wg)
	go runRESTServer(&wg)

	wg.Wait()
}

```

运行服务, 使用curl工具测试http接口
```sh
➜ curl -X POST http://localhost:8081/v1/helloworld -H "Content-Type: application/json" -d '{"name":"Sunce"}'
# ouput:
# {"message":"Sunce world"}
```
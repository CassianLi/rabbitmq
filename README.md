# RabbitMQ Go 客户端库说明

这是一个用于连接和操作 RabbitMQ 的 Go 语言客户端库。它提供了一种简单的方式来处理 RabbitMQ 连接、消息发布和消费。

## 安装

你可以使用以下命令通过 Go get 安装此库：

```bash
go get github.com/JokerLiAnother/rabbitmq
```

## 使用示例

以下是一个简单的示例，演示如何使用此库来连接 RabbitMQ、发布消息和消费消息：

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/JokerLiAnother/rabbitmq"
)

func main() {
	// 创建一个 RabbitMQ 实例
	rabbitURI := "amqp://guest:guest@localhost:5672/"
	exchange := "myexchange"
	exchangeType := "direct"
	queue := "myqueue"

	rabbit, err := rabbitmq.NewDefaultRabbitMQ(rabbitURI, exchange, exchangeType, queue, true)
	if err != nil {
		log.Fatalf("无法创建 RabbitMQ 实例: %v", err)
	}
	defer rabbit.Close()

	// 发布消息
	message := []byte("Hello, RabbitMQ!")
	err = rabbit.Publish(message)
	if err != nil {
		log.Printf("无法发布消息: %v", err)
	}

	// 消费消息
	consumerCallback := func(msg string) {
		fmt.Printf("收到消息: %s\n", msg)
	}

	err = rabbit.Consume(false, consumerCallback, true)
	if err != nil {
		log.Printf("无法启动消费者: %v", err)
	}

	// 等待一段时间，以便消费消息
	time.Sleep(10 * time.Second)
}
```

## 构造函数

此库提供了几种构造函数来创建 RabbitMQ 实例：

### NewDefaultRabbitMQ

```go
func NewDefaultRabbitMQ(amqpURI, exchange, exchangeType, queue string, autoCreate bool) (*RabbitMQ, error)
```

此构造函数创建一个默认的 RabbitMQ 实例，不启用连接恢复机制。

### NewRecoverRabbitMQ

```go
func NewRecoverRabbitMQ(amqpURI, exchange, exchangeType, queue string, heartbeat, reConnectInterval time.Duration, maxReconnects int) (*RabbitMQ, error)
```

此构造函数创建一个启用连接恢复机制的 RabbitMQ 实例。你可以指定心跳间隔、重连间隔和最大重连次数。

### NewRabbitMQ

```go
func NewRabbitMQ(amqpURI, exchange, exchangeType, queue string, heartbeat, reConnectInterval time.Duration, maxReconnects int, autoCreate bool) (*RabbitMQ, error)
```

此构造函数创建一个自定义的 RabbitMQ 实例，你可以指定是否启用连接恢复机制、心跳间隔、重连间隔和最大重连次数。

## 消费者恢复

如果启用了连接恢复机制，当连接重新建立时，消费者会自动恢复并继续消费消息。

## 重要注意事项

- 请确保在使用此库之前，你已经正确安装并配置了 RabbitMQ 服务器。
- 本库需要使用 Go 模块来管理依赖关系，请确保你的项目启用了 Go 模块。

## 贡献

如果你发现任何问题或者有改进建议，请随时提出 Issue 或者提交 Pull Request。

```

你可以将上述内容复制粘贴到你的 README.md 文件中，并根据需要进行修改。最终提交到仓库后，用户就可以根据这份文档来了解和使用你的 RabbitMQ 客户端库了。
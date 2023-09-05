以下是更新后的README.md文档，其中包括如何使用这个客户端以及注意事项：

# RabbitMQ Go 客户端

RabbitMQ Go 客户端是一个用于与 RabbitMQ 通信的 Go 语言库。它允许您轻松地创建、发布和消费消息，同时提供了灵活的配置选项。

## 安装

使用以下命令安装 RabbitMQ Go 客户端库：

```
go get github.com/JokerLiAnother/rabbitmq
```

## 使用示例

### 创建 RabbitMQ 实例

要使用 RabbitMQ Go 客户端，首先创建一个 `RabbitMQ` 实例。您可以选择使用默认选项或自定义配置。以下是创建实例的两种方法：

#### 使用默认选项

```go
import (
    "github.com/your/package/rabbitmq"
)

func main() {
    // 使用默认选项创建 RabbitMQ 实例
    mq, err := rabbitmq.NewDefaultRabbitMQ("amqp://guest:guest@localhost:5672/", "my_exchange", "direct", "my_queue", true)
    if err != nil {
        log.Fatalf("Failed to create RabbitMQ instance: %v", err)
    }

    defer mq.Close()

    // 其他操作...
}
```

#### 自定义配置

```go
import (
    "github.com/your/package/rabbitmq"
)

func main() {
    // 自定义配置创建 RabbitMQ 实例
    connectionOptions := rabbitmq.ConnectionOptions{
        Heartbeat:         30 * time.Second,
        ReConnectInterval: 10 * time.Second,
        MaxReconnects:     5,
    }

    declareParams := rabbitmq.DeclareParams{
        Durable:    true,
        AutoDelete: false,
        Exclusive:  false,
        NoWait:     false,
        Args:       nil,
    }

    mq, err := rabbitmq.NewRabbitMQ("amqp://guest:guest@localhost:5672/", "my_exchange", "direct", "my_queue", connectionOptions, true, declareParams)
    if err != nil {
        log.Fatalf("Failed to create RabbitMQ instance: %v", err)
    }

    defer mq.Close()

    // 其他操作...
}
```

### 发布消息

```go
// 发布消息到队列
func publishMessage(mq *rabbitmq.RabbitMQ, message string) {
    body := []byte(message)
    if err := mq.Publish(body); err != nil {
        log.Printf("Failed to publish message: %v", err)
    }
}
```

### 消费消息

```go
// 消费消息
func consumeMessages(mq *rabbitmq.RabbitMQ) {
    callback := func(msg string) {
        log.Printf("Received message: %s", msg)
        // 处理消息...
    }

    if err := mq.Consume(false, false, false, callback); err != nil {
        log.Printf("Failed to start consuming messages: %v", err)
    }
}
```

## 配置选项

### DeclareParams

`DeclareParams` 用于配置交换机和队列的声明参数。以下是一些可用的参数：

- `Durable`：是否持久化，默认为 `true`。
- `AutoDelete`：是否自动删除，默认为 `false`。
- `Exclusive`：是否排他，默认为 `false`。
- `NoWait`：是否不等待，默认为 `false`。
- `Args`：额外的参数，类型为 `amqp.Table`。

### ConnectionOptions

`ConnectionOptions` 用于配置连接和重连参数。以下是一些可用的参数：

- `Heartbeat`：心跳间隔，默认为 `0`（禁用心跳）。
- `ReConnectInterval`：重连间隔，默认为 `0`（禁用重连）。
- `MaxReconnects`：最大重连次数，默认为 `0`（不限制）。

### ConsumerOptions

`ConsumerOptions` 用于配置消费者的参数。以下是一些可用的参数：

- `AutoAck`：是否自动确认消息，默认为 `false`。
- `Exclusive`：是否排他性，默认为 `false`。
- `RecoverConsumer`：是否在重连后恢复消费者，默认为 `false`。
- `Callback`：消息处理回调函数。

## 注意事项

- 在使用 `RabbitMQ` 实例后，请确保调用 `Close()` 方法以关闭连接和通道，以释放资源。
- 当配置交换机和队列时，请确保队列名称不为空，否则会导致错误。
- 使用自定义配置时，请确保您了解各个配置参数的含义和影响，以便满足您的需求。

这是一个基本的 RabbitMQ Go 客户端的示例和配置指南。您可以根据自己的需求进行定制和扩展。
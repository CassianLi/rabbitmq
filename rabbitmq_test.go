package rabbitmq

import (
	"fmt"
	"testing"
	"time"
)

func TestRabbitMQ_Publish(t *testing.T) {

	// 创建MQ链接
	// 如果交换机和队列不存在，会自动创建
	client, _ := NewDefaultRabbitMQ("amqp://USER:PASSWORD@127.0.0.1:5672",
		"",
		"",
		"test.golang", true)
	defer client.Close()

	type args struct {
		body []byte
	}
	tests := []struct {
		name    string
		fields  RabbitMQ
		args    args
		wantErr bool
	}{
		{name: "test1", fields: *client, args: args{[]byte("{\"msg\": \"This is a test message!\", \"hello\": \"world!\"}")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mq := &RabbitMQ{
				conn:              tt.fields.conn,
				channel:           tt.fields.channel,
				exchange:          tt.fields.exchange,
				exchangeType:      tt.fields.exchangeType,
				queue:             tt.fields.queue,
				connected:         tt.fields.connected,
				reConnectInterval: tt.fields.reConnectInterval,
				maxReconnects:     tt.fields.maxReconnects,
				recoverConsumer:   tt.fields.recoverConsumer,
				autoAck:           tt.fields.autoAck,
				callback:          tt.fields.callback,
			}
			if err := mq.Publish(tt.args.body); (err != nil) != tt.wantErr {
				t.Errorf("Publish() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRabbitMQ_Consume(t *testing.T) {
	// 重连测试
	client, _ := NewRecoverRabbitMQ("amqp://USER:PASSWORD@127.0.0.1:5672",
		"",
		"",
		"test.golang",
		true,          // queue dont exist, will create it
		5*time.Second, // heartbeat interval
		5*time.Second, // reconnect interval
		10,            // max reconnects
	)
	defer client.Close()

	callbackRecover := func(msg string) {
		fmt.Println("consumer received message: ", msg)
	}
	go client.Consume(true, callbackRecover)
	// 启动后，可关闭网络，模拟断线重连

	select {}

}

func TestRabbitMQ_Consume1(t *testing.T) {
	// 普通连接
	client, _ := NewDefaultRabbitMQ("amqp://USER:PASSWORD@127.0.0.1:5672",
		"",
		"",
		"test.golang",
		true,
	)
	defer client.Close()

	callback := func(msg string) {
		fmt.Println("consumer received message: ", msg)
	}
	go client.Consume(true, callback)

	select {}

}

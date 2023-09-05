package rabbitmq

import (
	"fmt"
	"log"
	"testing"
	"time"
)

const amqpURI = "amqp://admin:admin@47.96.106.87:5672"

func TestRabbitMQ_Publish(t *testing.T) {

	// 创建MQ链接
	// 如果交换机和队列不存在，会自动创建
	client, _ := NewDefaultRabbitMQ(amqpURI,
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
		{name: "test1", fields: *client, args: args{[]byte("{\"msg\": \"This is a test message 1!\", \"hello\": \"world!\"}")}},
		{name: "test2", fields: *client, args: args{[]byte("{\"msg\": \"This is a test message 2!\", \"hello\": \"world!\"}")}},
		{name: "test3", fields: *client, args: args{[]byte("{\"msg\": \"This is a test message 3!\", \"hello\": \"world!\"}")}},
		{name: "test4", fields: *client, args: args{[]byte("{\"msg\": \"This is a test message 4!\", \"hello\": \"world!\"}")}},
		{name: "test4", fields: *client, args: args{[]byte("{\"msg\": \"This is a test message 5!\", \"hello\": \"world!\"}")}},
		{name: "test4", fields: *client, args: args{[]byte("{\"msg\": \"This is a test message 6!\", \"hello\": \"world!\"}")}},
		{name: "test4", fields: *client, args: args{[]byte("{\"msg\": \"This is a test message 7!\", \"hello\": \"world!\"}")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := client.Publish(tt.args.body); (err != nil) != tt.wantErr {
				t.Errorf("Publish() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRabbitMQ_Consume(t *testing.T) {
	// 重连测试
	client, _ := NewRabbitMQ(amqpURI,
		"",
		"direct",
		"test.golang",
		ConnectionOptions{
			Heartbeat:         5,
			ReConnectInterval: 5,
			MaxReconnects:     5,
		},
		true,
		DeclareParams{
			Durable:    true,
			AutoDelete: false,
			Exclusive:  false,
			NoWait:     false,
			Args:       nil,
		},
	)

	callbackRecover := func(msg string) {
		fmt.Println("consumer received message: ", msg)
		time.Sleep(1 * time.Minute)
	}

	go func() {
		log.Printf("consumer start...")
		err := client.Consume(true, false, true, callbackRecover)
		if err != nil {
			log.Default().Println("Consume() error = ", err)
		}
	}()

	// 启动后，可关闭网络，模拟断线重连
	select {}
}

func TestRabbitMQ_Consume_1(t *testing.T) {
	// 重连测试
	client, _ := NewRabbitMQ(amqpURI,
		"",
		"direct",
		"test.golang",
		ConnectionOptions{
			Heartbeat:         5,
			ReConnectInterval: 5,
			MaxReconnects:     5,
		},
		true,
		DeclareParams{
			Durable:    true,
			AutoDelete: false,
			Exclusive:  false,
			NoWait:     false,
			Args:       nil,
		},
	)

	callbackRecover := func(msg string) {
		fmt.Println("consumer 1 received message: ", msg)
		time.Sleep(2 * time.Minute)
	}

	go func() {
		log.Printf("consumer 1 start...")
		err := client.Consume(true, false, true, callbackRecover)
		if err != nil {
			log.Default().Println("Consume() error = ", err)
		}
	}()

	// 启动后，可关闭网络，模拟断线重连
	select {}
}

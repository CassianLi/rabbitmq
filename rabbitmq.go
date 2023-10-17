package rabbitmq

import (
	"errors"
	"fmt"
	"github.com/streadway/amqp"
	"log"
	"time"
)

// DeclareParams exchange and queue declare params
type DeclareParams struct {
	Durable       bool       // 持久化,默认true
	AutoDelete    bool       // 自动删除,默认false
	Exclusive     bool       // 排他性,默认false
	NoWait        bool       // 不等待,默认false
	Args          amqp.Table // 额外参数
	PrefetchCount int        // 消费者预取数量,默认0,不限制
	PrefetchSize  int        // 消费者预取大小,默认0,不限制
}

// ConnectionOptions 重连等配置参数
type ConnectionOptions struct {
	Heartbeat         time.Duration // 心跳间隔,默认0,不开启
	ReConnectInterval time.Duration // 重连间隔,默认0,不重连
	MaxReconnects     int           // 最大重连次数,默认0,不限制
}

type ConsumerOptions struct {
	AutoAck         bool             // 是否自动确认消息,默认false
	Exclusive       bool             // 是否排他性,默认false
	RecoverConsumer bool             // 是否在重连后恢复消费者,默认false
	Callback        func(msg string) // 消费者回调函数
}

type RabbitMQ struct {
	exchange     string
	queue        string
	exchangeType string
	extraParams  DeclareParams // exchange and queue extra params

	connectionOptions ConnectionOptions // 重连等配置参数

	consumerOptions ConsumerOptions // 消费者配置参数

	conn      *amqp.Connection
	channel   *amqp.Channel
	connected bool // Track connection status
}

// NewDefaultRabbitMQ Create a publisher with default options
func NewDefaultRabbitMQ(amqpURI, exchange, exchangeType, queue string, autoCreate bool) (*RabbitMQ, error) {
	return NewRabbitMQ(amqpURI, exchange, exchangeType, queue, ConnectionOptions{}, autoCreate, DeclareParams{Durable: true})
}

// NewRabbitMQ Create a new instance of RabbitMQ
// amqpURI: amqp://guest:guest@localhost:5672/
// exchange: exchange name
// exchangeType: exchange type, default direct
// queue: queue name
// connectionOptions: connection options
// autoCreate: auto create exchange and queue if not exists
// declareParams: exchange and queue declare params
func NewRabbitMQ(amqpURI, exchange, exchangeType, queue string, connectionOptions ConnectionOptions, autoCreate bool, declareParams DeclareParams) (*RabbitMQ, error) {
	if exchangeType == "" {
		exchangeType = "direct"
	}

	if queue == "" {
		return nil, errors.New("queue name is empty")
	}

	mq := &RabbitMQ{
		conn:              nil,
		channel:           nil,
		exchange:          exchange,
		exchangeType:      exchangeType,
		queue:             queue,
		connectionOptions: connectionOptions,
		connected:         false, // Initialize as disconnected
	}

	// Try to establish the initial connection
	err := mq.connectToBroker(amqpURI, autoCreate, declareParams)
	if err != nil {
		mq.connected = false
		log.Printf("failed to connect initially: %v", err)
	} else {
		mq.connected = true
	}

	// Start a goroutine to handle reconnection
	if connectionOptions.ReConnectInterval > 0 && mq.connected {
		// Start a goroutine to handle reconnection
		go mq.reconnectLoop(amqpURI, autoCreate, declareParams)
	}

	return mq, nil
}

// connectToBroker Connect to the broker
func (mq *RabbitMQ) connectToBroker(amqpURI string, autoCreate bool, params DeclareParams) error {
	var err error
	if mq.connectionOptions.Heartbeat > 0 {
		mq.conn, err = amqp.DialConfig(amqpURI, amqp.Config{
			Heartbeat: mq.connectionOptions.Heartbeat,
		})
	} else {
		mq.conn, err = amqp.Dial(amqpURI)
	}

	if err != nil {
		return err
	}

	mq.channel, err = mq.conn.Channel()
	if err != nil {
		log.Printf("failed to open channel: %v", err)
		// Auto-create queue and exchange if enabled
		if autoCreate {
			err = mq.setupExchangeAndQueue(params)
			if err != nil {
				log.Printf("auto-creating queue and exchange: %v", err)
				return err
			}
		} else {
			return err
		}
	}
	log.Printf("declared queue params (%v)", params)
	err = mq.channel.Qos(params.PrefetchCount, params.PrefetchSize, false)
	if err != nil {
		return err
	}

	return nil
}

// reconnectLoop Reconnect to the server in case it goes down, with exponential backoff
func (mq *RabbitMQ) reconnectLoop(amqpURI string, autoCreate bool, params DeclareParams) {
	log.Println("starting reconnect loop")
	reconnectCount := 0 // Initialize reconnect count

	for {
		// Check if maximum reconnection attempts reached
		if mq.connectionOptions.MaxReconnects > 0 && reconnectCount >= mq.connectionOptions.MaxReconnects {
			log.Printf("exceeded maximum reconnection attempts (%d), giving up", mq.connectionOptions.MaxReconnects)
			return
		}

		for !mq.connected {
			// Try to establish the initial connection
			err := mq.connectToBroker(amqpURI, autoCreate, params)
			if err != nil {
				time.Sleep(mq.connectionOptions.ReConnectInterval) // Wait before the next reconnection attempt
				log.Printf("trying to reconnect failed, (attempt %d, after: %v)", reconnectCount+1, mq.connectionOptions.ReConnectInterval)
				// Increment reconnect count
				reconnectCount++
			} else {
				mq.connected = true
				log.Println("reconnected successfully")

				if mq.consumerOptions.RecoverConsumer {
					log.Printf("recovering consumer...")
					// Recover consumer
					// Connection restored, restart message consumption
					err := mq.startConsume()
					if err != nil {
						log.Printf("Error in message consumption: %v", err)
					} else {
						log.Println("recovered consumer successfully")
					}
				}
			}
		}

		// Ensure the channel is closed when connection is lost
		select {
		case <-mq.conn.NotifyClose(make(chan *amqp.Error)):
			mq.connected = false
			log.Println("connection lost, trying to reconnect...")
		}
	}
}

// Close mq connection
func (mq *RabbitMQ) Close() {
	if mq.channel != nil {
		_ = mq.channel.Close()
	}
	if mq.conn != nil {
		_ = mq.conn.Close()
	}
}

// setupExchangeAndQueue Declare exchange and queue
func (mq *RabbitMQ) setupExchangeAndQueue(params DeclareParams) error {
	log.Printf("declaring exchange (%s)", mq.exchange)
	// Declare exchange
	if mq.exchange != "" {
		err := mq.channel.ExchangeDeclare(
			mq.exchange,
			mq.exchangeType,
			params.Durable,
			params.AutoDelete,
			params.Exclusive,
			params.NoWait,
			params.Args,
		)
		if err != nil {
			return err
		}
	}

	log.Printf("declaring queue (%s)", mq.queue)

	// Declare queue
	_, err := mq.channel.QueueDeclare(
		mq.queue,
		params.Durable,
		params.AutoDelete,
		params.Exclusive,
		params.NoWait,
		params.Args,
	)
	if err != nil {
		return err
	}

	// Bind queue to exchange
	if mq.exchange != "" {
		err = mq.channel.QueueBind(
			mq.queue,
			"",
			mq.exchange,
			params.NoWait,
			nil,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// Publish  a message to the queue, creating the queue if needed
func (mq *RabbitMQ) Publish(body []byte) error {
	// Publish 不允许设置重连
	if mq.connectionOptions.ReConnectInterval > 0 {
		return errors.New("reconnect is not allowed when publishing message, please use NewDefaultRabbitMQ to create RabbitMQ instance")
	}
	if !mq.connected {
		return errors.New("not connected to RabbitMQ")
	}

	if err := mq.channel.Publish(
		mq.exchange,
		mq.queue,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        body,
		}); err != nil {
		return err
	}
	return nil
}

// startConsume Start consuming messages
func (mq *RabbitMQ) startConsume() error {
	if !mq.connected {
		return errors.New("not connected to RabbitMQ")
	}

	deliveries, err := mq.channel.Consume(
		mq.queue,
		"",
		mq.consumerOptions.AutoAck,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	for d := range deliveries {
		msg := string(d.Body)

		// Process message
		if mq.consumerOptions.Callback == nil {
			return errors.New("callback function is not set")
		}
		mq.consumerOptions.Callback(msg)

		if !mq.consumerOptions.AutoAck {
			if err := mq.sendAckMsg(d, 5, 5*time.Second); err != nil {
				log.Printf("failed to ack msg: %s, err: %v", msg, err)
			}
		}
	}

	return nil
}

// Consume start consuming messages, autoAck is used to set auto ack message or not
// callback is the function to process message
func (mq *RabbitMQ) Consume(autoAck, exclusive, recover bool, callback func(msg string)) error {
	mq.consumerOptions = ConsumerOptions{
		AutoAck:         autoAck,
		Exclusive:       exclusive,
		RecoverConsumer: recover,
		Callback:        callback,
	}

	return mq.startConsume()
}

// Send Ack confirmation message, set retry count and interval
func (mq *RabbitMQ) sendAckMsg(delivery amqp.Delivery, retryCount int, retryInterval time.Duration) error {
	// 重复发送Ack确认消息
	for i := 0; i < retryCount; i++ {
		if err := delivery.Ack(false); err != nil {
			log.Printf("failed to send ack, after %d times retry !!, err: %v", i, err)
			time.Sleep(retryInterval)
		} else {
			return nil
		}
	}

	return fmt.Errorf("failed to ack after %d retries", retryCount)
}

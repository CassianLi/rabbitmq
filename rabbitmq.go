package rabbitmq

import (
	"errors"
	"fmt"
	"github.com/streadway/amqp"
	"log"
	"time"
)

type RabbitMQ struct {
	conn              *amqp.Connection
	channel           *amqp.Channel
	exchange          string
	exchangeType      string
	queue             string
	connected         bool             // Track connection status
	reConnectInterval time.Duration    // Reconnection interval in seconds,default is 5s
	maxReconnects     int              // Maximum number of reconnection attempts, default is 0 for unlimited
	recoverConsumer   bool             // Whether to recover consumer when connection re-established, default is false
	autoAck           bool             // Whether to auto ack message, default is false
	callback          func(msg string) // Callback function to process message

}

func NewDefaultRabbitMQ(amqpURI, exchange, exchangeType, queue string, autoCreate bool) (*RabbitMQ, error) {
	return NewRabbitMQ(amqpURI, exchange, exchangeType, queue, 0, 0, 0, autoCreate, false)
}

// NewRecoverRabbitMQ Create a new RabbitMQ instance,
// heartbeat is the interval of heartbeats, set to 0 to disable
// reConnectInterval is the reconnection interval in seconds, default is 0, which means no reconnection
// maxReconnects is the maximum number of reconnection attempts, default is 0 for unlimited
func NewRecoverRabbitMQ(amqpURI, exchange, exchangeType, queue string, autoCreate bool, heartbeat, reConnectInterval time.Duration, maxReconnects int) (*RabbitMQ, error) {
	return NewRabbitMQ(amqpURI, exchange, exchangeType, queue, heartbeat, reConnectInterval, maxReconnects, autoCreate, true)
}

// NewRabbitMQ Create a new RabbitMQ instance,
//
//	with exchange declared if needed,
//	heartbeat is the interval of heartbeats, set to 0 to disable
//	reConnectInterval is the reconnection interval in seconds, default is 0, which means no reconnection
//	maxReconnects is the maximum number of reconnection attempts, default is 0 for unlimited
//	autoCreate is used to create exchange and queue if needed
func NewRabbitMQ(amqpURI, exchange, exchangeType, queue string, heartbeat, reConnectInterval time.Duration, maxReconnects int, autoCreate, recoverConsumer bool) (*RabbitMQ, error) {
	if exchangeType == "" {
		exchangeType = "direct"

	}

	mq := &RabbitMQ{
		conn:              nil,
		channel:           nil,
		exchange:          exchange,
		exchangeType:      exchangeType,
		queue:             queue,
		connected:         false, // Initialize as disconnected
		reConnectInterval: reConnectInterval,
		maxReconnects:     maxReconnects,
		recoverConsumer:   recoverConsumer,
		autoAck:           false,
	}

	// Try to establish the initial connection
	err := mq.connectToBroker(amqpURI, heartbeat, autoCreate)
	if err != nil {
		mq.connected = false
		log.Printf("failed to connect initially: %v", err)
	} else {
		mq.connected = true
	}

	// Start a goroutine to handle reconnection
	if reConnectInterval > 0 && mq.connected {
		// Start a goroutine to handle reconnection
		go mq.reconnectLoop(amqpURI, heartbeat, autoCreate)
	}

	return mq, nil
}

// connectToBroker Connect to the broker
func (mq *RabbitMQ) connectToBroker(amqpURI string, heartbeat time.Duration, autoCreate bool) error {
	var err error
	if heartbeat > 0 {
		mq.conn, err = amqp.DialConfig(amqpURI, amqp.Config{
			Heartbeat: heartbeat,
		})
	} else {
		mq.conn, err = amqp.Dial(amqpURI)
	}

	if err != nil {
		return err
	}

	mq.channel, err = mq.conn.Channel()
	if err != nil {
		return err
	}

	// Auto-create queue and exchange if enabled
	if autoCreate {
		err = mq.setupExchangeAndQueue()
		if err != nil {
			return err
		}
	}

	return nil
}

// reconnectLoop Reconnect to the server in case it goes down, with exponential backoff
func (mq *RabbitMQ) reconnectLoop(amqpURI string, heartbeat time.Duration, autoCreate bool) {
	log.Println("starting reconnect loop")

	reconnectCount := 0 // Initialize reconnect count

	for {
		// Check if maximum reconnection attempts reached
		if mq.maxReconnects > 0 && reconnectCount >= mq.maxReconnects {
			log.Printf("exceeded maximum reconnection attempts (%d), giving up", mq.maxReconnects)
			return
		}

		for !mq.connected {
			// Try to establish the initial connection
			err := mq.connectToBroker(amqpURI, heartbeat, autoCreate)
			if err != nil {
				time.Sleep(mq.reConnectInterval) // Wait before the next reconnection attempt
				log.Printf("trying to reconnect failed, (attempt %d, after: %v)", reconnectCount+1, mq.reConnectInterval)
				// Increment reconnect count
				reconnectCount++
			} else {
				mq.connected = true
				log.Println("reconnected successfully")
				log.Println("recoverConsumer: ", mq.recoverConsumer)

				if mq.recoverConsumer {
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
func (mq *RabbitMQ) setupExchangeAndQueue() error {
	// Declare exchange
	if mq.exchange != "" {
		err := mq.channel.ExchangeDeclare(
			mq.exchange,
			mq.exchangeType,
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return err
		}
	}

	// Declare queue
	_, err := mq.channel.QueueDeclare(
		mq.queue,
		true,
		false,
		false,
		false,
		nil,
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
			false,
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
	if mq.reConnectInterval > 0 {
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
	log.Println(mq.autoAck)
	if !mq.connected {
		return errors.New("not connected to RabbitMQ")
	}

	deliveries, err := mq.channel.Consume(
		mq.queue,
		"",
		mq.autoAck,
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
		if mq.callback == nil {
			return errors.New("callback function is not set")
		}
		mq.callback(msg)

		if !mq.autoAck {
			if err := mq.sendAckMsg(d, 5, 5*time.Second); err != nil {
				log.Printf("failed to ack msg: %s, err: %v", msg, err)
			}
		}
	}

	return nil
}

// Consume start consuming messages, autoAck is used to set auto ack message or not
// callback is the function to process message
func (mq *RabbitMQ) Consume(autoAck bool, callback func(msg string)) error {
	mq.autoAck = autoAck
	mq.callback = callback

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

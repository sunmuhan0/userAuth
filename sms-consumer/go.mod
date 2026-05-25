module ttuser/sms-consumer

go 1.18

require ttuser/event-consumer v0.0.0

require github.com/streadway/amqp v1.1.0 // indirect

replace ttuser/event-consumer => ../event-consumer

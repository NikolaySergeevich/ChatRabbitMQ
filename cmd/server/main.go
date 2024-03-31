package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/streadway/amqp"
)

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			},
		),
	)

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		logger.Debug("amqp Dial", slog.Any("err", err))
		return
	}

	defer conn.Close()

	amqpChannel, err := conn.Channel()
	if err != nil {
		logger.Error("conn Channel", slog.Any("err", err))
		return
	}

	// создаем точку обмена типа fanout для рассылки во все очереди с именем chat
	err = amqpChannel.ExchangeDeclare(
		"chat",
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logger.Error("channel ExchangeDeclare", slog.Any("err", err))
		return
	}

	// создаем очередь, которую слушаем. В нее клиенты отправляют нам сообщения
	msgQ, err := amqpChannel.QueueDeclare(
		"msg_queue",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logger.Error("channel QueueDeclare", slog.Any("err", err))
		return
	}

	// Создаем консьюмер для прослушивания этой очереди
	msgCh, err := amqpChannel.Consume(
		msgQ.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logger.Error("channel Consume", slog.Any("err", err))
		return
	}
	_ = msgCh
}

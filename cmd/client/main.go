package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/streadway/amqp"
)

var username string

func init() {
	// флаг через который мы передаем имя пользователя
	flag.StringVar(&username, "u", "anonymous", "-u username")
	flag.Parse()
}

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

	// создаем подключение
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		logger.Debug("amqp Dial", slog.Any("err", err))
		return
	}

	defer conn.Close()

	// Создаем канал
	amqpChannel, err := conn.Channel()
	if err != nil {
		logger.Error("conn Channel", slog.Any("err", err))
		return
	}
	defer amqpChannel.Close()

	// декларируем уникальную очередь для клиента.
	msgQ, err := amqpChannel.QueueDeclare(
		"", // Имя генерируется автоматчиески
		false,
		false,
		true,
		false,
		nil,
	)

	// линкуем нашу очередь к точке обмена chat
	err = amqpChannel.QueueBind(
		msgQ.Name,
		"",
		"chat", // используем точку обмена chat
		false,
		nil,
	)

	// создаем консьюмер для вычитки сообщений
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

	type message struct {
		ID       string `json:"id"`
		Message  string `json:"body"`
		Username string `json:"username"`
		Ts       int64  `json:"ts"`
	}

	// вставьте код отправки и приема сообщий здесь
}

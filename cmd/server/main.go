package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
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

	wd, err := os.Getwd()
	if err != nil {
		logger.Error("os Getwd", slog.Any("err", err))
		return
	}

	f, err := os.Create(filepath.Join(wd, "output.txt"))
	if err != nil {
		logger.Error("os Create", slog.Any("err", err))
		return
	}
	defer f.Close()

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

	go func() {
		<-ctx.Done()
		amqpChannel.Close()
	}()

	g := sync.WaitGroup{}
	g.Add(1)

	go func() {
		defer g.Done()
		for d := range msgCh {
			if _, err := f.Write(append(d.Body, []byte{'\n'}...)); err != nil {
				logger.Error("file Write", slog.Any("err", err))
				return
			}
			err = amqpChannel.Publish(
				"chat",
				"",
				false,
				false,
				amqp.Publishing{
					ContentType: "text/plain",
					Body:        d.Body,
				},
			)
			if err != nil {
				logger.Error("channel Publish", slog.Any("err", err))
				return
			}
		}
	}()

	logger.Debug("[*] Waiting for messages. To exit press CTRL+C")
	g.Wait()
}

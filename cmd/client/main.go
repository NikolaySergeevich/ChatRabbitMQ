package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

var username string

func init() {
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
	defer amqpChannel.Close()

	msgQ, err := amqpChannel.QueueDeclare(
		"",
		false,
		false,
		true,
		false,
		nil,
	)

	err = amqpChannel.QueueBind(
		msgQ.Name,
		"",
		"chat",
		false,
		nil,
	)

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

	type message struct {
		ID       string `json:"id"`
		Message  string `json:"body"`
		Username string `json:"username"`
		Ts       int64  `json:"ts"`
	}

	var id string
	var mu sync.RWMutex

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d := <-msgCh:
				var m message
				if err := json.Unmarshal(d.Body, &m); err != nil {
					logger.Error("json Unmarshal", slog.Any("err", err))
					continue
				}
				mu.RLock()
				if m.ID == id {
					mu.RUnlock()
					continue
				}
				mu.RUnlock()

				fmt.Printf("\r%s:> %s", m.Username, m.Message)
				fmt.Printf("%s> ", username)
			}
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		fmt.Printf("%s:> ", username)
		msg, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		mu.Lock()
		id = uuid.New().String()
		mu.Unlock()

		encoded, err := json.Marshal(
			message{
				ID:       id,
				Message:  msg,
				Username: username,
				Ts:       time.Now().Unix(),
			},
		)
		if err != nil {
			log.Fatal(err)
		}

		err = amqpChannel.Publish(
			"",
			"msg_queue",
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        encoded,
			},
		)
		if err != nil {
			logger.Error("amqp channel Publish", slog.Any("err", err))
			continue
		}
	}
}

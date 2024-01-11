package main

import (
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

func Run(f func(ctx context.Context, log *zap.Logger) error) {
	log, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()
	// No graceful shutdown.
	ctx := context.Background()
	if err := f(ctx, log); err != nil {
		log.Fatal("Run failed", zap.Error(err))
	}
	// Done.
}

func main() {
	Run(func(ctx context.Context, log *zap.Logger) error {
		// Dispatcher handles incoming updates.
		dispatcher := tg.NewUpdateDispatcher()
		opts := telegram.Options{
			Logger:        log,
			UpdateHandler: dispatcher,
		}
		return telegram.BotFromEnvironment(ctx, opts, func(ctx context.Context, client *telegram.Client) error {
			// Raw MTProto API client, allows making raw RPC calls.
			api := tg.NewClient(client)

			// Helper for sending messages.
			sender := message.NewSender(api)

			// Setting up handler for incoming message.
			dispatcher.OnNewMessage(func(ctx context.Context, entities tg.Entities, u *tg.UpdateNewMessage) error {
				m, ok := u.Message.(*tg.Message)
				if !ok || m.Out {
					// Outgoing message, not interesting.
					return nil
				}

				// Sending reply.
				_, err := sender.Reply(entities, u).Text(ctx, m.Message)
				return err
			})
			return nil
		}, telegram.RunUntilCanceled)
	})
}

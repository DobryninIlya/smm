package tg_assembly

import (
	"context"
	"fmt"
	"github.com/gotd/td/telegram/message/peer"
	"github.com/gotd/td/tg"
	"strconv"
)

func OnNewChannelMessageHandler(client *Client, like, parse, comment bool, clientName string) func(ctx context.Context, e tg.Entities, u *tg.UpdateNewChannelMessage) error {
	return func(ctx context.Context, e tg.Entities, u *tg.UpdateNewChannelMessage) error {
		msg, ok := u.Message.(*tg.Message)
		if ok {
			if parse {
				fmt.Println(msg.Message)
				//messages <- msg
			}
			entities := peer.NewEntities(e.Users, e.Chats, e.Channels)
			peerID, err := entities.ExtractPeer(msg.GetPeerID())
			if like {

				reaction := []tg.ReactionClass{
					&tg.ReactionEmoji{Emoticon: "👍"},
				}
				if err != nil {
					client.log.Println("Ошибка пиров: " + err.Error())
				}
				_, err = client.api.MessagesSendReaction(ctx, &tg.MessagesSendReactionRequest{
					Peer:     peerID,
					MsgID:    msg.ID - 1,
					Reaction: reaction,
				})
				reactions, err := client.api.MessagesGetAvailableReactions(ctx, 0)
				resultReaction, ok := reactions.(*tg.MessagesAvailableReactions)
				if !ok {
					client.log.Println("Ошибка при получении реакций")
					return nil
				}
				if err != nil {
					reaction[0] = &tg.ReactionEmoji{Emoticon: resultReaction.Reactions[0].Reaction}
				}
				_, err = client.sender.To(peerID).Reaction(ctx, msg.ID, reaction...)
				if err != nil {
					client.log.Println("Ошибка постановки реакции на сообщение: " + err.Error() + "\nВ чате " + msg.GetPeerID().String())
				} else {
					successCounter++
					client.log.Println(clientName + " | " + strconv.Itoa(successCounter) + " Поставил реакцию на сообщение в чате " + msg.GetPeerID().String())
				}
			}
			//client.sender.Reply(e, u)
			if comment {
				message, _ := u.GetMessage().AsNotEmpty()
				if message.GetPost() {
					discussion, err := client.api.MessagesGetDiscussionMessage(ctx, &tg.MessagesGetDiscussionMessageRequest{
						Peer:  peerID,
						MsgID: msg.ID,
					})
					//discussionPeer, err := discussion.GetChats()[0]
					discussionMessage, ok := discussion.Messages[0].(*tg.Message)
					if !ok {
						client.log.Println("Ошибка при преобразовании сообщения")
						return nil
					}
					discussionMessage.GetPeerID()
					discussion.Chats[0].GetID()
					channel, ok := discussion.Chats[0].(*tg.Channel)
					if !ok {
						client.log.Println("Ошибка при преобразовании канала")
						return nil
					}
					discussionPeerID := tg.InputPeerClass(&tg.InputPeerChannel{
						ChannelID:  channel.ID,
						AccessHash: channel.AccessHash,
					})
					_, err = client.sender.To(discussionPeerID).ReplyMsg(discussion.GetMessages()[0]).Text(ctx, "Круто")
					if err != nil {
						client.log.Println("Ошибка отправки сообщения: " + err.Error())
					}
				}
			}
		}
		return nil
	}
}
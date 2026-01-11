package ws

import (
	"context"
	"errors"
	"sync"
)

type RuntimeClient struct {
	ctx      context.Context
	cancel   context.CancelFunc
	ws       *WebSocket
	senderID string
	convID   string
	out      chan []byte
	once     sync.Once
}

func NewClient(
	parent context.Context,
	ws *WebSocket,
	senderID, convID string,
) *RuntimeClient {
	ctx, cancel := context.WithCancel(parent)
	c := &RuntimeClient{
		ctx:      ctx,
		cancel:   cancel,
		ws:       ws,
		senderID: senderID,
		convID:   convID,
		out:      make(chan []byte, 256),
	}
	go c.writeLoop()
	return c
}

func (c *RuntimeClient) SenderID() string       { return c.senderID }
func (c *RuntimeClient) ConversationID() string { return c.convID }

func (c *RuntimeClient) Send(ctx context.Context, data []byte) error {
	select {
	case c.out <- data:
		return nil
	case <-c.ctx.Done():
		return errors.New("client closed")
	}
}

func (c *RuntimeClient) Close() {
	c.once.Do(func() {
		c.cancel()
		close(c.out)
		c.ws.Close()
	})
}

func (c *RuntimeClient) writeLoop() {
	defer c.Close()
	for {
		select {
		case <-c.ctx.Done():
			return
		case data, ok := <-c.out:
			if !ok {
				return
			}
			_ = c.ws.WriteMessage(data)
		}
	}
}

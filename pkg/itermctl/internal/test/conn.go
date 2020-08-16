package test

import (
	log "github.com/sirupsen/logrus"
	iterm2 "mrz.io/itermctl/pkg/itermctl/proto"
	"sync"
)

type Conn struct {
	requests       []*iterm2.ClientOriginatedMessage
	responsesInCh  <-chan *iterm2.ServerOriginatedMessage
	responsesOutCh chan *iterm2.ServerOriginatedMessage
	mx             *sync.Mutex
	closed         bool
	stopCh         chan struct{}
}

func NewConn(responses <-chan *iterm2.ServerOriginatedMessage) *Conn {
	conn := &Conn{responsesInCh: responses, responsesOutCh: make(chan *iterm2.ServerOriginatedMessage),
		stopCh: make(chan struct{}), mx: &sync.Mutex{}}

	if responses != nil {
		go func() {
			for resp := range conn.responsesInCh {
				conn.responsesOutCh <- resp
				log.Error("written 1 message")
			}

			conn.mx.Lock()
			defer conn.mx.Unlock()
			close(conn.responsesOutCh)
			close(conn.stopCh)
		}()
	}

	return conn
}

func (c *Conn) SendRequests(src <-chan *iterm2.ClientOriginatedMessage) {
	go func() {
		for msg := range src {
			go func(msg *iterm2.ClientOriginatedMessage) {
				c.mx.Lock()
				defer c.mx.Unlock()
				c.requests = append(c.requests, msg)
			}(msg)
		}
	}()
}

func (c *Conn) Requests() []*iterm2.ClientOriginatedMessage {
	c.mx.Lock()
	defer c.mx.Unlock()
	return c.requests
}

func (c *Conn) Responses() <-chan *iterm2.ServerOriginatedMessage {
	return c.responsesOutCh
}

func (c *Conn) Close() error {
	return nil
}

func (c *Conn) Done() <-chan struct{} {
	return c.stopCh
}

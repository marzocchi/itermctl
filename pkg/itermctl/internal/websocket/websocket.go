package websocket

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"mrz.io/itermctl/pkg/itermctl/proto"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Websocket struct {
	conn           *websocket.Conn
	mx             *sync.Mutex
	headers        map[string][]string
	subproto       string
	serviceUrl     url.URL
	closeCtx       context.Context
	cancelCloseCtx context.CancelFunc
}

func NewWebsocket(dialFunc func() (net.Conn, error), serviceUrl url.URL, subproto string, headers map[string][]string) (*Websocket, error) {
	closeCtx, cancelCloseCtx := context.WithCancel(context.Background())
	ws := &Websocket{
		mx:             &sync.Mutex{},
		subproto:       subproto,
		headers:        headers,
		serviceUrl:     serviceUrl,
		closeCtx:       closeCtx,
		cancelCloseCtx: cancelCloseCtx,
	}

	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 7 * time.Second,
		NetDial: func(network, addr string) (net.Conn, error) {
			return dialFunc()
		},
	}

	err := ws.connect(dialer)
	if err != nil {
		return nil, fmt.Errorf("websocket: connect: %w", err)
	}

	log.Debug("websocket: connected")

	return ws, nil
}

func (ws *Websocket) connect(dialer *websocket.Dialer) error {
	dialer.Subprotocols = []string{ws.subproto}

	conn, _, err := dialer.Dial(ws.serviceUrl.String(), ws.headers)

	if err != nil {
		return err
	}

	ws.mx.Lock()
	defer ws.mx.Unlock()
	ws.conn = conn

	return nil
}

func (ws *Websocket) Close() error {
	ws.mx.Lock()
	defer ws.mx.Unlock()
	err := ws.conn.Close()

	return err
}

func (ws *Websocket) Done() <-chan struct{} {
	return ws.closeCtx.Done()
}

func (ws *Websocket) SendRequests(src <-chan *iterm2.ClientOriginatedMessage) {
	go func() {
		for msg := range src {
			data, err := proto.Marshal(msg)
			if err != nil {
				log.Errorf("websocket: could not marshal msg: %s", msg)
				continue
			}

			if err := ws.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				log.Errorf("websocket: could not write messages: %s", err)
				continue
			}
		}
	}()
}

func (ws *Websocket) Responses() <-chan *iterm2.ServerOriginatedMessage {
	responses := make(chan *iterm2.ServerOriginatedMessage, 1000)

	go func() {
		for {
			_, data, err := ws.conn.ReadMessage()
			if err != nil {
				log.Errorf("websocket: error reading message: %s", err)
				break
			}

			msg := &iterm2.ServerOriginatedMessage{}
			if err := proto.Unmarshal(data, msg); err != nil {
				log.Errorf("websocket: could not unmarshal message: %s", err)
				continue
			}

			responses <- msg
		}
		close(responses)
		ws.cancelCloseCtx()
	}()
	return responses
}

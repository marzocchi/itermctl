package itermctl

import (
	"context"
	"fmt"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"mrz.io/itermctl/pkg/itermctl/internal/seq"
	"mrz.io/itermctl/pkg/itermctl/internal/websocket"
	"mrz.io/itermctl/pkg/itermctl/proto"
	"net"
	"net/url"
	"os"
	"reflect"
	"sync"
	"time"
)

const (
	AllSessions = "all"
	magicString = "61DF88DC-3423-4823-B725-22570E01C027"
)

var (
	Socket                           = "~/Library/Application Support/iTerm2/private/socket"
	Subprotocol                      = "api.iterm2.com"
	AppName                          = "itermctl"
	LibraryVersion                   = "itermctl 0.0.1"
	Origin                           = "ws://localhost/"
	Url                              = url.URL{Scheme: "ws", Host: "localhost:1912"}
	ErrClosed                        = fmt.Errorf("connection is closed")
	ErrNoMessageId                   = fmt.Errorf("can't get response without a message ID")
	ErrSessionNotFound               = fmt.Errorf("NotificationResponse_SESSION_NOT_FOUND")
	ErrRequestMalformed              = fmt.Errorf("NotificationResponse_REQUEST_MALFORMED")
	ErrNotSubscribed                 = fmt.Errorf("NotificationResponse_NOT_SUBSCRIBED")
	ErrAlreadySubscribed             = fmt.Errorf("NotificationResponse_ALREADY_SUBSCRIBED")
	ErrDuplicatedServerOriginatedRpc = fmt.Errorf("NotificationResponse_DUPLICATE_SERVER_ORIGINATED_RPC")
	ErrInvalidIdentifier             = fmt.Errorf("NotificationResponse_INVALID_IDENTIFIER")
	WaitResponseTimeout              = 5 * time.Second
)

func init() {
	var level log.Level
	logLevel := os.Getenv("ITERMCTL_LOG_LEVEL")

	if logLevel != "" {
		level, _ = log.ParseLevel(logLevel)
	}

	log.SetLevel(level)
}

// AcceptFunc can be used when creating a Receiver to filter out uninteresting ServerOriginatedMessages.
type AcceptFunc func(msg *iterm2.ServerOriginatedMessage) bool

// AcceptSubmessageType filters ServerOriginatedMessages whose submessage has the same type as the given example.
func AcceptSubmessageType(example interface{}) AcceptFunc {
	return func(msg *iterm2.ServerOriginatedMessage) bool {
		if example == nil {
			return true
		}
		submessageType := reflect.TypeOf(msg.Submessage)
		return submessageType == reflect.TypeOf(example)
	}
}

// AcceptNotificationType filters ServerOriginatedMessages whose submessage is a Notification of the given type.
func AcceptNotificationType(t iterm2.NotificationType) AcceptFunc {
	return func(msg *iterm2.ServerOriginatedMessage) bool {
		actualType := getNotificationType(msg.GetNotification())
		return actualType == t
	}
}

func acceptMessageId(msgId int64) AcceptFunc {
	return func(msg *iterm2.ServerOriginatedMessage) bool {
		return msg.GetId() == msgId
	}
}

type Client struct {
	requests        chan *iterm2.ClientOriginatedMessage
	responses       <-chan *iterm2.ServerOriginatedMessage
	addReceivers    chan *receiver
	deleteReceivers chan *receiver
	closed          bool
	closingLock     *sync.Mutex
}

type Connection interface {
	SendRequests(src <-chan *iterm2.ClientOriginatedMessage)
	Responses() <-chan *iterm2.ServerOriginatedMessage
	Close() error
	Done() <-chan struct{}
}

// GetCredentialsAndConnect checks if iTerm2 is configured to require authentication, retrieves the cookie and key if
// necessary, and then establishes the connection to iTerm2's socket.
func GetCredentialsAndConnect(appName string, active bool) (Connection, error) {
	if appName == "" {
		appName = AppName
	}

	var creds *Credentials

	if err := AuthDisabled(); err != nil {
		creds, err = GetCredentials(appName, active)
		if err != nil {
			return nil, err
		}
	}

	conn, err := Connect(appName, creds)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// Connect connects to iTerm2's socket using the optional credentials.
func Connect(appName string, creds *Credentials) (Connection, error) {
	if appName == "" {
		appName = AppName
	}

	socket, err := homedir.Expand(Socket)
	if err != nil {
		return nil, fmt.Errorf("connect: cannot resolve socket path: %w", err)
	}

	var headers = map[string][]string{
		"Origin":                   {Origin},
		"x-iterm2-disable-auth-ui": {"false"},
		"x-iterm2-advisory-name":   {appName},
		"x-iterm2-library-version": {LibraryVersion},
	}

	if creds != nil {
		headers["x-iterm2-cookie"] = []string{creds.Cookie}
		headers["x-iterm2-key"] = []string{creds.Key}
	}

	conn, err := websocket.NewWebsocket(
		func() (net.Conn, error) {
			return net.Dial("unix", socket)
		},
		Url,
		Subprotocol,
		headers,
	)

	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	return conn, err
}

// NewClient creates a new iTerm2 API Client.
func NewClient(conn Connection) *Client {
	client := &Client{
		requests: make(chan *iterm2.ClientOriginatedMessage),
		closed:   false,
	}

	conn.SendRequests(client.requests)
	client.responses = conn.Responses()

	client.addReceivers = make(chan *receiver)
	client.deleteReceivers = make(chan *receiver)
	client.closingLock = &sync.Mutex{}

	go func() {
		var receivers []*receiver

		for {
			select {
			case newReceiver := <-client.addReceivers:
				receivers = append(receivers, newReceiver)

			case recv := <-client.deleteReceivers:
				var tmp []*receiver
				for _, r := range receivers {
					if r == recv {
						close(r.ch)
					} else {
						tmp = append(tmp, r)
					}
				}

				receivers = tmp

			case msg, ok := <-client.responses:
				if !ok {
					goto shutdown
				}

				if len(receivers) == 0 {
					log.Warnf("message ID %d lost: no receivers registered", msg.GetId())
					continue
				}

				for _, recv := range receivers {
					if !recv.accept(msg) {
						log.Debugf("message ID %d not accepted by %q", msg.GetId(), recv.name)
						continue
					}

					select {
					case recv.ch <- msg:
						log.Debugf("message ID %d sent to to %q", msg.GetId(), recv.name)
					case <-time.After(1 * time.Second):
						log.Errorf("message ID %d send to %q timed out", msg.GetId(), recv.name)
					}
				}
			}
		}
	shutdown:
		client.closingLock.Lock()
		defer client.closingLock.Unlock()
		client.closed = true

		close(client.addReceivers)
		close(client.deleteReceivers)
		close(client.requests)

		for _, recv := range receivers {
			close(recv.ch)
		}
	}()

	return client
}

// Receiver returns a channel from which ServerOriginatedMessages can be read until the Connection is closed or the
// given context is canceled. A context should be given only to interrupt receiving before the Connection is closed,
// and should not be the same as the one used to cancel the Connection. The receiver channel will receive a copy of any
// ServerOriginatedMessage being shipped on the Connection, but an AcceptFunc can be given to exclude uninteresting
// messages.
func (c *Client) Receiver(ctx context.Context, name string, f AcceptFunc) (<-chan *iterm2.ServerOriginatedMessage, error) {
	c.closingLock.Lock()
	defer c.closingLock.Unlock()
	if c.closed {
		return nil, ErrClosed
	}

	recv := newReceiver(name, f)

	if ctx != nil {
		go func() {
			<-ctx.Done()
			c.closingLock.Lock()
			defer c.closingLock.Unlock()
			if !c.closed {
				c.deleteReceivers <- recv
			}
		}()
	}

	c.addReceivers <- recv
	return recv.ch, nil
}

// Send sends a message to iTerm2, without waiting for a response. ErrClosed is returned when Send is called after
// the connection was closed.
func (c *Client) Send(msg *iterm2.ClientOriginatedMessage) error {
	c.closingLock.Lock()
	defer c.closingLock.Unlock()

	if c.closed {
		return ErrClosed
	}

	c.requests <- msg
	return nil
}

// Request sends a message to iTerm2 expecting a response that will be written to the returned channel. The channel is
// closed after the first message arrives. A context can be given to cancel waiting for the response.
func (c *Client) Request(ctx context.Context, req *iterm2.ClientOriginatedMessage) (<-chan *iterm2.ServerOriginatedMessage, error) {
	if req.Id == nil {
		return nil, ErrNoMessageId
	}

	respCh := make(chan *iterm2.ServerOriginatedMessage)
	recvCtx, cancelRecv := context.WithCancel(ctx)

	recv, err := c.Receiver(
		recvCtx,
		fmt.Sprintf("message ID %d receiver", req.GetId()),
		acceptMessageId(req.GetId()),
	)

	if err != nil {
		return nil, err
	}

	err = c.Send(req)
	if err != nil {
		return nil, err
	}

	go func() {
		select {
		case <-recvCtx.Done():
			break
		case resp := <-recv:
			defer cancelRecv()
			if resp != nil {
				respCh <- resp
			}
			break
		}
		close(respCh)
	}()

	return respCh, nil
}

// GetResponse sends a message to iTerm2, and waits up to WaitResponseTimeout for a message to be read from src and
// returns it. If the message is an error from iTerm2, a nil message and an error are returned. A nil message with an
// error will also be returned if WaitResponseTimeout expires before a message is received.
func (c *Client) GetResponse(ctx context.Context, req *iterm2.ClientOriginatedMessage) (*iterm2.ServerOriginatedMessage, error) {
	src, err := c.Request(ctx, req)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, WaitResponseTimeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-src:
		if resp != nil {
			if err := GetServerError(resp); err != nil {
				return nil, err
			}
		}
		return resp, nil
	}
}

// Subscribe uses the given NotificationRequest to subscribe with iTerm2, and returns a channel from which notifications
// of requested type can be read. The NotificationRequest will be modified to ensure the Subscribe field is set to true.
// The subscription will be canceled automatically as soon as the context is canceled. The subscription lasts until the
// give context is canceled or the client connection is closed.
func (c *Client) Subscribe(ctx context.Context, req *iterm2.NotificationRequest) (<-chan *iterm2.Notification, error) {
	notificationsCh := make(chan *iterm2.Notification)

	subscribe := true
	req.Subscribe = &subscribe

	msg := NewClientOriginatedMessage()
	msg.Submessage = &iterm2.ClientOriginatedMessage_NotificationRequest{
		NotificationRequest: req,
	}

	respCh, err := c.Request(context.Background(), msg)
	if err != nil {
		return nil, err
	}

	resp := <-respCh

	subscriptionErr := getSubscriptionStatusError(resp)
	if subscriptionErr != nil {
		return nil, fmt.Errorf("client: subscription: %w", subscriptionErr)
	}

	if ctx == nil {
		ctx = context.Background()
	}

	recv, err := c.Receiver(ctx, fmt.Sprintf("%s receiver", req.NotificationType.String()),
		AcceptNotificationType(req.GetNotificationType()))

	if err != nil {
		return nil, err
	}

	go func() {
		for msg := range recv {
			notificationsCh <- msg.GetNotification()
		}
		close(notificationsCh)
		log.Debugf("unsubscribing from %s", req.GetNotificationType())

		unsubReq := NewNotificationRequest(false, req.GetNotificationType(), req.GetSession())
		unsubReq.Arguments = req.Arguments

		unsubErr := c.unsubscribe(unsubReq)
		if unsubErr == nil {
			log.Debugf("successfully unsubscribed to %s", req.GetNotificationType())
		} else {
			log.Errorf("unsubscribe: %s", unsubErr)
		}
	}()

	return notificationsCh, nil
}

func (c *Client) unsubscribe(req *iterm2.NotificationRequest) error {
	msg := NewClientOriginatedMessage()
	msg.Submessage = &iterm2.ClientOriginatedMessage_NotificationRequest{
		NotificationRequest: req,
	}

	respCh, err := c.Request(context.Background(), msg)
	if err != nil {
		return err
	}

	resp := <-respCh

	return getSubscriptionStatusError(resp)
}

func GetServerError(msg *iterm2.ServerOriginatedMessage) error {
	if msg.GetError() != "" {
		return fmt.Errorf("client: error response for message ID %d: %s", msg.GetId(), msg.GetError())
	}
	return nil
}

// NewClientOriginatedMessage creates a new messages with an unique Id
func NewClientOriginatedMessage() *iterm2.ClientOriginatedMessage {
	return &iterm2.ClientOriginatedMessage{
		Id: seq.MessageId.Next(),
	}
}

// NewNotificationRequest creates a notification request to subscribe or unsubscribe for the given notification
// type. If an empty sessionId is given, the subscription is created for all sessions.
func NewNotificationRequest(subscribe bool, nt iterm2.NotificationType, sessionId string) *iterm2.NotificationRequest {
	if sessionId == "" {
		sessionId = AllSessions
	}

	return &iterm2.NotificationRequest{
		Session:          &sessionId,
		Subscribe:        &subscribe,
		NotificationType: &nt,
	}
}

type receiver struct {
	name       string
	ch         chan *iterm2.ServerOriginatedMessage
	acceptFunc AcceptFunc
}

func newReceiver(name string, accept AcceptFunc) *receiver {
	if accept == nil {
		accept = func(message *iterm2.ServerOriginatedMessage) bool {
			return true
		}
	}

	return &receiver{name: name, acceptFunc: accept, ch: make(chan *iterm2.ServerOriginatedMessage)}
}

func (r *receiver) accept(msg *iterm2.ServerOriginatedMessage) bool {
	return r.acceptFunc(msg)
}

func getNotificationType(n *iterm2.Notification) iterm2.NotificationType {
	if n.GetTerminateSessionNotification() != nil {
		return iterm2.NotificationType_NOTIFY_ON_TERMINATE_SESSION
	} else if n.GetNewSessionNotification() != nil {
		return iterm2.NotificationType_NOTIFY_ON_NEW_SESSION
	} else if n.GetCustomEscapeSequenceNotification() != nil {
		return iterm2.NotificationType_NOTIFY_ON_CUSTOM_ESCAPE_SEQUENCE
	} else if n.GetServerOriginatedRpcNotification() != nil {
		return iterm2.NotificationType_NOTIFY_ON_SERVER_ORIGINATED_RPC
	} else if n.GetBroadcastDomainsChanged() != nil {
		return iterm2.NotificationType_NOTIFY_ON_BROADCAST_CHANGE
	} else if n.GetFocusChangedNotification() != nil {
		return iterm2.NotificationType_NOTIFY_ON_FOCUS_CHANGE
	} else if n.GetKeystrokeNotification() != nil {
		return iterm2.NotificationType_NOTIFY_ON_KEYSTROKE
	} else if n.GetProfileChangedNotification() != nil {
		return iterm2.NotificationType_NOTIFY_ON_PROFILE_CHANGE
	} else if n.GetScreenUpdateNotification() != nil {
		return iterm2.NotificationType_NOTIFY_ON_SCREEN_UPDATE
	} else if n.GetLayoutChangedNotification() != nil {
		return iterm2.NotificationType_NOTIFY_ON_LAYOUT_CHANGE
	} else if n.GetVariableChangedNotification() != nil {
		return iterm2.NotificationType_NOTIFY_ON_VARIABLE_CHANGE
	} else if n.GetPromptNotification() != nil {
		return iterm2.NotificationType_NOTIFY_ON_PROMPT
	}
	return 0
}

func getSubscriptionStatusError(resp *iterm2.ServerOriginatedMessage) error {
	switch resp.GetNotificationResponse().GetStatus() {
	case iterm2.NotificationResponse_SESSION_NOT_FOUND:
		return ErrSessionNotFound
	case iterm2.NotificationResponse_REQUEST_MALFORMED:
		return ErrRequestMalformed
	case iterm2.NotificationResponse_DUPLICATE_SERVER_ORIGINATED_RPC:
		return ErrDuplicatedServerOriginatedRpc
	case iterm2.NotificationResponse_INVALID_IDENTIFIER:
		return ErrInvalidIdentifier
	case iterm2.NotificationResponse_ALREADY_SUBSCRIBED:
		return ErrAlreadySubscribed
	case iterm2.NotificationResponse_NOT_SUBSCRIBED:
		return ErrNotSubscribed
	}
	return nil
}

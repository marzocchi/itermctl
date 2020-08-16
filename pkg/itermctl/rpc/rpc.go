package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"mrz.io/itermctl/pkg/itermctl"
	"mrz.io/itermctl/pkg/itermctl/internal/seq"
	"mrz.io/itermctl/pkg/itermctl/proto"
)

var (
	ErrUnnamedFunc = fmt.Errorf("callback must have a name")
	ErrUnnamedArg  = fmt.Errorf("argument must have a name")
)

// RPC is a function that iTerm2 can invoke in response to some action, such a a keypress or a trigger.
// See https://www.iterm2.com/python-api/registration.html
type RPC struct {
	f    Callback
	name string
	args []Arg
}

// NewRPC returns a new RPC.
func NewRPC(name string, f Callback, args ...Arg) *RPC {
	cb := &RPC{
		name: name,
		args: args,
		f:    f,
	}

	return cb
}

func (rpc *RPC) Invoke(i *Invocation) (interface{}, error) {
	return rpc.f(i)
}

// Arg is used on callback registration to the describe one of the callbacks arguments. If the Arg has only a Name, the
// value has to be provided while calling the RPC. Use Reference to have iTerm2 set the argument value to the value of a
// built in variable at the moment of invocation.
// See https://www.iterm2.com/documentation-variables.html
type Arg struct {
	Name      string
	Reference string
}

// Callback is the implementation of an RPC.
type Callback func(i *Invocation) (interface{}, error)

// Invocation contains the name and arguments of the current invocation of the Callback. Arguments are accessed using
// the Get method or the typed shortcuts GetString and GetFloat64.
type Invocation struct {
	name string
	args map[string]string
}

// NewInvocation creates a new Invocation. The arguments are given as a map of argument name: json-encoded value.
func NewInvocation(name string, args map[string]string) *Invocation {
	return &Invocation{name: name, args: args}
}

// Name returns the name used by iTerm2 for this invocation of an RPC callback.
func (a *Invocation) Name() string {
	return a.name
}

// Get unmarshalls the named argument into the given target.
func (a *Invocation) Get(name string, target interface{}) error {
	data, err := a.getJsonValue(name)

	if err != nil {
		return fmt.Errorf("rpc: %w", err)
	}

	if err := json.Unmarshal([]byte(data), target); err != nil {
		return fmt.Errorf("rpc: %w", err)
	}

	return nil
}

// GetString returns the named argument as a string.
func (a *Invocation) GetString(name string) (string, error) {
	var value string
	if err := a.Get(name, &value); err != nil {
		return "", err
	}

	return value, nil
}

// GetFloat64 returns the named argument as a float64.
func (a *Invocation) GetFloat64(name string) (float64, error) {
	var value float64
	if err := a.Get(name, &value); err != nil {
		return 0, err
	}

	return value, nil
}

func (a *Invocation) getJsonValue(name string) (string, error) {
	if value, ok := a.args[name]; ok {
		return value, nil
	}
	return "", fmt.Errorf("no argument named %q", name)
}

// Register registers the RPC, invokes its Callback and sends its return value or error to iTerm2. Registration lasts
// until the given context is canceled, or the client's connection shuts down.
func Register(ctx context.Context, client *itermctl.Client, rpc *RPC) error {
	if err := validateRpc(rpc); err != nil {
		return err
	}

	subscribe := true
	role := iterm2.RPCRegistrationRequest_GENERIC
	notificationType := iterm2.NotificationType_NOTIFY_ON_SERVER_ORIGINATED_RPC

	req := &iterm2.NotificationRequest{
		Subscribe:        &subscribe,
		NotificationType: &notificationType,
		Arguments: &iterm2.NotificationRequest_RpcRegistrationRequest{
			RpcRegistrationRequest: &iterm2.RPCRegistrationRequest{
				Role:      &role,
				Name:      &rpc.name,
				Arguments: getArgumentsList(rpc.args...),
				Defaults:  getDefaults(rpc.args...),
			},
		},
	}

	recv, err := client.Subscribe(ctx, req)
	if err != nil {
		return fmt.Errorf("register RPC: %s", err)
	}

	handleNotifications(client, rpc, recv)
	return nil
}

func handleNotifications(client *itermctl.Client, callback *RPC, notifications <-chan *iterm2.Notification) {
	go func() {
		for notification := range notifications {
			result := apply(callback, notification.GetServerOriginatedRpcNotification())

			msg := &iterm2.ClientOriginatedMessage{
				Id: seq.MessageId.Next(),
				Submessage: &iterm2.ClientOriginatedMessage_ServerOriginatedRpcResultRequest{
					ServerOriginatedRpcResultRequest: result,
				},
			}

			err := client.Send(msg)
			if err != nil {
				log.Errorf("RPC send: %s", err)
			}
		}
	}()
}

func apply(rpc *RPC, notification *iterm2.ServerOriginatedRPCNotification) *iterm2.ServerOriginatedRPCResultRequest {
	returnValue, returnErr := rpc.Invoke(getInvocationArguments(notification.GetRpc()))

	var result *iterm2.ServerOriginatedRPCResultRequest

	if returnErr == nil {
		result = &iterm2.ServerOriginatedRPCResultRequest{
			RequestId: notification.RequestId,
			Result: &iterm2.ServerOriginatedRPCResultRequest_JsonValue{
				JsonValue: marshalJson(returnValue),
			},
		}
	} else {
		result = &iterm2.ServerOriginatedRPCResultRequest{
			RequestId: notification.RequestId,
			Result: &iterm2.ServerOriginatedRPCResultRequest_JsonException{
				JsonException: marshalJson(map[string]string{"reason": returnErr.Error()}),
			},
		}
	}

	return result
}

func getInvocationArguments(call *iterm2.ServerOriginatedRPC) *Invocation {
	argsMap := make(map[string]string)

	for _, arg := range call.GetArguments() {
		argsMap[arg.GetName()] = arg.GetJsonValue()
	}

	return NewInvocation(call.GetName(), argsMap)
}

func validateRpc(c *RPC) error {
	if c.name == "" {
		return ErrUnnamedFunc
	}

	for _, arg := range c.args {
		if arg.Name == "" {
			return ErrUnnamedArg
		}
	}

	return nil
}

func getArgumentsList(args ...Arg) (reqArgs []*iterm2.RPCRegistrationRequest_RPCArgumentSignature) {
	for _, arg := range args {
		reqArgs = append(
			reqArgs,
			func(arg Arg) *iterm2.RPCRegistrationRequest_RPCArgumentSignature {
				return &iterm2.RPCRegistrationRequest_RPCArgumentSignature{Name: &arg.Name}
			}(arg),
		)
	}
	return
}

func getDefaults(args ...Arg) (reqDefaults []*iterm2.RPCRegistrationRequest_RPCArgument) {
	for _, arg := range args {
		if arg.Reference == "" {
			continue
		}
		reqDefaults = append(
			reqDefaults,
			func(arg Arg) *iterm2.RPCRegistrationRequest_RPCArgument {
				return &iterm2.RPCRegistrationRequest_RPCArgument{
					Name: &arg.Name,
					Path: &arg.Reference,
				}
			}(arg))
	}
	return
}

func marshalJson(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}

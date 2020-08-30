package itermctl

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"mrz.io/itermctl/pkg/itermctl/internal/json"
	"mrz.io/itermctl/pkg/itermctl/proto"
	"reflect"
	"strings"
)

// Rpc is a function that can be registered with iTerm2 using RegisterRpc and will be invoked in response to some
// action or event, such as a keypress or a trigger.
type Rpc struct {
	// Name is the function's name and makes up the function signature, together with Args.
	Name string

	// Args define the function's expected arguments, given as a struct or *struct. Only fields of type bool, string and
	// float64 are considered, while fields of other types will be ignored. Each field can also be annotated with
	// arg.name and arg.ref to provide the argument's name and a default value as a reference to an iTerm2's built-in
	// variable. If there's no arg.name tag, the value of the json tag or the struct field in lower case are used as a
	// fallback.
	// See https://www.iterm2.com/documentation-variables.html.
	Args interface{}

	// F is the Rpc function implementation.
	F RpcFunc
}

// RpcFunc is the implementation of an Rpc function.
type RpcFunc func(invocation *RpcInvocation) (interface{}, error)

type ClickArgs struct {
	SessionId string `arg.name:"session_id"`
}

type statusBarComponentIdentifierValueKey string

// See https://iterm2.com/python-api/statusbar.html.
type StatusBarComponent struct {
	// ShortDescription is shown below the component in the picker UI.
	ShortDescription string

	// Description is used in the tool tip for the component in the picker UI.
	Description string

	// Exemplar is the sample content of the component shown in the picker UI.
	Exemplar string

	// UpdateCadence defines how frequently iTerm2 should invoke the component's Rpc. Zero disables updates.
	UpdateCadence float32

	// Identifier is the unique identifier for the component. Use reverse domain name notation.
	Identifier string

	// Knobs defines the custom controls in the component's configuration panel, as a struct or *struct. Fields of
	// type string, bool and float64 are used in order to create StringKnobs, CheckboxKnob and
	// PositiveFloatingPointKnob, other field types are ignored. The field's name is used as the knob's name and
	// placeholder unless tagged with knob.name and knob.placeholder. The knob's key is set using the `json` tag.
	// TODO support for color knobs.
	Knobs interface{}

	// Rpc is the implementation of the component.
	Rpc Rpc

	OnClick RpcFunc
}

// TitleProvider is an Rpc that gets called to compute the title of a session, as frequently as iTerm2 deems necessary,
// for example when any argument that is a reference to a variable in the session's context changes.
// See https://iterm2.com/python-api/registration.html#iterm2.registration.TitleProviderRPC.
type TitleProvider struct {
	// DisplayName is shown in the title provider's drop down in a profile's preference panel.
	DisplayName string

	// Identifier is the unique identifier for the provider. Use reverse domain name notation.
	Identifier string

	// Rpc is the implementation of the provider.
	Rpc Rpc
}

// RpcInvocation contains all the arguments of the current invocation of a RpcFunc.
type RpcInvocation struct {
	requestId string
	conn      *Connection
	name      string
	args      map[string]string

	statusBarComponentIdentifier string
}

// Name gives the name of used by iTerm2 to invoke this callback's parent Rpc.
func (inv *RpcInvocation) Name() string {
	return inv.name
}

// Args unmarshalls the invocation arguments into the given *struct, usually the same one used as the Rpc's arguments.
func (inv *RpcInvocation) Args(target interface{}) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() == reflect.Ptr {
		targetValue = targetValue.Elem()
	}
	targetType := targetValue.Type()

	if targetValue.NumField() < 1 {
		return nil
	}

	for i := 0; i < targetValue.NumField(); i++ {
		f := targetValue.Field(i)

		name, err := getFirstNamedTag(targetType.Field(i).Tag, "arg.name")
		if err != nil || name == "" {
			name = strings.ToLower(targetType.Field(i).Name)
		}

		var argJsonValue string
		var ok bool

		if argJsonValue, ok = inv.args[name]; !ok {
			continue
		}

		switch {
		case f.Kind() == reflect.Bool:
			var value bool
			if err := json.UnmarshalString(argJsonValue, &value); err != nil {
				return fmt.Errorf("cannot unmarshal %q: %w", name, err)
			}
			f.SetBool(value)
		case f.Kind() == reflect.String:
			var value string
			if err := json.UnmarshalString(argJsonValue, &value); err != nil {
				return fmt.Errorf("cannot unmarshal %q: %w", name, err)
			}
			f.SetString(value)
		case f.Kind() == reflect.Float64:
			var value float64
			if err := json.UnmarshalString(argJsonValue, &value); err != nil {
				return fmt.Errorf("cannot unmarshal %q: %w", name, err)
			}
			f.SetFloat(value)
		}
	}
	return nil
}

// Knobs unmarshalls the invocation's knobs into the given *struct, usually the same one used as the
// StatusBarComponent's knobs. Returns ErrNoKnobs when the 'knobs' argument is not found in this invocation (which
// usually means the Rpc was not registered as a StatusBarComponent).
func (inv *RpcInvocation) Knobs(target interface{}) error {
	knobs, ok := inv.args["knobs"]
	if !ok {
		return ErrNoKnobs
	}

	if err := json.UnmarshalTwice(knobs, target); err != nil {
		return fmt.Errorf("knobs: %w", err)
	}

	return nil
}

func (inv *RpcInvocation) OpenPopover(html string, width, height int32) error {
	args := ClickArgs{}
	if err := inv.Args(&args); err != nil {
		return fmt.Errorf("popover: can't get session ID: %w", err)
	}

	msg := &iterm2.ClientOriginatedMessage{
		Submessage: &iterm2.ClientOriginatedMessage_StatusBarComponentRequest{
			StatusBarComponentRequest: &iterm2.StatusBarComponentRequest{
				Request: &iterm2.StatusBarComponentRequest_OpenPopover_{
					OpenPopover: &iterm2.StatusBarComponentRequest_OpenPopover{
						SessionId: &args.SessionId,
						Html:      &html,
						Size: &iterm2.Size{
							Width:  &width,
							Height: &height,
						},
					},
				},
				Identifier: &inv.statusBarComponentIdentifier,
			},
		},
	}

	if err := inv.conn.Send(msg); err != nil {
		return fmt.Errorf("popover: %w", err)
	}
	return nil
}

func acceptRpc(rpc Rpc) AcceptFunc {
	return func(msg *iterm2.ServerOriginatedMessage) bool {
		if notification := msg.GetNotification(); notification != nil {
			if rpcNotification := notification.GetServerOriginatedRpcNotification(); rpcNotification != nil {
				if rpcNotification.GetRpc() != nil {
					return rpcNotification.GetRpc().GetName() == rpc.Name
				}
			}
		}
		return false
	}
}

// RegisterRpc registers the Rpc, invokes its callback when requested by iTerm2 and writes back to iTerm2 the callback's
// return value or return error. Registration lasts until the context is canceled, or the underlying connection is
// closed.
// See https://www.iterm2.com/python-api/registration.html.
func (conn *Connection) RegisterRpc(ctx context.Context, rpc Rpc) error {
	role := iterm2.RPCRegistrationRequest_GENERIC

	req := newRegistrationRequest(role, rpc)

	recv, err := conn.Subscribe(ctx, req)
	if err != nil {
		return fmt.Errorf("register rpc: %s", err)
	}

	recv.SetName(fmt.Sprintf("receive rpc: %s", rpc.Name))
	recv.SetAcceptFunc(acceptRpc(rpc))

	go func() {
		for msg := range recv.Ch() {
			rpcNotification := msg.GetNotification().GetServerOriginatedRpcNotification()
			args := getInvocationArguments(ctx, conn, rpcNotification)
			invoke(conn, rpc, args)
		}
	}()

	return nil
}

// RegisterStatusBarComponent registers a Status Bar Component. Registration lasts until the context is canceled or
// the connection is closed.
// See https://www.iterm2.com/python-api/registration.html#iterm2.registration.StatusBarRPC.
func (conn *Connection) RegisterStatusBarComponent(ctx context.Context, cmp StatusBarComponent) error {
	var cadence *float32
	knobs := "knobs"

	if cmp.UpdateCadence > 0 {
		cadence = &cmp.UpdateCadence
	}

	req := newRegistrationRequest(iterm2.RPCRegistrationRequest_STATUS_BAR_COMPONENT, cmp.Rpc)

	args := req.GetRpcRegistrationRequest().GetArguments()
	args = append(args, &iterm2.RPCRegistrationRequest_RPCArgumentSignature{Name: &knobs})
	req.GetRpcRegistrationRequest().Arguments = args

	req.GetRpcRegistrationRequest().RoleSpecificAttributes = &iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_{
		StatusBarComponentAttributes: &iterm2.RPCRegistrationRequest_StatusBarComponentAttributes{
			ShortDescription:    &cmp.ShortDescription,
			DetailedDescription: &cmp.Description,
			Exemplar:            &cmp.Exemplar,
			UniqueIdentifier:    &cmp.Identifier,
			UpdateCadence:       cadence,
			Knobs:               getKnobs(cmp.Knobs),
			Icons:               nil,
		},
	}

	recv, err := conn.Subscribe(ctx, req)
	if err != nil {
		return fmt.Errorf("register status bar component: %w", err)
	}

	recv.SetName(fmt.Sprintf("receive SBC %s, rpc: %s", cmp.Identifier, cmp.Rpc.Name))
	recv.SetAcceptFunc(acceptRpc(cmp.Rpc))

	go func() {
		for msg := range recv.Ch() {
			rpcNotification := msg.GetNotification().GetServerOriginatedRpcNotification()
			args := getInvocationArguments(ctx, conn, rpcNotification)
			args.statusBarComponentIdentifier = cmp.Identifier
			invoke(conn, cmp.Rpc, args)
		}
	}()

	if cmp.OnClick != nil {
		if err := conn.registerClickHandler(ctx, cmp); err != nil {
			return err
		}
	}

	return nil
}

func (conn *Connection) registerClickHandler(ctx context.Context, cmp StatusBarComponent) error {
	clickRpc := Rpc{
		Name: fmt.Sprintf("__%s__on_click",
			strings.Replace(strings.Replace(cmp.Identifier, ".", "_", -1), "-", "_", -1)),
		Args: ClickArgs{},
		F:    cmp.OnClick,
	}

	ctx = context.WithValue(ctx, statusBarComponentIdentifierValueKey("identifier"), cmp.Identifier)

	if err := conn.RegisterRpc(ctx, clickRpc); err != nil {
		return fmt.Errorf("register click handler: %w", err)
	}
	return nil
}

// RegisterSessionTitleProvider registers a Session Title Provider. Registration lasts until the context is  canceled
// or the conn's connection shuts down.
// See https://www.iterm2.com/python-api/registration.html#iterm2.registration.TitleProviderRPC.
func (conn *Connection) RegisterSessionTitleProvider(ctx context.Context, tp TitleProvider) error {
	req := newRegistrationRequest(iterm2.RPCRegistrationRequest_SESSION_TITLE, tp.Rpc)

	req.GetRpcRegistrationRequest().RoleSpecificAttributes = &iterm2.RPCRegistrationRequest_SessionTitleAttributes_{
		SessionTitleAttributes: &iterm2.RPCRegistrationRequest_SessionTitleAttributes{
			DisplayName:      &tp.DisplayName,
			UniqueIdentifier: &tp.Identifier,
		},
	}
	recv, err := conn.Subscribe(ctx, req)
	if err != nil {
		return fmt.Errorf("register title provider: %s", err)
	}

	recv.SetName(fmt.Sprintf("receive TP %s, rpc: %s", tp.Identifier, tp.Rpc.Name))
	recv.SetAcceptFunc(acceptRpc(tp.Rpc))

	go func() {
		for msg := range recv.Ch() {
			rpcNotification := msg.GetNotification().GetServerOriginatedRpcNotification()
			args := getInvocationArguments(ctx, conn, rpcNotification)
			invoke(conn, tp.Rpc, args)
		}
	}()

	return nil
}

func newRegistrationRequest(role iterm2.RPCRegistrationRequest_Role, rpc Rpc) *iterm2.NotificationRequest {
	subscribe := true
	notificationType := iterm2.NotificationType_NOTIFY_ON_SERVER_ORIGINATED_RPC

	args, defaults := getArgs(rpc.Args)

	return &iterm2.NotificationRequest{
		Subscribe:        &subscribe,
		NotificationType: &notificationType,
		Arguments: &iterm2.NotificationRequest_RpcRegistrationRequest{
			RpcRegistrationRequest: &iterm2.RPCRegistrationRequest{
				Role:      &role,
				Name:      &rpc.Name,
				Arguments: args,
				Defaults:  defaults,
			},
		},
	}
}

func getKnobs(v interface{}) []*iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_Knob {
	var knobs []*iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_Knob

	value := reflect.ValueOf(v)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	tt := value.Type()

	if value.NumField() < 1 {
		return nil
	}

	for i := 0; i < value.NumField(); i++ {
		f := value.Field(i)

		var knobType iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_Knob_Type
		var defaultJson string

		name, err := getFirstNamedTag(tt.Field(i).Tag, "knob.name")
		if err != nil || name == "" {
			name = tt.Field(i).Name
		}

		placeholder, err := getFirstNamedTag(tt.Field(i).Tag, "knob.placeholder")
		if err != nil || placeholder == "" {
			placeholder = name
		}

		key, err := getFirstNamedTag(tt.Field(i).Tag, "json")
		if err != nil || key == "" {
			key = strings.ToLower(tt.Field(i).Name)
		}

		switch {
		case f.Kind() == reflect.Bool:
			knobType = iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_Knob_String
			defaultJson = json.MustMarshal(f.Bool())
		case f.Kind() == reflect.String:
			knobType = iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_Knob_String
			defaultJson = json.MustMarshal(f.String())
		case f.Kind() == reflect.Float64:
			knobType = iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_Knob_PositiveFloatingPoint
			defaultJson = json.MustMarshal(f.Float())
		default:
			continue
		}

		knobs = append(knobs, &iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_Knob{
			Type:             &knobType,
			Name:             &name,
			Key:              &key,
			Placeholder:      &placeholder,
			JsonDefaultValue: &defaultJson,
		})
	}

	return knobs
}

func getFirstNamedTag(tag reflect.StructTag, tagNames ...string) (string, error) {
	for _, name := range tagNames {
		value := tag.Get(name)
		if value != "" {
			return value, nil
		}
	}
	return "", fmt.Errorf("none of the tags has an usable value: %s", strings.Join(tagNames, ", "))
}

func getInvocationArguments(ctx context.Context, conn *Connection, rpcNotification *iterm2.ServerOriginatedRPCNotification) *RpcInvocation {
	argsMap := make(map[string]string)

	for _, arg := range rpcNotification.GetRpc().GetArguments() {
		argsMap[arg.GetName()] = arg.GetJsonValue()
	}

	invocation := &RpcInvocation{
		conn:      conn,
		name:      rpcNotification.GetRpc().GetName(),
		args:      argsMap,
		requestId: rpcNotification.GetRequestId(),
	}

	if v := ctx.Value(statusBarComponentIdentifierValueKey("identifier")); v != nil {
		invocation.statusBarComponentIdentifier = v.(string)
	}

	return invocation
}

func getArgs(v interface{}) (arguments []*iterm2.RPCRegistrationRequest_RPCArgumentSignature, defaults []*iterm2.RPCRegistrationRequest_RPCArgument) {
	if v == nil {
		return
	}

	value := reflect.ValueOf(v)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	tt := value.Type()

	if value.NumField() < 1 {
		return
	}

	for i := 0; i < value.NumField(); i++ {
		name, err := getFirstNamedTag(tt.Field(i).Tag, "arg.name", "json")
		if err != nil || name == "" {
			name = strings.ToLower(tt.Field(i).Name)
		}

		arguments = append(arguments, &iterm2.RPCRegistrationRequest_RPCArgumentSignature{Name: &name})

		ref, err := getFirstNamedTag(tt.Field(i).Tag, "arg.ref")

		if err != nil || ref == "" {
			continue
		}

		defaults = append(defaults, &iterm2.RPCRegistrationRequest_RPCArgument{
			Name: &name,
			Path: &ref,
		})
	}
	return
}

func invoke(conn *Connection, rpc Rpc, args *RpcInvocation) {
	returnValue, returnErr := rpc.F(args)

	var result *iterm2.ServerOriginatedRPCResultRequest

	if returnErr == nil {
		result = &iterm2.ServerOriginatedRPCResultRequest{
			RequestId: &args.requestId,
			Result: &iterm2.ServerOriginatedRPCResultRequest_JsonValue{
				JsonValue: json.MustMarshal(returnValue),
			},
		}
	} else {
		result = &iterm2.ServerOriginatedRPCResultRequest{
			RequestId: &args.requestId,
			Result: &iterm2.ServerOriginatedRPCResultRequest_JsonException{
				JsonException: json.MustMarshal(map[string]string{"reason": returnErr.Error()}),
			},
		}
	}

	msg := &iterm2.ClientOriginatedMessage{
		Submessage: &iterm2.ClientOriginatedMessage_ServerOriginatedRpcResultRequest{
			ServerOriginatedRpcResultRequest: result,
		},
	}

	err := conn.Send(msg)
	if err != nil {
		logrus.Errorf("RpcFunc send: %s", err)
	}
}

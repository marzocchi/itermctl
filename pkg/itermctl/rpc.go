package itermctl

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"mrz.io/itermctl/pkg/itermctl/internal/seq"
	"mrz.io/itermctl/pkg/itermctl/proto"
	"reflect"
	"strings"
)

var ErrNoKnobs = fmt.Errorf("no argument named 'knobs'")

// Rpc is a function that iTerm2 can invoke in response to some action, such as a keypress or a trigger, after it has
// been registered with iTerm2 using RegisterRpc.
type Rpc struct {
	// Name is the function's name and makes up the function signature, together with Args.
	Name string

	// Args define the function's expected arguments, given as a struct or *struct. Only fields of type bool, string and
	// float64 are considered, while fields of other types will be ignored. Each field can also be annotated to provide
	// the argument's name and a default value as a reference to an iTerm2's built-in variable.
	// See https://www.iterm2.com/documentation-variables.html.
	Args interface{}

	// F is the Rpc implementation.
	F RpcFunc
}

// RpcFunc is the implementation of an Rpc function.
type RpcFunc func(invocation *RpcInvocation) (interface{}, error)

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
	args map[string]string
}

// Args unmarshalls the invocation arguments into the given *struct, usually the same one used as the Rpc's arguments.
func (inv *RpcInvocation) Args(target interface{}) error {
	targetValue := reflect.ValueOf(target).Elem()
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
			if err := json.Unmarshal([]byte(argJsonValue), &value); err != nil {
				return fmt.Errorf("cannot unmarshal %q: %w", name, err)
			}
			f.SetBool(value)
		case f.Kind() == reflect.String:
			var value string
			if err := json.Unmarshal([]byte(argJsonValue), &value); err != nil {
				return fmt.Errorf("cannot unmarshal %q: %w", name, err)
			}
			f.SetString(value)
		case f.Kind() == reflect.Float64:
			var value float64
			if err := json.Unmarshal([]byte(argJsonValue), &value); err != nil {
				return fmt.Errorf("cannot unmarshal %q: %w", name, err)
			}
			f.SetFloat(value)
		}
	}
	return nil
}

// Knobs unmarshalls the invocation's knobs into the given *struct, usually the same one used as the
// StatusBarComponent's knobs. Returns ErrNoKnobs when the 'knobs' argument is not found in this invocation (because
// the Rpc was not registered as a StatusBarComponent).
func (inv *RpcInvocation) Knobs(target interface{}) error {
	knobs, ok := inv.args["knobs"]
	if !ok {
		return ErrNoKnobs
	}

	var intermediate string
	if err := json.Unmarshal([]byte(knobs), &intermediate); err != nil {
		return fmt.Errorf("knobs: %w", err)
	}

	if err := json.Unmarshal([]byte(intermediate), target); err != nil {
		return fmt.Errorf("knobs: %w", err)
	}

	return nil
}

// RegisterRpc registers the Rpc, invokes its callback when requested by iTerm2 and sends back its return value.
// Registration lasts until the given context is canceled, or the underlying connection shuts down.
func RegisterRpc(ctx context.Context, client *Client, rpc Rpc) error {
	args, defaults := getArgs(rpc.Args)

	subscribe := true
	role := iterm2.RPCRegistrationRequest_GENERIC
	notificationType := iterm2.NotificationType_NOTIFY_ON_SERVER_ORIGINATED_RPC

	req := &iterm2.NotificationRequest{
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

	invocations, err := client.Subscribe(ctx, req)
	if err != nil {
		return fmt.Errorf("register RpcFunc: %s", err)
	}

	handleInvocations(client, rpc.F, invocations)
	return nil
}

func RegisterStatusBarComponent(ctx context.Context, client *Client, cmp StatusBarComponent) error {
	knobs := "knobs"
	subscribe := true
	role := iterm2.RPCRegistrationRequest_STATUS_BAR_COMPONENT
	notificationType := iterm2.NotificationType_NOTIFY_ON_SERVER_ORIGINATED_RPC

	arguments, defaults := getArgs(cmp.Rpc.Args)
	arguments = append(arguments, &iterm2.RPCRegistrationRequest_RPCArgumentSignature{Name: &knobs})
	knobsList := getKnobs(cmp.Knobs)
	var cadence *float32

	if cmp.UpdateCadence > 0 {
		cadence = &cmp.UpdateCadence
	}

	req := &iterm2.NotificationRequest{
		Subscribe:        &subscribe,
		NotificationType: &notificationType,
		Arguments: &iterm2.NotificationRequest_RpcRegistrationRequest{
			RpcRegistrationRequest: &iterm2.RPCRegistrationRequest{
				Name:      &cmp.Rpc.Name,
				Arguments: arguments,
				Defaults:  defaults,
				Role:      &role,
				RoleSpecificAttributes: &iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_{
					StatusBarComponentAttributes: &iterm2.RPCRegistrationRequest_StatusBarComponentAttributes{
						ShortDescription:    &cmp.ShortDescription,
						DetailedDescription: &cmp.Description,
						Exemplar:            &cmp.Exemplar,
						UpdateCadence:       cadence,
						UniqueIdentifier:    &cmp.Identifier,
						Icons:               nil,
						Knobs:               knobsList,
					},
				},
			},
		},
	}

	recv, err := client.Subscribe(ctx, req)
	if err != nil {
		return fmt.Errorf("register status bar component: %w", err)
	}

	handleInvocations(client, cmp.Rpc.F, recv)
	return nil
}

// RegisterSessionTitleProvider registers a Session Title Provider. Registration lasts until the given context is
// canceled, or the client's connection shuts down.
func RegisterSessionTitleProvider(ctx context.Context, client *Client, tp TitleProvider) error {
	subscribe := true
	role := iterm2.RPCRegistrationRequest_SESSION_TITLE
	notificationType := iterm2.NotificationType_NOTIFY_ON_SERVER_ORIGINATED_RPC

	arguments, defaults := getArgs(tp.Rpc.Args)

	req := &iterm2.NotificationRequest{
		Subscribe:        &subscribe,
		NotificationType: &notificationType,
		Arguments: &iterm2.NotificationRequest_RpcRegistrationRequest{
			RpcRegistrationRequest: &iterm2.RPCRegistrationRequest{
				RoleSpecificAttributes: &iterm2.RPCRegistrationRequest_SessionTitleAttributes_{
					SessionTitleAttributes: &iterm2.RPCRegistrationRequest_SessionTitleAttributes{
						DisplayName:      &tp.DisplayName,
						UniqueIdentifier: &tp.Identifier,
					},
				},
				Role:      &role,
				Name:      &tp.Rpc.Name,
				Arguments: arguments,
				Defaults:  defaults,
			},
		},
	}

	recv, err := client.Subscribe(ctx, req)
	if err != nil {
		return fmt.Errorf("register RpcFunc: %s", err)
	}

	handleInvocations(client, tp.Rpc.F, recv)
	return nil
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
			defaultJson = asJsonString(f.Bool())
		case f.Kind() == reflect.String:
			knobType = iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_Knob_String
			defaultJson = asJsonString(f.String())
		case f.Kind() == reflect.Float64:
			knobType = iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_Knob_PositiveFloatingPoint
			defaultJson = asJsonString(f.Float())
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

func asJsonString(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
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

func getInvocationArguments(call *iterm2.ServerOriginatedRPC) *RpcInvocation {
	argsMap := make(map[string]string)

	for _, arg := range call.GetArguments() {
		argsMap[arg.GetName()] = arg.GetJsonValue()
	}

	return &RpcInvocation{args: argsMap}
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

func handleInvocations(client *Client, callback RpcFunc, invocations <-chan *iterm2.Notification) {
	go func() {
		for invocation := range invocations {
			result := invoke(callback, invocation.GetServerOriginatedRpcNotification())

			msg := &iterm2.ClientOriginatedMessage{
				Id: seq.MessageId.Next(),
				Submessage: &iterm2.ClientOriginatedMessage_ServerOriginatedRpcResultRequest{
					ServerOriginatedRpcResultRequest: result,
				},
			}

			err := client.Send(msg)
			if err != nil {
				logrus.Errorf("RpcFunc send: %s", err)
			}
		}
	}()
}

func invoke(callback RpcFunc, invocation *iterm2.ServerOriginatedRPCNotification) *iterm2.ServerOriginatedRPCResultRequest {
	returnValue, returnErr := callback(getInvocationArguments(invocation.GetRpc()))

	var result *iterm2.ServerOriginatedRPCResultRequest

	if returnErr == nil {
		result = &iterm2.ServerOriginatedRPCResultRequest{
			RequestId: invocation.RequestId,
			Result: &iterm2.ServerOriginatedRPCResultRequest_JsonValue{
				JsonValue: asJsonString(returnValue),
			},
		}
	} else {
		result = &iterm2.ServerOriginatedRPCResultRequest{
			RequestId: invocation.RequestId,
			Result: &iterm2.ServerOriginatedRPCResultRequest_JsonException{
				JsonException: asJsonString(map[string]string{"reason": returnErr.Error()}),
			},
		}
	}

	return result
}

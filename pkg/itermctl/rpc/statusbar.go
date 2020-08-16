package rpc

import (
	"context"
	"fmt"
	"mrz.io/itermctl/pkg/itermctl"
	"mrz.io/itermctl/pkg/itermctl/proto"
)

type Knob interface {
	Name() string
	toProto() *iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_Knob
}

// TODO
// type ColorKnob struct {
// }

type PositiveFloatingPointKnob struct {
	name         string
	key          string
	defaultValue float64
}

func NewPositiveFloatingPointKnob(name string, key string, defaultValue float64) *PositiveFloatingPointKnob {
	return &PositiveFloatingPointKnob{name: name, key: key, defaultValue: defaultValue}
}

func (p *PositiveFloatingPointKnob) Name() string {
	return p.name
}

func (p *PositiveFloatingPointKnob) toProto() *iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_Knob {
	t := iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_Knob_PositiveFloatingPoint

	defaultJson := marshalJson(p.defaultValue)
	return &iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_Knob{
		Type:             &t,
		Name:             &p.name,
		Key:              &p.key,
		Placeholder:      &p.name,
		JsonDefaultValue: &defaultJson,
	}
}

type StringKnob struct {
	name         string
	key          string
	defaultValue string
}

func NewStringKnob(name string, key string, defaultValue string) *StringKnob {
	return &StringKnob{name: name, key: key, defaultValue: defaultValue}

}
func (s *StringKnob) Name() string {
	return s.name
}

func (s *StringKnob) toProto() *iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_Knob {
	t := iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_Knob_String

	defaultJson := marshalJson(s.defaultValue)
	return &iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_Knob{
		Type:             &t,
		Name:             &s.name,
		Key:              &s.key,
		Placeholder:      &s.name,
		JsonDefaultValue: &defaultJson,
	}
}

type CheckboxKnob struct {
	name         string
	key          string
	defaultValue bool
}

func NewCheckboxKnob(name string, key string, defaultValue bool) *CheckboxKnob {
	return &CheckboxKnob{name: name, key: key, defaultValue: defaultValue}
}

func (cb *CheckboxKnob) Name() string {
	return cb.name
}

func (cb *CheckboxKnob) toProto() *iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_Knob {
	t := iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_Knob_Checkbox

	defaultJson := marshalJson(cb.defaultValue)
	return &iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_Knob{
		Type:             &t,
		Name:             &cb.name,
		Key:              &cb.key,
		Placeholder:      &cb.name,
		JsonDefaultValue: &defaultJson,
	}
}

type StatusBarComponent struct {
	ShortDescription string
	Description      string
	Exemplar         string
	UpdateCadence    float32
	Identifier       string
	Knobs            []Knob
	RPC              *RPC
}

func RegisterStatusBarComponent(ctx context.Context, client *itermctl.Client, cmp StatusBarComponent) error {
	rpc := cmp.RPC
	rpc.args = []Arg{{
		Name: "knobs",
	}}

	if err := validateRpc(rpc); err != nil {
		return err
	}

	var knobs []*iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_Knob
	for _, k := range cmp.Knobs {
		knobs = append(knobs, k.toProto())
	}

	subscribe := true
	role := iterm2.RPCRegistrationRequest_STATUS_BAR_COMPONENT
	notificationType := iterm2.NotificationType_NOTIFY_ON_SERVER_ORIGINATED_RPC

	req := &iterm2.NotificationRequest{
		Subscribe:        &subscribe,
		NotificationType: &notificationType,
		Arguments: &iterm2.NotificationRequest_RpcRegistrationRequest{
			RpcRegistrationRequest: &iterm2.RPCRegistrationRequest{
				Name:      &rpc.name,
				Arguments: getArgumentsList(rpc.args...),
				Defaults:  getDefaults(rpc.args...),
				Role:      &role,
				RoleSpecificAttributes: &iterm2.RPCRegistrationRequest_StatusBarComponentAttributes_{
					StatusBarComponentAttributes: &iterm2.RPCRegistrationRequest_StatusBarComponentAttributes{
						ShortDescription:    &cmp.ShortDescription,
						DetailedDescription: &cmp.Description,
						Exemplar:            &cmp.Exemplar,
						UpdateCadence:       &cmp.UpdateCadence,
						UniqueIdentifier:    &cmp.Identifier,
						Icons:               nil,
						Knobs:               knobs,
					},
				},
			},
		},
	}

	recv, err := client.Subscribe(ctx, req)
	if err != nil {
		return fmt.Errorf("register status bar component: %w", err)
	}

	handleNotifications(client, rpc, recv)
	return nil
}

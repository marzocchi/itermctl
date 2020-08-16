package rpc

import (
	"context"
	"fmt"
	"mrz.io/itermctl/pkg/itermctl"
	iterm2 "mrz.io/itermctl/pkg/itermctl/proto"
)

type TitleProvider struct {
	DisplayName string
	Identifier  string
	RPC         *RPC
}

// RegisterSessionTitleProvider registers a Session Title Provider. Registration lasts until the given context is
// canceled, or the client's connection shuts down.
func RegisterSessionTitleProvider(ctx context.Context, client *itermctl.Client, tp TitleProvider) error {
	if err := validateRpc(tp.RPC); err != nil {
		return err
	}

	subscribe := true
	role := iterm2.RPCRegistrationRequest_SESSION_TITLE
	notificationType := iterm2.NotificationType_NOTIFY_ON_SERVER_ORIGINATED_RPC

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
				Name:      &tp.RPC.name,
				Arguments: getArgumentsList(tp.RPC.args...),
				Defaults:  getDefaults(tp.RPC.args...),
			},
		},
	}

	recv, err := client.Subscribe(ctx, req)
	if err != nil {
		return fmt.Errorf("register RPC: %s", err)
	}

	handleNotifications(client, tp.RPC, recv)
	return nil
}

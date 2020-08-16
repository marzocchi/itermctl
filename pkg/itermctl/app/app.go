package app

import (
	"encoding/json"
	"fmt"
	"mrz.io/itermctl/pkg/itermctl/internal/seq"
	"mrz.io/itermctl/pkg/itermctl/proto"
)

const DefaultProfileName = "Default"

type Client interface {
	GetResponse(req *iterm2.ClientOriginatedMessage) (*iterm2.ServerOriginatedMessage, error)
}

// App provides methods to interact with the running iTerm2 Application.
type App struct {
	client Client
}

// NewApp creates a new App bound to the given Client.
func NewApp(client Client) *App {
	return &App{client: client}
}

func (a *App) InvokeFunction(invocation string, result interface{}) error {
	req := &iterm2.ClientOriginatedMessage{
		Id: seq.MessageId.Next(),
		Submessage: &iterm2.ClientOriginatedMessage_InvokeFunctionRequest{
			InvokeFunctionRequest: &iterm2.InvokeFunctionRequest{
				Context:    &iterm2.InvokeFunctionRequest_App_{},
				Invocation: &invocation,
			},
		},
	}

	resp, err := a.client.GetResponse(req)
	if err != nil {
		return err
	}

	if invocationErr := resp.GetInvokeFunctionResponse().GetError(); invocationErr != nil {
		return fmt.Errorf("%s: %s", invocationErr.GetStatus(), invocationErr.GetErrorReason())
	}

	jsonResult := resp.GetInvokeFunctionResponse().GetSuccess().GetJsonResult()

	if err := json.Unmarshal([]byte(jsonResult), &result); err != nil {
		return fmt.Errorf("could not unmarshal invocation result: %w", err)
	}

	return nil
}

func (a *App) CloseTab(force bool, tabIds ...int32) error {
	var tabIdsStrings []string
	for _, tabId := range tabIds {
		tabIdsStrings = append(tabIdsStrings, fmt.Sprintf("%d", tabId))
	}

	return a.close(&iterm2.CloseRequest{
		Target: &iterm2.CloseRequest_Tabs{
			Tabs: &iterm2.CloseRequest_CloseTabs{TabIds: tabIdsStrings},
		},
		Force: &force,
	})
}

func (a *App) CloseSession(force bool, sessionIds ...string) error {
	return a.close(&iterm2.CloseRequest{
		Target: &iterm2.CloseRequest_Sessions{
			Sessions: &iterm2.CloseRequest_CloseSessions{SessionIds: sessionIds},
		},
		Force: &force,
	})
}

func (a *App) CloseWindow(force bool, windowIds ...string) error {
	return a.close(&iterm2.CloseRequest{
		Target: &iterm2.CloseRequest_Windows{
			Windows: &iterm2.CloseRequest_CloseWindows{WindowIds: windowIds},
		},
		Force: &force,
	})
}

// CreateTab creates a new tab in the targeted window, at the specified index, with the Default or named profile.
func (a *App) CreateTab(windowId string, tabIndex uint32, profileName string) (*iterm2.CreateTabResponse, error) {
	if profileName == "" {
		profileName = DefaultProfileName
	}

	createReq := &iterm2.CreateTabRequest{}
	createReq.TabIndex = &tabIndex

	if windowId != "" {
		createReq.WindowId = &windowId
	}

	if profileName != "" {
		createReq.ProfileName = &profileName
	}

	req := &iterm2.ClientOriginatedMessage{
		Id: seq.MessageId.Next(),
		Submessage: &iterm2.ClientOriginatedMessage_CreateTabRequest{
			CreateTabRequest: createReq,
		},
	}

	resp, err := a.client.GetResponse(req)
	if err != nil {
		return nil, fmt.Errorf("create tab: %w", err)
	}

	var returnErr error

	if resp.GetCreateTabResponse().GetStatus() != iterm2.CreateTabResponse_OK {
		returnErr = fmt.Errorf("create tab: %s", resp.GetCreateTabResponse().GetStatus())
	}

	return resp.GetCreateTabResponse(), returnErr
}

func (a *App) close(cr *iterm2.CloseRequest) error {
	req := &iterm2.ClientOriginatedMessage{
		Id: seq.MessageId.Next(),
		Submessage: &iterm2.ClientOriginatedMessage_CloseRequest{
			CloseRequest: cr,
		},
	}

	resp, err := a.client.GetResponse(req)
	if err != nil {
		return fmt.Errorf("close: %w", err)
	}

	for _, s := range resp.GetCloseResponse().GetStatuses() {
		if s != iterm2.CloseResponse_OK {
			return fmt.Errorf("close: %s", s.String())
		}
	}

	return nil
}

// GetFocus returns the state of every tabs and windows as multiple FocusChangedNotifications.
func (a *App) GetFocus() ([]*iterm2.FocusChangedNotification, error) {
	req := &iterm2.ClientOriginatedMessage{
		Id: seq.MessageId.Next(),
		Submessage: &iterm2.ClientOriginatedMessage_FocusRequest{
			FocusRequest: &iterm2.FocusRequest{},
		},
	}

	resp, err := a.client.GetResponse(req)
	if err != nil {
		return nil, fmt.Errorf("get focus: %w", err)
	}

	return resp.GetFocusResponse().GetNotifications(), nil
}

// GetActiveSessionId returns the ID of the currently active session through a GetFocus call.
func (a *App) GetActiveSessionId() (string, error) {
	resp, err := a.GetFocus()
	if err != nil {
		return "", fmt.Errorf("get active session ID: %w", err)
	}

	for _, n := range resp {
		if n.GetSession() != "" {
			return n.GetSession(), nil
		}
	}

	return "", nil
}

// ListSessions gets current sessions information.
func (a *App) ListSessions() (*iterm2.ListSessionsResponse, error) {
	req := &iterm2.ClientOriginatedMessage{
		Id: seq.MessageId.Next(),
		Submessage: &iterm2.ClientOriginatedMessage_ListSessionsRequest{
			ListSessionsRequest: &iterm2.ListSessionsRequest{},
		},
	}

	resp, err := a.client.GetResponse(req)

	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	return resp.GetListSessionsResponse(), nil
}

// SendText sends text to a session, optionally broadcasting it if broadcast is enabled.
func (a *App) SendText(sessionId, text string, useBroadcastIfEnabled bool) error {
	suppressBroadcast := useBroadcastIfEnabled

	req := &iterm2.ClientOriginatedMessage{
		Id: seq.MessageId.Next(),
		Submessage: &iterm2.ClientOriginatedMessage_SendTextRequest{
			SendTextRequest: &iterm2.SendTextRequest{
				Session:           &sessionId,
				Text:              &text,
				SuppressBroadcast: &suppressBroadcast,
			},
		},
	}

	resp, err := a.client.GetResponse(req)
	if err != nil {
		return fmt.Errorf("send text: %w", err)
	}

	if resp.GetSendTextResponse().GetStatus() != iterm2.SendTextResponse_OK {
		return fmt.Errorf("send text: %s", resp.GetSendTextResponse().GetStatus().String())
	}

	return nil
}

// SplitPane splits the pane of the target session, returning the new session IDs on success.
func (a *App) SplitPane(sessionId string, vertical bool, before bool) ([]string, error) {
	// TODO profile and profile_customizations flags

	var direction iterm2.SplitPaneRequest_SplitDirection
	if vertical {
		direction = iterm2.SplitPaneRequest_VERTICAL
	} else {
		direction = iterm2.SplitPaneRequest_HORIZONTAL
	}

	req := &iterm2.ClientOriginatedMessage{
		Id: seq.MessageId.Next(),
		Submessage: &iterm2.ClientOriginatedMessage_SplitPaneRequest{
			SplitPaneRequest: &iterm2.SplitPaneRequest{
				Session:        &sessionId,
				SplitDirection: &direction,
				Before:         &before,
			},
		},
	}

	resp, err := a.client.GetResponse(req)
	if err != nil {
		return nil, fmt.Errorf("split pane: %w", err)
	}

	var returnErr error

	if resp.GetSplitPaneResponse().GetStatus() != iterm2.SplitPaneResponse_OK {
		returnErr = fmt.Errorf("split pane: %s", resp.GetSplitPaneResponse().GetStatus())
	}

	return resp.GetSplitPaneResponse().GetSessionId(), returnErr
}

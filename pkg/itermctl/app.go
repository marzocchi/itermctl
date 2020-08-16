package itermctl

import (
	"context"
	"encoding/json"
	"fmt"
	"mrz.io/itermctl/pkg/itermctl/internal/seq"
	"mrz.io/itermctl/pkg/itermctl/proto"
)

const DefaultProfileName = "Default"

type Alert struct {
	Title    string
	Subtitle string
	Buttons  []string
}

type TextInputAlert struct {
	Title        string
	Subtitle     string
	Placeholder  string
	DefaultValue string
}

// App provides methods to interact with the running iTerm2 Application.
type App struct {
	client *Client
}

// NewApp creates a new App bound to the given Client.
func NewApp(client *Client) *App {
	return &App{client: client}
}

// Activate iTerm2 (eg. gives it focus).
func (a *App) Activate(raiseAllWindow bool, ignoringOtherApps bool) error {
	return a.activate(&iterm2.ActivateRequest{
		ActivateApp: &iterm2.ActivateRequest_App{
			RaiseAllWindows:   &raiseAllWindow,
			IgnoringOtherApps: &ignoringOtherApps,
		},
	})
}

// ActivateWindow brings a window to the front.
func (a *App) ActivateWindow(id string) error {
	return a.activate(&iterm2.ActivateRequest{
		Identifier: &iterm2.ActivateRequest_WindowId{WindowId: id},
	})
}

// ActivateTab brings a tab to the front.
func (a *App) ActivateTab(id string) error {
	orderWindowFront := true
	selectTab := true
	return a.activate(&iterm2.ActivateRequest{
		Identifier:       &iterm2.ActivateRequest_TabId{TabId: id},
		OrderWindowFront: &orderWindowFront,
		SelectTab:        &selectTab,
	})
}

// ActivateSession brings a session to the front.
func (a *App) ActivateSession(id string) error {
	orderWindowFront := true
	selectSession := true
	selectTab := true
	return a.activate(&iterm2.ActivateRequest{
		Identifier:       &iterm2.ActivateRequest_SessionId{SessionId: id},
		OrderWindowFront: &orderWindowFront,
		SelectSession:    &selectSession,
		SelectTab:        &selectTab,
	})
}

func (a *App) activate(ae *iterm2.ActivateRequest) error {
	req := &iterm2.ClientOriginatedMessage{
		Id: seq.MessageId.Next(),
		Submessage: &iterm2.ClientOriginatedMessage_ActivateRequest{
			ActivateRequest: ae,
		},
	}

	resp, err := a.client.GetResponse(context.Background(), req)
	if resp == nil {
		return err
	}

	if resp.GetActivateResponse().GetStatus() != iterm2.ActivateResponse_OK {
		return fmt.Errorf("activate: %s", resp.GetActivateResponse().GetStatus().String())
	}

	return nil
}

// Active tells if iTerm2 is currently active.
func (a *App) Active() (bool, error) {
	resp, err := a.getFocus()
	if err != nil {
		return false, err
	}

	for _, n := range resp {
		if e, ok := n.GetEvent().(*iterm2.FocusChangedNotification_ApplicationActive); ok {
			return e.ApplicationActive, nil
		}
	}

	return false, nil
}

// ActiveWindowId returns the ID of the currently active window.
func (a *App) ActiveWindowId() (string, error) {
	resp, err := a.getFocus()
	if err != nil {
		return "", err
	}

	for _, n := range resp {
		if n.GetWindow() != nil {
			return n.GetWindow().GetWindowId(), nil
		}
	}

	return "", nil
}

// ActiveTabId returns the ID of the currently active window.
func (a *App) ActiveTabId() (string, error) {
	resp, err := a.getFocus()
	if err != nil {
		return "", err
	}

	for _, n := range resp {
		if n.GetSelectedTab() != "" {
			return n.GetSelectedTab(), nil
		}
	}

	return "", nil
}

// ActiveSessionId returns the ID of the currently active session through a getFocus call.
func (a *App) ActiveSessionId() (string, error) {
	resp, err := a.getFocus()
	if err != nil {
		return "", err
	}

	for _, n := range resp {
		if n.GetSession() != "" {
			return n.GetSession(), nil
		}
	}

	return "", nil
}

func (a *App) getFocus() ([]*iterm2.FocusChangedNotification, error) {
	req := &iterm2.ClientOriginatedMessage{
		Id: seq.MessageId.Next(),
		Submessage: &iterm2.ClientOriginatedMessage_FocusRequest{
			FocusRequest: &iterm2.FocusRequest{},
		},
	}

	resp, err := a.client.GetResponse(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("get focus: %w", err)
	}

	return resp.GetFocusResponse().GetNotifications(), nil
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

	resp, err := a.client.GetResponse(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("split pane: %w", err)
	}

	var returnErr error

	if resp.GetSplitPaneResponse().GetStatus() != iterm2.SplitPaneResponse_OK {
		returnErr = fmt.Errorf("split pane: %s", resp.GetSplitPaneResponse().GetStatus())
	}

	return resp.GetSplitPaneResponse().GetSessionId(), returnErr
}

// CloseTab closes the tabs specified by the given IDs. An error is returned when iTerm2 reports an error
// closing at least one tab.
func (a *App) CloseTab(force bool, tabIds ...string) error {
	return a.close(&iterm2.CloseRequest{
		Target: &iterm2.CloseRequest_Tabs{
			Tabs: &iterm2.CloseRequest_CloseTabs{TabIds: tabIds},
		},
		Force: &force,
	})
}

// CloseSession closes the sessions specified by the given IDs. An error is returned when iTerm2 reports an error
// closing at least one session.
func (a *App) CloseSession(force bool, sessionIds ...string) error {
	return a.close(&iterm2.CloseRequest{
		Target: &iterm2.CloseRequest_Sessions{
			Sessions: &iterm2.CloseRequest_CloseSessions{SessionIds: sessionIds},
		},
		Force: &force,
	})
}

// CloseWindow closes the windows specified by the given IDs. An error is returned when iTerm2 reports an error
// closing at least one window.
func (a *App) CloseWindow(force bool, windowIds ...string) error {
	return a.close(&iterm2.CloseRequest{
		Target: &iterm2.CloseRequest_Windows{
			Windows: &iterm2.CloseRequest_CloseWindows{WindowIds: windowIds},
		},
		Force: &force,
	})
}

func (a *App) close(cr *iterm2.CloseRequest) error {
	req := &iterm2.ClientOriginatedMessage{
		Id: seq.MessageId.Next(),
		Submessage: &iterm2.ClientOriginatedMessage_CloseRequest{
			CloseRequest: cr,
		},
	}

	resp, err := a.client.GetResponse(context.Background(), req)
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

	resp, err := a.client.GetResponse(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("create tab: %w", err)
	}

	var returnErr error
	if resp.GetCreateTabResponse().GetStatus() != iterm2.CreateTabResponse_OK {
		returnErr = fmt.Errorf("create tab: %s", resp.GetCreateTabResponse().GetStatus())
	}

	return resp.GetCreateTabResponse(), returnErr
}

// ScreenContents returns the current screen's contents for a session.
func (a *App) ScreenContents(sessionId string) (*iterm2.GetBufferResponse, error) {
	screenContents := true
	req := &iterm2.ClientOriginatedMessage{
		Id: seq.MessageId.Next(),
		Submessage: &iterm2.ClientOriginatedMessage_GetBufferRequest{
			GetBufferRequest: &iterm2.GetBufferRequest{
				Session: &sessionId,
				LineRange: &iterm2.LineRange{
					ScreenContentsOnly: &screenContents,
				},
			},
		},
	}

	resp, err := a.client.GetResponse(context.Background(), req)
	if err != nil {
		return nil, err
	}

	if resp.GetGetBufferResponse().GetStatus() != iterm2.GetBufferResponse_OK {
		return nil, fmt.Errorf("screen contents: %s", resp.GetGetBufferResponse().GetStatus())
	}

	return resp.GetGetBufferResponse(), nil
}

// ListSessions gets current sessions information.
func (a *App) ListSessions() (*iterm2.ListSessionsResponse, error) {
	req := &iterm2.ClientOriginatedMessage{
		Id: seq.MessageId.Next(),
		Submessage: &iterm2.ClientOriginatedMessage_ListSessionsRequest{
			ListSessionsRequest: &iterm2.ListSessionsRequest{},
		},
	}

	resp, err := a.client.GetResponse(context.Background(), req)
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

	resp, err := a.client.GetResponse(context.Background(), req)
	if err != nil {
		return fmt.Errorf("send text: %w", err)
	}

	if resp.GetSendTextResponse().GetStatus() != iterm2.SendTextResponse_OK {
		return fmt.Errorf("send text: %s", resp.GetSendTextResponse().GetStatus().String())
	}

	return nil
}

// GetText shows the TextInputAlert and blocks until the user types some text and hits OK. The TextInputAlert is
// application-modal unless a windowId is given. Returns the user's input text.
func (a *App) GetText(alert TextInputAlert, windowId string) (string, error) {
	invocation := fmt.Sprintf(
		"iterm2.get_string(title: %s, subtitle: %s, placeholder: %s, defaultValue: %s, window_id: %s)",
		asJsonString(alert.Title),
		asJsonString(alert.Subtitle),
		asJsonString(alert.Placeholder),
		asJsonString(alert.DefaultValue),
		asJsonString(windowId),
	)

	var reply string
	err := a.InvokeFunction(invocation, &reply)
	if err != nil {
		return "", err
	}

	return reply, nil
}

// ShowAlert shows the Alert and blocks until the user clicks one of the Alert's button. The Alert is application-modal
// unless a windowId is given. Returns the clicked button's text, or "OK" if the Alert has no custom button.
func (a *App) ShowAlert(alert Alert, windowId string) (string, error) {
	if alert.Buttons == nil {
		alert.Buttons = []string{}
	}

	invocation := fmt.Sprintf("iterm2.alert(title: %s, subtitle: %s, buttons: %s, window_id: %s)",
		asJsonString(alert.Title),
		asJsonString(alert.Subtitle),
		asJsonString(alert.Buttons),
		asJsonString(windowId),
	)

	var button int64
	err := a.InvokeFunction(invocation, &button)
	if err != nil {
		return "", err
	}

	if len(alert.Buttons) == 0 {
		return "OK", nil
	}

	return alert.Buttons[button-1000], nil
}

// InvokeFunction invokes an RPC function and unmarshalls the result into target. If iTerm2's response to the invocation
// is an error, target is left untouched and an error is returned.
func (a *App) InvokeFunction(invocation string, target interface{}) error {
	req := &iterm2.ClientOriginatedMessage{
		Id: seq.MessageId.Next(),
		Submessage: &iterm2.ClientOriginatedMessage_InvokeFunctionRequest{
			InvokeFunctionRequest: &iterm2.InvokeFunctionRequest{
				Context:    &iterm2.InvokeFunctionRequest_App_{},
				Invocation: &invocation,
			},
		},
	}

	resp, err := a.client.GetResponse(context.Background(), req)
	if resp == nil {
		return err
	}

	if invocationErr := resp.GetInvokeFunctionResponse().GetError(); invocationErr != nil {
		return fmt.Errorf("%s: %s", invocationErr.GetStatus(), invocationErr.GetErrorReason())
	}

	jsonResult := resp.GetInvokeFunctionResponse().GetSuccess().GetJsonResult()
	if err := json.Unmarshal([]byte(jsonResult), &target); err != nil {
		return fmt.Errorf("could not unmarshal invocation target: %w", err)
	}

	return nil
}

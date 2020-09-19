package itermctl

import (
	"context"
	"fmt"
	"mrz.io/itermctl/pkg/itermctl/internal/json"
	"mrz.io/itermctl/pkg/itermctl/iterm2"
	"sync"
)

const DefaultProfileName = "Default"

// Alert is an app-modal or window-modal alert window. Use it with App.ShowAlert.
// See https://www.iterm2.com/python-api/alert.html#iterm2.Alert.
type Alert struct {
	// Title is the window's title.
	Title string
	// Subtitle is an informative text that can span multiple lines.
	Subtitle string
	// Buttons optionally specifies a list of button labels; if no button is given, the alert will show a default "OK"
	// button.
	Buttons []string
}

// TextInputAlert is an app-modal or window-modal alert window with a text input field. Use it with App.GetText.
// See https://www.iterm2.com/python-api/registration.html#iterm2.registration.StatusBarRPC.
type TextInputAlert struct {
	// Title is the window's title.
	Title string
	// Subtitle is an informative text that can span multiple lines.
	Subtitle string

	// Placeholder is a text that appears when the alert's text field is empty
	Placeholder string

	// DefaultValue is the text field's initial content.
	DefaultValue string
}

// App provides methods to interact with the running iTerm2 Application.
type App struct {
	conn     *Connection
	mx       *sync.Mutex
	sessions map[string]*Session

	active        bool
	activeSession *Session
}

// NewApp creates a new App bound to the given Connection.
func NewApp(conn *Connection) (*App, error) {
	a := &App{conn: conn, mx: &sync.Mutex{}}

	ctx, cancel := context.WithCancel(context.Background())

	newSessions, err := MonitorNewSessions(ctx, conn)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("app: %w", err)
	}

	terminatedSessions, err := MonitorSessionsTermination(ctx, conn)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("app: %w", err)
	}

	focusUpdates, err := MonitorFocus(ctx, conn)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("app: %w", err)
	}

	a.sessions = make(map[string]*Session)
	a.applyStateChanges(newSessions, terminatedSessions, focusUpdates)

	focusChangedNotifications, err := a.GetFocus()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("app: %w", err)
	}

	for _, n := range focusChangedNotifications {
		a.applyFocusUpdate(GetFocusUpdate(n))
	}

	return a, nil
}

func (a *App) setActive(active bool) {
	a.mx.Lock()
	defer a.mx.Unlock()
	a.active = active
}

func (a *App) setActiveSessionById(sessionId string) {
	a.mx.Lock()
	defer a.mx.Unlock()

	if _, ok := a.sessions[sessionId]; ok {
		a.sessions[sessionId].setActive(true)
	} else {
		a.sessions[sessionId] = newSession(sessionId, a, a.conn, true)
	}

	a.activeSession = a.sessions[sessionId]
}

func (a *App) addSession(sessionId string) {
	a.mx.Lock()
	defer a.mx.Unlock()

	if _, ok := a.sessions[sessionId]; !ok {
		a.sessions[sessionId] = newSession(sessionId, a, a.conn, false)
	}
}

func (a *App) deleteSession(sessionId string) {
	a.mx.Lock()
	defer a.mx.Unlock()
	delete(a.sessions, sessionId)

	if a.activeSession != nil && a.activeSession.Id() == sessionId {
		a.activeSession = nil
	}
}

func (a *App) applyStateChanges(newSessions <-chan *iterm2.NewSessionNotification,
	terminatedSessions <-chan *iterm2.TerminateSessionNotification,
	focusUpdates <-chan FocusUpdate) {

	go func() {
		for newSessions != nil || terminatedSessions != nil || focusUpdates != nil {
			select {
			case focusUpdate, ok := <-focusUpdates:
				if !ok {
					focusUpdates = nil
					continue
				}

				a.applyFocusUpdate(focusUpdate)

			case newSession, ok := <-newSessions:
				if !ok {
					newSessions = nil
					continue
				}

				a.addSession(newSession.GetSessionId())
			case terminatedSession, ok := <-terminatedSessions:
				if !ok {
					terminatedSessions = nil
					continue
				}

				a.deleteSession(terminatedSession.GetSessionId())
			}
		}
	}()
}

func (a *App) applyFocusUpdate(focusUpdate FocusUpdate) {
	switch focusUpdate.Which {
	case SessionSelected:
		a.setActiveSessionById(focusUpdate.Id)
	case ApplicationBecameActive:
		a.setActive(true)
	case ApplicationResignedActive:
		a.setActive(false)
	}
}

// Activate activates iTerm2 (eg. gives it focus).
func (a *App) Activate(raiseAllWindow bool, ignoringOtherApps bool) error {
	return a.sendActivateRequest(&iterm2.ActivateRequest{
		ActivateApp: &iterm2.ActivateRequest_App{
			RaiseAllWindows:   &raiseAllWindow,
			IgnoringOtherApps: &ignoringOtherApps,
		},
	})
}

// Active tells if iTerm2 is currently active.
func (a *App) Active() bool {
	a.mx.Lock()
	defer a.mx.Unlock()
	return a.active
}

// ActivateTerminalWindow brings a window to the front.
func (a *App) ActivateTerminalWindow(id string) error {
	return a.sendActivateRequest(&iterm2.ActivateRequest{
		Identifier: &iterm2.ActivateRequest_WindowId{WindowId: id},
	})
}

// ActiveTerminalWindowId returns the ID of the currently active window.
func (a *App) ActiveTerminalWindowId() (string, error) {
	resp, err := a.GetFocus()
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

// CloseTerminalWindow closes the windows specified by the given IDs. An error is returned when iTerm2 reports an error
// closing at least one window.
func (a *App) CloseTerminalWindow(force bool, windowIds ...string) error {
	return a.sendCloseRequest(&iterm2.CloseRequest{
		Target: &iterm2.CloseRequest_Windows{
			Windows: &iterm2.CloseRequest_CloseWindows{WindowIds: windowIds},
		},
		Force: &force,
	})
}

// SelectTab brings a tab to the front.
func (a *App) SelectTab(id string) error {
	orderWindowFront := true
	selectTab := true
	return a.sendActivateRequest(&iterm2.ActivateRequest{
		Identifier:       &iterm2.ActivateRequest_TabId{TabId: id},
		OrderWindowFront: &orderWindowFront,
		SelectTab:        &selectTab,
	})
}

// SelectedTabId returns the ID of the currently active tab.
func (a *App) SelectedTabId() (string, error) {
	resp, err := a.GetFocus()
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
		Submessage: &iterm2.ClientOriginatedMessage_CreateTabRequest{
			CreateTabRequest: createReq,
		},
	}

	resp, err := a.conn.GetResponse(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("create tab: %w", err)
	}

	var returnErr error
	if resp.GetCreateTabResponse().GetStatus() != iterm2.CreateTabResponse_OK {
		returnErr = fmt.Errorf("create tab: %s", resp.GetCreateTabResponse().GetStatus())
	}

	return resp.GetCreateTabResponse(), returnErr
}

// CloseTab closes the tabs specified by the given IDs. An error is returned when iTerm2 reports an error
// closing at least one tab.
func (a *App) CloseTab(force bool, tabIds ...string) error {
	return a.sendCloseRequest(&iterm2.CloseRequest{
		Target: &iterm2.CloseRequest_Tabs{
			Tabs: &iterm2.CloseRequest_CloseTabs{TabIds: tabIds},
		},
		Force: &force,
	})
}

func (a *App) Session(id string) *Session {
	a.mx.Lock()
	defer a.mx.Unlock()

	if s, ok := a.sessions[id]; ok {
		return s
	}
	return nil
}

func (a *App) ActiveSession() *Session {
	a.mx.Lock()
	defer a.mx.Unlock()

	return a.activeSession
}

// ListSessions gets current sessions information.
func (a *App) ListSessions() (*iterm2.ListSessionsResponse, error) {
	req := &iterm2.ClientOriginatedMessage{
		Submessage: &iterm2.ClientOriginatedMessage_ListSessionsRequest{
			ListSessionsRequest: &iterm2.ListSessionsRequest{},
		},
	}

	resp, err := a.conn.GetResponse(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	return resp.GetListSessionsResponse(), nil
}

func (a *App) sendActivateRequest(activateReq *iterm2.ActivateRequest) error {
	req := &iterm2.ClientOriginatedMessage{
		Submessage: &iterm2.ClientOriginatedMessage_ActivateRequest{
			ActivateRequest: activateReq,
		},
	}

	resp, err := a.conn.GetResponse(context.Background(), req)
	if resp == nil {
		return err
	}

	if resp.GetActivateResponse().GetStatus() != iterm2.ActivateResponse_OK {
		return fmt.Errorf("sendActivateRequest: %s", resp.GetActivateResponse().GetStatus().String())
	}

	return nil
}

func (a *App) GetFocus() ([]*iterm2.FocusChangedNotification, error) {
	req := &iterm2.ClientOriginatedMessage{
		Submessage: &iterm2.ClientOriginatedMessage_FocusRequest{
			FocusRequest: &iterm2.FocusRequest{},
		},
	}

	resp, err := a.conn.GetResponse(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("get focus: %w", err)
	}

	return resp.GetFocusResponse().GetNotifications(), nil
}

func (a *App) sendCloseRequest(cr *iterm2.CloseRequest) error {
	req := &iterm2.ClientOriginatedMessage{
		Submessage: &iterm2.ClientOriginatedMessage_CloseRequest{
			CloseRequest: cr,
		},
	}

	resp, err := a.conn.GetResponse(context.Background(), req)
	if err != nil {
		return fmt.Errorf("sendCloseRequest: %w", err)
	}

	for _, s := range resp.GetCloseResponse().GetStatuses() {
		if s != iterm2.CloseResponse_OK {
			return fmt.Errorf("sendCloseRequest: %s", s.String())
		}
	}

	return nil
}

// GetText shows the TextInputAlert and blocks until the user types some text and hits OK. The TextInputAlert is
// application-modal unless a windowId is given. Returns the user's input text.
func (a *App) GetText(alert TextInputAlert, windowId string) (string, error) {
	invocation := fmt.Sprintf(
		"iterm2.get_string(title: %s, subtitle: %s, placeholder: %s, defaultValue: %s, window_id: %s)",
		json.MustMarshal(alert.Title),
		json.MustMarshal(alert.Subtitle),
		json.MustMarshal(alert.Placeholder),
		json.MustMarshal(alert.DefaultValue),
		json.MustMarshal(windowId),
	)

	var reply string
	err := a.conn.InvokeFunction(invocation, &reply)
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
		json.MustMarshal(alert.Title),
		json.MustMarshal(alert.Subtitle),
		json.MustMarshal(alert.Buttons),
		json.MustMarshal(windowId),
	)

	var button int64
	err := a.conn.InvokeFunction(invocation, &button)
	if err != nil {
		return "", err
	}

	if len(alert.Buttons) == 0 {
		return "OK", nil
	}

	return alert.Buttons[button-1000], nil
}

package itermctl

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"mrz.io/itermctl/pkg/itermctl/internal/json"
	"mrz.io/itermctl/pkg/itermctl/proto"
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

type NumberOfLines struct {
	FirstVisible int32 `json:"first_visible"`
	Overflow     int32 `json:"overflow"`
	Grid         int32 `json:"grid"`
	History      int32 `json:"history"`
}

type GridSize struct {
	Width  int32 `json:"width"`
	Height int32 `json:"height"`
}

type Session struct {
	id   string
	app  *App
	conn *Connection
}

func (s *Session) Id() string {
	return s.id
}

func (s *Session) Active() (bool, error) {
	if activeSessionId, err := s.app.ActiveSessionId(); err != nil {
		return false, err
	} else {
		return activeSessionId == s.id, nil
	}
}

// Activate brings a session to the front.
func (s *Session) Activate() error {
	orderWindowFront := true
	selectSession := true
	selectTab := true
	return s.app.activate(&iterm2.ActivateRequest{
		Identifier:       &iterm2.ActivateRequest_SessionId{SessionId: s.id},
		OrderWindowFront: &orderWindowFront,
		SelectSession:    &selectSession,
		SelectTab:        &selectTab,
	})
}

// SplitPane splits the pane of the this session, returning the new session IDs on success.
func (s *Session) SplitPane(vertical bool, before bool) ([]string, error) {
	// TODO profile and profile_customizations flags

	var direction iterm2.SplitPaneRequest_SplitDirection
	if vertical {
		direction = iterm2.SplitPaneRequest_VERTICAL
	} else {
		direction = iterm2.SplitPaneRequest_HORIZONTAL
	}

	req := &iterm2.ClientOriginatedMessage{
		Submessage: &iterm2.ClientOriginatedMessage_SplitPaneRequest{
			SplitPaneRequest: &iterm2.SplitPaneRequest{
				Session:        &s.id,
				SplitDirection: &direction,
				Before:         &before,
			},
		},
	}

	resp, err := s.conn.GetResponse(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("split pane: %w", err)
	}

	var returnErr error

	if resp.GetSplitPaneResponse().GetStatus() != iterm2.SplitPaneResponse_OK {
		returnErr = fmt.Errorf("split pane: %s", resp.GetSplitPaneResponse().GetStatus())
	}

	return resp.GetSplitPaneResponse().GetSessionId(), returnErr
}

// Close closes this session.
func (s *Session) Close(force bool) error {
	sessionIds := []string{s.id}
	return s.app.close(&iterm2.CloseRequest{
		Target: &iterm2.CloseRequest_Sessions{
			Sessions: &iterm2.CloseRequest_CloseSessions{SessionIds: sessionIds},
		},
		Force: &force,
	})
}

// SendText sends text to the session, optionally broadcasting it if broadcast is enabled.
func (s *Session) SendText(text string, useBroadcastIfEnabled bool) error {
	suppressBroadcast := useBroadcastIfEnabled

	req := &iterm2.ClientOriginatedMessage{
		Submessage: &iterm2.ClientOriginatedMessage_SendTextRequest{
			SendTextRequest: &iterm2.SendTextRequest{
				Session:           &s.id,
				Text:              &text,
				SuppressBroadcast: &suppressBroadcast,
			},
		},
	}

	resp, err := s.conn.GetResponse(context.Background(), req)
	if err != nil {
		return fmt.Errorf("send text: %w", err)
	}

	if resp.GetSendTextResponse().GetStatus() != iterm2.SendTextResponse_OK {
		return fmt.Errorf("send text: %s", resp.GetSendTextResponse().GetStatus().String())
	}

	return nil
}

func (s *Session) TrailingLines(n int32) (*iterm2.GetBufferResponse, error) {
	req := &iterm2.ClientOriginatedMessage{
		Submessage: &iterm2.ClientOriginatedMessage_GetBufferRequest{
			GetBufferRequest: &iterm2.GetBufferRequest{
				Session: &s.id,
				LineRange: &iterm2.LineRange{
					TrailingLines: &n,
				},
			},
		},
	}

	resp, err := s.conn.GetResponse(context.Background(), req)
	if err != nil {
		return nil, err
	}

	if resp.GetGetBufferResponse().GetStatus() != iterm2.GetBufferResponse_OK {
		return nil, fmt.Errorf("screen contents: %s", resp.GetGetBufferResponse().GetStatus())
	}

	return resp.GetGetBufferResponse(), nil
}

// ScreenContents returns the current screen's contents.
func (s *Session) ScreenContents() (*iterm2.GetBufferResponse, error) {
	screenContentsOnly := true
	req := &iterm2.ClientOriginatedMessage{
		Submessage: &iterm2.ClientOriginatedMessage_GetBufferRequest{
			GetBufferRequest: &iterm2.GetBufferRequest{
				Session: &s.id,
				LineRange: &iterm2.LineRange{
					ScreenContentsOnly: &screenContentsOnly,
				},
			},
		},
	}

	resp, err := s.conn.GetResponse(context.Background(), req)
	if err != nil {
		return nil, err
	}

	if resp.GetGetBufferResponse().GetStatus() != iterm2.GetBufferResponse_OK {
		return nil, fmt.Errorf("screen contents: %s", resp.GetGetBufferResponse().GetStatus())
	}

	return resp.GetGetBufferResponse(), nil
}

func (s *Session) NumberOfLines() (NumberOfLines, error) {
	result := NumberOfLines{}
	if err := s.getSessionProperty("number_of_lines", &result); err != nil {
		return NumberOfLines{}, err
	}
	return result, nil
}

func (s *Session) Buried() (bool, error) {
	var result bool
	if err := s.getSessionProperty("buried", &result); err != nil {
		return false, err
	}

	return result, nil
}

func (s *Session) getSessionProperty(propName string, target interface{}) error {
	req := &iterm2.ClientOriginatedMessage{
		Submessage: &iterm2.ClientOriginatedMessage_GetPropertyRequest{
			GetPropertyRequest: &iterm2.GetPropertyRequest{
				Identifier: &iterm2.GetPropertyRequest_SessionId{
					SessionId: s.id,
				},
				Name: &propName,
			},
		},
	}

	resp, err := s.conn.GetResponse(context.Background(), req)
	if err != nil {
		return fmt.Errorf("get property: %w", err)
	}

	if resp.GetGetPropertyResponse().GetStatus() != iterm2.GetPropertyResponse_OK {
		return fmt.Errorf("get property: %s", resp.GetGetPropertyResponse().GetStatus())
	}

	if err := json.UnmarshalString(resp.GetGetPropertyResponse().GetJsonValue(), target); err != nil {
		return fmt.Errorf("get property: %w", err)
	}

	return nil
}

// App provides methods to interact with the running iTerm2 Application.
type App struct {
	conn     *Connection
	mx       *sync.Mutex
	sessions map[string]*Session
}

// NewApp creates a new App bound to the given Connection.
func NewApp(conn *Connection) (*App, error) {
	a := &App{conn: conn, mx: &sync.Mutex{}}

	newSessions, err := conn.MonitorNewSessions(context.Background())
	if err != nil {
		return nil, fmt.Errorf("app: %w", err)
	}

	terminatedSessions, err := conn.MonitorSessionsTermination(context.Background())
	if err != nil {
		return nil, fmt.Errorf("app: %w", err)
	}

	a.sessions = make(map[string]*Session)
	a.maintainSessions(newSessions, terminatedSessions)

	if sessions, err := a.ListSessions(); err != nil {
		return nil, fmt.Errorf("app: %w", err)
	} else {
		for _, window := range sessions.GetWindows() {
			for _, tabs := range window.GetTabs() {
				for _, link := range tabs.GetRoot().GetLinks() {
					a.addSession(link.GetSession().GetUniqueIdentifier())
				}
			}
		}
	}

	return a, nil
}

func (a *App) addSession(sessionId string) {
	a.mx.Lock()
	defer a.mx.Unlock()
	a.sessions[sessionId] = &Session{id: sessionId, app: a, conn: a.conn}
}

func (a *App) deleteSession(sessionId string) {
	a.mx.Lock()
	defer a.mx.Unlock()
	delete(a.sessions, sessionId)
}

func (a *App) maintainSessions(newSessions <-chan *iterm2.NewSessionNotification, terminatedSessions <-chan *iterm2.TerminateSessionNotification) {
	go func() {
		for newSessions != nil && terminatedSessions != nil {
			select {
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

func (a *App) Session(id string) (*Session, error) {
	a.mx.Lock()
	defer a.mx.Unlock()

	if s, ok := a.sessions[id]; ok {
		return s, nil
	}

	return nil, fmt.Errorf("app: session unknown: %s", id)
}

func (a *App) BeginTransaction(ctx context.Context) error {
	start := true
	startMessage := &iterm2.ClientOriginatedMessage{
		Submessage: &iterm2.ClientOriginatedMessage_TransactionRequest{
			TransactionRequest: &iterm2.TransactionRequest{Begin: &start},
		},
	}

	_, err := a.conn.GetResponse(ctx, startMessage)

	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	go func() {
		<-ctx.Done()
		end := false
		endMessage := &iterm2.ClientOriginatedMessage{
			Submessage: &iterm2.ClientOriginatedMessage_TransactionRequest{
				TransactionRequest: &iterm2.TransactionRequest{Begin: &end},
			},
		}

		_, err := a.conn.GetResponse(context.Background(), endMessage)
		if err != nil {
			logrus.Errorf("end transaction: %s", err)
		}
	}()

	return nil
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

func (a *App) activate(ae *iterm2.ActivateRequest) error {
	req := &iterm2.ClientOriginatedMessage{
		Submessage: &iterm2.ClientOriginatedMessage_ActivateRequest{
			ActivateRequest: ae,
		},
	}

	resp, err := a.conn.GetResponse(context.Background(), req)
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

func (a *App) ActiveSession() (*Session, error) {
	if id, err := a.ActiveSessionId(); err == nil {
		return a.Session(id)
	} else {
		return nil, err
	}
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

	return "", fmt.Errorf("app: no session")
}

func (a *App) getFocus() ([]*iterm2.FocusChangedNotification, error) {
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
		Submessage: &iterm2.ClientOriginatedMessage_CloseRequest{
			CloseRequest: cr,
		},
	}

	resp, err := a.conn.GetResponse(context.Background(), req)
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

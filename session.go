package itermctl

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"mrz.io/itermctl/internal/json"
	"mrz.io/itermctl/iterm2"
	"strings"
	"sync"
)

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
	id     string
	app    *App
	conn   *Connection
	active bool
	mx     *sync.Mutex
}

func newSession(id string, app *App, conn *Connection, active bool) *Session {
	return &Session{id: id, app: app, conn: conn, active: active, mx: &sync.Mutex{}}
}

func (s *Session) setActive(active bool) {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.active = active
}

func (s *Session) Id() string {
	return s.id
}

func (s *Session) Active() bool {
	s.mx.Lock()
	defer s.mx.Unlock()
	return s.active
}

// Activate brings a session to the front.
func (s *Session) Activate() error {
	orderWindowFront := true
	selectSession := true
	selectTab := true
	return s.app.sendActivateRequest(&iterm2.ActivateRequest{
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
	return s.app.sendCloseRequest(&iterm2.CloseRequest{
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
		return fmt.Errorf("Send text: %w", err)
	}

	if resp.GetSendTextResponse().GetStatus() != iterm2.SendTextResponse_OK {
		return fmt.Errorf("Send text: %s", resp.GetSendTextResponse().GetStatus().String())
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
func (s *Session) ScreenContents(coordRange *iterm2.WindowedCoordRange) (*iterm2.GetBufferResponse, error) {
	screenContentsOnly := coordRange == nil
	req := &iterm2.ClientOriginatedMessage{
		Submessage: &iterm2.ClientOriginatedMessage_GetBufferRequest{
			GetBufferRequest: &iterm2.GetBufferRequest{
				Session: &s.id,
				LineRange: &iterm2.LineRange{
					ScreenContentsOnly: &screenContentsOnly,
					WindowedCoordRange: coordRange,
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

// SelectedText returns the first subselection as a string.
// TODO merge all subselections as in `iterm2.selection.Selection.async_get_string`
func (s *Session) SelectedText() (string, error) {
	req := &iterm2.ClientOriginatedMessage{
		Submessage: &iterm2.ClientOriginatedMessage_SelectionRequest{
			SelectionRequest: &iterm2.SelectionRequest{
				Request: &iterm2.SelectionRequest_GetSelectionRequest_{
					GetSelectionRequest: &iterm2.SelectionRequest_GetSelectionRequest{SessionId: &s.id},
				},
			},
		},
	}

	resp, err := s.conn.GetResponse(context.Background(), req)

	if err != nil {
		return "", fmt.Errorf("selected text: %w", err)
	}

	tx, err := s.conn.Transaction()
	if err != nil {
		return "", fmt.Errorf("selected text: %w", err)
	}

	defer func() {
		if err := tx.End(); err != nil {
			logrus.Errorf("selected text: %s", err)
		}
	}()

	for _, subsel := range resp.GetSelectionResponse().GetGetSelectionResponse().GetSelection().GetSubSelections() {
		sc, err := s.ScreenContents(subsel.GetWindowedCoordRange())
		if err != nil {
			return "", fmt.Errorf("selected text: %w", err)
		}

		return ToString(sc.GetContents()), nil
	}

	return "", nil
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

func ToString(lines []*iterm2.LineContents) string {
	str := &strings.Builder{}
	for _, line := range lines {
		str.WriteString(line.GetText())

		if line.GetContinuation() == iterm2.LineContents_CONTINUATION_HARD_EOL {
			str.WriteString("\n")
		}
	}

	return str.String()
}

package itermctl

//
//func TestMain(m *testing.M) {
//	WaitResponseTimeout = 100 * time.Millisecond
//	os.Exit(m.Run())
//}
//
//func TestClient_Send(t *testing.T) {
//	t.Parallel()
//
//	responses := make(chan *iterm2.ServerOriginatedMessage)
//
//	conn := test.NewConn(responses)
//	defer func() {
//		close(responses)
//	}()
//
//	client := NewClient(conn)
//
//	msg := &iterm2.ClientOriginatedMessage{
//		Id: seq.MessageId.Next(),
//	}
//
//	client.Send(msg)
//
//	// wait for the message to actually have reached conn
//	<-time.After(1 * time.Millisecond)
//	if conn.Requests()[0] != msg {
//		t.Fatalf("expected %s, got %s", conn.Requests()[0], msg)
//	}
//}
//
//func TestClient_SendAfterClose(t *testing.T) {
//	t.Parallel()
//
//	responses := make(chan *iterm2.ServerOriginatedMessage)
//
//	conn := test.NewConn(responses)
//	client := NewClient(conn)
//
//	msg := &iterm2.ClientOriginatedMessage{
//		Id: seq.MessageId.Next(),
//	}
//
//	close(responses)
//	<-time.After(1 * time.Millisecond)
//
//	err := client.Send(msg)
//	if err != ErrClosed {
//		t.Fatalf("expected %s, got %#v", ErrClosed, err)
//	}
//}
//
//func TestClient_Receiver(t *testing.T) {
//	t.Parallel()
//
//	responses := make(chan *iterm2.ServerOriginatedMessage)
//
//	conn := test.NewConn(responses)
//	client := NewClient(conn)
//
//	recv, err := client.Receiver(context.Background(), "test receiver", nil)
//
//	if err != nil {
//		t.Fatalf("expected no error, got %s", err)
//	}
//
//	responses <- &iterm2.ServerOriginatedMessage{
//		Submessage: &iterm2.ServerOriginatedMessage_Notification{Notification: &iterm2.Notification{
//			NewSessionNotification: &iterm2.NewSessionNotification{},
//		}},
//	}
//
//	responses <- &iterm2.ServerOriginatedMessage{
//		Submessage: &iterm2.ServerOriginatedMessage_Notification{Notification: &iterm2.Notification{
//			TerminateSessionNotification: &iterm2.TerminateSessionNotification{},
//		}},
//	}
//	close(responses)
//
//	var receivedResponses []*iterm2.ServerOriginatedMessage
//
//	for response := range recv {
//		receivedResponses = append(receivedResponses, response)
//	}
//
//	if len(receivedResponses) != 2 {
//		t.Fatalf("expected 2 responses, got %d", len(receivedResponses))
//	}
//}
//
//func TestClient_Request(t *testing.T) {
//	t.Parallel()
//
//	responses := make(chan *iterm2.ServerOriginatedMessage)
//
//	conn := test.NewConn(responses)
//	client := NewClient(conn)
//
//	id := seq.MessageId.Next()
//
//	req := &iterm2.ClientOriginatedMessage{
//		Id:         id,
//		Submessage: &iterm2.ClientOriginatedMessage_ActivateRequest{},
//	}
//
//	respCh, err := client.Request(context.Background(), req)
//	if err != nil {
//		t.Fatalf("expected no error, got %s", err)
//	}
//
//	responses <- &iterm2.ServerOriginatedMessage{
//		Id:         id,
//		Submessage: &iterm2.ServerOriginatedMessage_ActivateResponse{},
//	}
//	close(responses)
//
//	resp := <-respCh
//
//	if resp.GetId() != req.GetId() {
//		t.Fatalf("expected message id: %d, got %d", req.GetId(), resp.GetId())
//	}
//}
//
//func TestClient_GetResponse(t *testing.T) {
//	t.Parallel()
//
//	responses := make(chan *iterm2.ServerOriginatedMessage)
//
//	conn := test.NewConn(responses)
//	client := NewClient(conn)
//
//	id := seq.MessageId.Next()
//
//	req := &iterm2.ClientOriginatedMessage{
//		Id:         id,
//		Submessage: &iterm2.ClientOriginatedMessage_ActivateRequest{},
//	}
//
//	go func() {
//		// wait to allow GetResponse to setup the receiver
//		<-time.After(50 * time.Millisecond)
//		responses <- &iterm2.ServerOriginatedMessage{
//			Id:         id,
//			Submessage: &iterm2.ServerOriginatedMessage_ActivateResponse{},
//		}
//		close(responses)
//	}()
//
//	resp, err := client.GetResponse(context.Background(), req)
//	if err != nil {
//		t.Fatalf("expected no error, got %s", err)
//	}
//
//	if resp.GetId() != req.GetId() {
//		t.Fatalf("expected message id: %d, got %d", req.GetId(), resp.GetId())
//	}
//}
//
//func TestClient_GetResponse_WaitTimeout(t *testing.T) {
//	t.Parallel()
//
//	responses := make(chan *iterm2.ServerOriginatedMessage)
//
//	conn := test.NewConn(responses)
//	client := NewClient(conn)
//	defer func() {
//		close(responses)
//	}()
//
//	id := seq.MessageId.Next()
//
//	req := &iterm2.ClientOriginatedMessage{
//		Id:         id,
//		Submessage: &iterm2.ClientOriginatedMessage_ActivateRequest{},
//	}
//
//	_, err := client.GetResponse(context.Background(), req)
//	if err != context.DeadlineExceeded {
//		t.Fatalf("expected DeadlineExceeded, got %#v", err)
//	}
//}

package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"

	"trip2g/internal/appconfig"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

// fakeSMTPServer is a minimal in-process SMTP server that advertises
// STARTTLS in its EHLO response (mirroring a real Postfix relay), but never
// actually performs the TLS handshake. It records whether the connecting
// client issued a STARTTLS command, so tests can assert the client never
// attempted opportunistic TLS when it shouldn't.
type fakeSMTPServer struct {
	ln net.Listener
	mu sync.Mutex

	starttlsIssued bool
}

func newFakeSMTPServer(t *testing.T) *fakeSMTPServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	s := &fakeSMTPServer{ln: ln}
	go s.serve()
	t.Cleanup(func() { ln.Close() })
	return s
}

func (s *fakeSMTPServer) addr() string {
	return s.ln.Addr().String()
}

func (s *fakeSMTPServer) sawSTARTTLS() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.starttlsIssued
}

func (s *fakeSMTPServer) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *fakeSMTPServer) handle(conn net.Conn) {
	defer conn.Close()

	r := bufio.NewReader(conn)
	w := conn

	fmt.Fprintf(w, "220 fake.smtp.local ESMTP\r\n")

	inData := false
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")

		if inData {
			if line == "." {
				inData = false
				fmt.Fprintf(w, "250 OK\r\n")
			}
			continue
		}

		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "EHLO"):
			fmt.Fprintf(w, "250-fake.smtp.local\r\n")
			fmt.Fprintf(w, "250 STARTTLS\r\n")
		case strings.HasPrefix(upper, "HELO"):
			fmt.Fprintf(w, "250 fake.smtp.local\r\n")
		case strings.HasPrefix(upper, "STARTTLS"):
			s.mu.Lock()
			s.starttlsIssued = true
			s.mu.Unlock()
			fmt.Fprintf(w, "220 ready to start TLS\r\n")
		case strings.HasPrefix(upper, "MAIL FROM"):
			fmt.Fprintf(w, "250 OK\r\n")
		case strings.HasPrefix(upper, "RCPT TO"):
			fmt.Fprintf(w, "250 OK\r\n")
		case upper == "DATA":
			inData = true
			fmt.Fprintf(w, "354 End data with <CR><LF>.<CR><LF>\r\n")
		case upper == "QUIT":
			fmt.Fprintf(w, "221 Bye\r\n")
			return
		default:
			fmt.Fprintf(w, "250 OK\r\n")
		}
	}
}

// TestSendMail_StartTLSFalse_NeverIssuesSTARTTLS reproduces the field bug:
// a server advertising STARTTLS in EHLO (like a real Postfix relay with a
// self-signed cert) must not trigger opportunistic TLS when
// SMTPStartTLS=false, or the send fails with a cert-verification error.
func TestSendMail_StartTLSFalse_NeverIssuesSTARTTLS(t *testing.T) {
	server := newFakeSMTPServer(t)
	host, port := splitHostPort(t, server.addr())

	a := &app{appState: &appState{
		log: &logger.TestLogger{},
		config: &appconfig.Config{
			SMTPHost:     host,
			SMTPPort:     port,
			SMTPStartTLS: false,
			MailFrom:     "no-reply@example.com",
		},
	}}

	err := a.SendMail(context.Background(), model.Mail{
		To:      "user@example.com",
		Subject: "Test",
		Plain:   []byte("hello"),
	})
	require.NoError(t, err)
	require.False(t, server.sawSTARTTLS(), "plaintext path must never issue STARTTLS")
}

func splitHostPort(t *testing.T, addr string) (string, int) {
	t.Helper()
	host, portStr, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	var port int
	_, err = fmt.Sscanf(portStr, "%d", &port)
	require.NoError(t, err)
	return host, port
}

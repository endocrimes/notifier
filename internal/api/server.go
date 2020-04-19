package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/endocrimes/notifier/internal/bot"
	"github.com/endocrimes/notifier/internal/tokensigner"
	"github.com/gorilla/mux"
	"github.com/hashicorp/go-hclog"
)

type server struct {
	logger        hclog.Logger
	bot           *bot.Bot
	tokenUnsigner tokensigner.TokenSigner
}

func NewServer(logger hclog.Logger, bot *bot.Bot, ts tokensigner.TokenSigner) Server {
	return &server{
		logger:        logger,
		bot:           bot,
		tokenUnsigner: ts,
	}
}

type Server interface {
	Start(ctx context.Context, iface string) error
}

func (s *server) Start(ctx context.Context, iface string) error {
	r := mux.NewRouter()
	s.registerCmds(r)
	hs := &http.Server{
		Addr:    iface,
		Handler: r,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancelFn := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancelFn()
		err := hs.Shutdown(shutdownCtx)
		if err != nil {
			s.logger.Error("error during graceful shutdown", "error", err)
		}
	}()

	return hs.ListenAndServe()
}

func (s *server) registerCmds(r *mux.Router) {
	r.HandleFunc("/notify", s.wrap(s.notify))
}

func (s *server) handleErr(resp http.ResponseWriter, req *http.Request, err error) {
	code := 500
	errMsg := err.Error()
	if http, ok := err.(HTTPCodedError); ok {
		code = http.Code()
	}

	resp.WriteHeader(code)
	resp.Write([]byte(errMsg))
	s.logger.Error("request failed", "method", req.Method, "path", reqURL, "error", err, "code", code)
}

// wrap is used to wrap functions to make them more convenient
func (s *server) wrap(handler func(resp http.ResponseWriter, req *http.Request) (interface{}, error)) func(resp http.ResponseWriter, req *http.Request) {
	f := func(resp http.ResponseWriter, req *http.Request) {
		// Invoke the handler
		reqURL := req.URL.String()
		start := time.Now()
		defer func() {
			s.logger.Debug("request complete", "method", req.Method, "path", reqURL, "duration", time.Now().Sub(start))
		}()
		obj, err := handler(resp, req)

		// Check for an error
		if err != nil {
			s.handleErr(resp, req, err)
			return
		}

		prettyPrint := false
		if v, ok := req.URL.Query()["pretty"]; ok {
			if len(v) > 0 && (len(v[0]) == 0 || v[0] != "0") {
				prettyPrint = true
			}
		}

		// Write out the JSON object
		if obj != nil {
			var buf bytes.Buffer
			enc := json.NewEncoder(&buf)
			if prettyPrint {
				enc.SetIndent("", "    ")
			}
			if err == nil {
				buf.Write([]byte("\n"))
			} else {
				s.handleErr(resp, req, err)
				return
			}
			resp.Header().Set("Content-Type", "application/json")
			resp.Write(buf.Bytes())
		}
	}
	return f
}

func (s *server) parseToken(r *http.Request) (string, error) {
	headerToken := r.Header.Get("Token")
	if headerToken != "" {
		return headerToken, nil
	}

	return "", CodedError(401, "Missing token in request")
}

func (s *server) notify(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	if r.Method != "POST" {
		return nil, MethodNotAllowedErr
	}

	token, err := s.parseToken(r)
	if err != nil {
		return nil, err
	}

	chatID, err := s.tokenUnsigner.VerifyToken([]byte(token))
	if err != nil {
		return nil, CodedError(401, fmt.Sprintf("Token validation failed: %v", err))
	}

	var req SendNotificationRequest
	dec := json.NewDecoder(r.Body)
	err = dec.Decode(&req)
	if err != nil {
		return nil, err
	}

	err = s.bot.Notify(chatID, req.Message)
	return &SendNotificationResponse{}, err
}

package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func (s *server) registerRoutes(r *mux.Router) {
	r.HandleFunc("/notify", s.wrap(s.notify)).Methods("POST")
}

func (s *server) notify(w http.ResponseWriter, r *http.Request) (interface{}, error) {
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

package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/transport/dto"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/usecase"
)

const debounceInterval = 500 * time.Millisecond

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// LiveHandler handles /ws/live connections.
type LiveHandler struct {
	hub       *Hub
	analyzeUC *usecase.AnalyzeTextUseCase
	sessionUC *usecase.AcceptRejectFlagUseCase
	applyUC   *usecase.ApplyFixUseCase
	logger    *zap.Logger
}

// NewLiveHandler creates a LiveHandler.
func NewLiveHandler(hub *Hub, analyzeUC *usecase.AnalyzeTextUseCase, sessionUC *usecase.AcceptRejectFlagUseCase, applyUC *usecase.ApplyFixUseCase, logger *zap.Logger) *LiveHandler {
	return &LiveHandler{hub: hub, analyzeUC: analyzeUC, sessionUC: sessionUC, applyUC: applyUC, logger: logger}
}

// --- WS message types ---

type checkMsg struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	TextHash string `json:"textHash"`
}

type flagActionMsg struct {
	Type   string `json:"type"`
	FlagID string `json:"flagId"`
}

type applyAllMsg struct {
	Type    string   `json:"type"`
	FlagIDs []string `json:"flagIds"`
}

type checkResultMsg struct {
	Type      string        `json:"type"`
	TextHash  string        `json:"textHash,omitempty"`
	SessionID string        `json:"session_id"`
	Flags     []dto.FlagDTO `json:"flags"`
	Scores    scoresMsg     `json:"scores"`
}

type scoresMsg struct {
	Cleanliness float64 `json:"cleanliness"`
	Readability float64 `json:"readability"`
}

type ackMsg struct {
	Type    string   `json:"type"`
	Action  string   `json:"action"`
	FlagID  string   `json:"flagId,omitempty"`
	FlagIDs []string `json:"flagIds,omitempty"`
	Status  string   `json:"status"` // "ok" or "error"
	Error   string   `json:"error,omitempty"`
}

type pendingCheck struct {
	text     string
	textHash string
}

// Handle upgrades HTTP to WS and manages the live session.
func (h *LiveHandler) Handle(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	client := &Client{
		conn: conn,
		send: make(chan []byte, 64),
		done: make(chan struct{}),
	}

	h.hub.Register(client)

	// Debounce channel for check/analyze messages
	checkCh := make(chan pendingCheck, 1)

	// Debounce goroutine
	go func() {
		timer := time.NewTimer(0)
		if !timer.Stop() {
			<-timer.C
		}
		defer timer.Stop()
		var pending pendingCheck

		for {
			select {
			case p := <-checkCh:
				pending = p
				timer.Reset(debounceInterval)
			case <-timer.C:
				h.runCheck(client, pending)
			case <-client.done:
				return
			}
		}
	}()

	// Write pump
	go h.writePump(client)

	// Read pump
	conn.SetReadLimit(1 << 20)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// Parse type discriminator first
		var typeOnly struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(msgBytes, &typeOnly); err != nil {
			h.logger.Warn("ws: invalid message", zap.Error(err))
			continue
		}

		switch typeOnly.Type {
		case "check", "analyze":
			var req checkMsg
			if err := json.Unmarshal(msgBytes, &req); err != nil {
				continue
			}
			p := pendingCheck{text: req.Text, textHash: req.TextHash}
			select {
			case checkCh <- p:
			default:
				<-checkCh
				checkCh <- p
			}

		case "accept":
			var req flagActionMsg
			if err := json.Unmarshal(msgBytes, &req); err != nil {
				continue
			}
			h.handleAccept(client, req.FlagID)

		case "reject":
			var req flagActionMsg
			if err := json.Unmarshal(msgBytes, &req); err != nil {
				continue
			}
			h.handleReject(client, req.FlagID)

		case "applyAll":
			var req applyAllMsg
			if err := json.Unmarshal(msgBytes, &req); err != nil {
				continue
			}
			h.handleApplyAll(client, req.FlagIDs)
		}
	}

	close(client.done)
	h.hub.Unregister(client)
	conn.Close()
	return nil
}

func (h *LiveHandler) writePump(client *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	defer client.conn.Close()

	for {
		select {
		case msg, ok := <-client.send:
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-client.done:
			return
		}
	}
}

// --- Check (analysis) ---

func (h *LiveHandler) runCheck(client *Client, p pendingCheck) {
	result, err := h.analyzeUC.Execute(context.Background(), usecase.AnalyzeRequest{
		Text:      p.text,
		SessionID: client.sessionID,
	})
	if err != nil {
		h.logger.Error("ws: check error", zap.Error(err), zap.String("session_id", client.sessionID.String()))
		return
	}
	h.sendJSON(client, checkResultMsg{
		Type:      "check_result",
		TextHash:  p.textHash,
		SessionID: client.sessionID.String(),
		Flags:     dto.FlagsToDTOs(result.Analysis.Flags()),
		Scores: scoresMsg{
			Cleanliness: result.Analysis.CleanlinessScore().Value(),
			Readability: result.Analysis.ReadabilityScore().Value(),
		},
	})
}

// --- Accept / Reject ---

func (h *LiveHandler) handleAccept(client *Client, flagIDStr string) {
	fid, err := parseFlagID(flagIDStr)
	if err != nil {
		h.sendJSON(client, ackMsg{Type: "ack", Action: "accept", Status: "error", Error: "invalid flagId"})
		return
	}
	if err := h.sessionUC.Execute(context.Background(), client.sessionID, fid, true); err != nil {
		h.sendJSON(client, ackMsg{Type: "ack", Action: "accept", Status: "error", Error: err.Error()})
		return
	}
	h.sendJSON(client, ackMsg{Type: "ack", Action: "accept", FlagID: flagIDStr, Status: "ok"})
}

func (h *LiveHandler) handleReject(client *Client, flagIDStr string) {
	fid, err := parseFlagID(flagIDStr)
	if err != nil {
		h.sendJSON(client, ackMsg{Type: "ack", Action: "reject", Status: "error", Error: "invalid flagId"})
		return
	}
	if err := h.sessionUC.Execute(context.Background(), client.sessionID, fid, false); err != nil {
		h.sendJSON(client, ackMsg{Type: "ack", Action: "reject", Status: "error", Error: err.Error()})
		return
	}
	h.sendJSON(client, ackMsg{Type: "ack", Action: "reject", FlagID: flagIDStr, Status: "ok"})
}

// --- Apply All ---

func (h *LiveHandler) handleApplyAll(client *Client, flagIDs []string) {
	var successIDs []string
	var lastErr error

	for _, fidStr := range flagIDs {
		fid, err := parseFlagID(fidStr)
		if err != nil {
			lastErr = err
			continue
		}
		if _, err := h.applyUC.Execute(context.Background(), client.sessionID, fid); err != nil {
			lastErr = err
			continue
		}
		successIDs = append(successIDs, fidStr)
	}

	if lastErr != nil && len(successIDs) == 0 {
		h.sendJSON(client, ackMsg{Type: "ack", Action: "applyAll", Status: "error", Error: lastErr.Error()})
		return
	}
	h.sendJSON(client, ackMsg{Type: "ack", Action: "applyAll", FlagIDs: successIDs, Status: "ok"})
}

// --- Helpers ---

func (h *LiveHandler) sendJSON(client *Client, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		h.logger.Error("ws: marshal error", zap.Error(err))
		return
	}
	select {
	case client.send <- data:
	default:
		h.logger.Warn("WS send buffer full, dropping message",
			zap.String("session_id", client.sessionID.String()),
		)
	}
}

func parseFlagID(s string) (model.FlagID, error) {
	if s == "" {
		return model.FlagID{}, fmt.Errorf("empty flagId")
	}
	var v uint64
	if _, err := fmt.Sscanf(s, "%016x", &v); err != nil {
		return model.FlagID{}, fmt.Errorf("invalid flagId format: %w", err)
	}
	return model.FlagIDFromUint64(v), nil
}

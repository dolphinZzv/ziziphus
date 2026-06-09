package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// ── 协议结构 ──────────────────────────────────────────────

type Frame struct {
	Type    int             `json:"type"`
	ID      string          `json:"id"`
	Payload json.RawMessage `json:"payload"`
}

type LoginReq struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

type LoginRes struct {
	UserID string `json:"user_id"`
	Token  string `json:"token"`
}

type MsgSendPayload struct {
	ConvID      string   `json:"conv_id"`
	ContentType int      `json:"content_type"`
	Body        string   `json:"body"`
	ClientSeq   int64    `json:"client_seq"`
	ReplyTo     int64    `json:"reply_to"`
	Mention     []string `json:"mention"`
}

type MsgPushPayload struct {
	MsgID    int64  `json:"msg_id"`
	ConvID   string `json:"conv_id"`
	SenderID string `json:"sender_id"`
	Body     string `json:"body"`
}

// ── Bot ───────────────────────────────────────────────────

type Bot struct {
	server    string
	account   string
	password  string
	token     string
	userID    string
	clientSeq atomic.Int64
	conn      *websocket.Conn
	done      chan struct{}
}

func NewBot(server, account, password string) *Bot {
	return &Bot{
		server:   server,
		account:  account,
		password: password,
		done:     make(chan struct{}),
	}
}

func (b *Bot) httpPost(path string, req, res any) error {
	body, _ := json.Marshal(req)
	r, err := http.NewRequest("POST", b.server+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json")
	if b.token != "" {
		r.Header.Set("Authorization", "Bearer "+b.token)
	}
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(res)
}

// Register 注册 Bot 账号（首次运行）
func (b *Bot) Register() error {
	var res LoginRes
	err := b.httpPost("/api/v1/users/register", LoginReq{
		Account:  b.account,
		Password: b.password,
	}, &res)
	if err != nil {
		return err
	}
	b.token = res.Token
	b.userID = res.UserID
	log.Printf("[注册] user_id=%s", b.userID)
	return nil
}

// Login 登录获取 token
func (b *Bot) Login() error {
	var res LoginRes
	err := b.httpPost("/api/v1/users/login", LoginReq{
		Account:  b.account,
		Password: b.password,
	}, &res)
	if err != nil {
		return err
	}
	b.token = res.Token
	b.userID = res.UserID
	log.Printf("[登录] user_id=%s", b.userID)
	return nil
}

// Connect 建立 WebSocket 连接
func (b *Bot) Connect() error {
	wsURL := "ws" + b.server[4:] + "/ws?token=" + b.token
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return err
	}
	b.conn = conn
	log.Println("[连接] WebSocket 已连接")
	return nil
}

// SendMessage 发送消息
func (b *Bot) SendMessage(convID, body string, replyTo int64) error {
	seq := b.clientSeq.Add(1)
	payload, _ := json.Marshal(MsgSendPayload{
		ConvID:      convID,
		ContentType: 1,
		Body:        body,
		ClientSeq:   seq,
		ReplyTo:     replyTo,
		Mention:     []string{},
	})
	return b.conn.WriteJSON(Frame{
		Type:    1,
		ID:      fmt.Sprintf("msg_%d", seq),
		Payload: payload,
	})
}

// handleCommand 处理命令消息
func (b *Bot) handleCommand(text, convID string, replyTo int64) (string, bool) {
	switch {
	case text == "/help":
		return "支持的命令:\n  /help  /ping  /time  /echo <text>", true
	case text == "/ping":
		return "pong", true
	case text == "/time":
		return time.Now().Format("15:04:05"), true
	case len(text) > 6 && text[:6] == "/echo ":
		return text[6:], true
	}
	return "", false
}

// handlePush 处理收到的消息
func (b *Bot) handlePush(raw json.RawMessage) {
	var push MsgPushPayload
	json.Unmarshal(raw, &push)

	// 不回复自己
	if push.SenderID == b.userID {
		return
	}

	log.Printf("[消息] from=%s body=%s", push.SenderID, push.Body[:min(len(push.Body), 60)])

	if reply, ok := b.handleCommand(push.Body, push.ConvID, push.MsgID); ok {
		if err := b.SendMessage(push.ConvID, reply, push.MsgID); err != nil {
			log.Printf("[错误] 发送失败: %v", err)
		}
	}
}

// Run 运行主循环
func (b *Bot) Run() error {
	// 尝试登录，失败则注册
	if err := b.Login(); err != nil {
		log.Println("[提示] 登录失败，尝试注册...")
		if err := b.Register(); err != nil {
			return fmt.Errorf("注册失败: %w", err)
		}
		if err := b.Login(); err != nil {
			return fmt.Errorf("登录失败: %w", err)
		}
	}

	if err := b.Connect(); err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}

	// 心跳
	go func() {
		ticker := time.NewTicker(55 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				b.conn.WriteJSON(Frame{Type: 61, ID: "ping", Payload: json.RawMessage("{}")})
			case <-b.done:
				return
			}
		}
	}()

	// 读循环
	go func() {
		for {
			var frame Frame
			if err := b.conn.ReadJSON(&frame); err != nil {
				log.Printf("[断开] %v", err)
				close(b.done)
				return
			}

			switch frame.Type {
			case 11: // MsgPush
				b.handlePush(frame.Payload)
			case 41: // SessionOnline
				var ev struct{ UserID string `json:"user_id"` }
				json.Unmarshal(frame.Payload, &ev)
				log.Printf("[上线] %s", ev.UserID)
			case 42: // SessionOffline
				var ev struct{ UserID string `json:"user_id"` }
				json.Unmarshal(frame.Payload, &ev)
				log.Printf("[下线] %s", ev.UserID)
			}
		}
	}()

	// 等待退出
	<-b.done
	return nil
}

func main() {
	server := flag.String("server", "http://localhost:8080", "server URL")
	account := flag.String("account", "example_bot", "bot account")
	password := flag.String("password", "bot123", "bot password")
	flag.Parse()

	bot := NewBot(*server, *account, *password)

	// 优雅退出
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		bot.conn.Close()
		os.Exit(0)
	}()

	for {
		err := bot.Run()
		log.Printf("[重连] 5秒后重连 (%v)", err)
		time.Sleep(5 * time.Second)
	}
}

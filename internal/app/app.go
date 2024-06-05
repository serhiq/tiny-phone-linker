package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/serhiq/tiny-phone-linker/internal/bot"
	"github.com/serhiq/tiny-phone-linker/internal/config"
	"github.com/serhiq/tiny-phone-linker/internal/logger"
	mysql "github.com/serhiq/tiny-phone-linker/internal/story"
	"go.uber.org/zap"
	tb "gopkg.in/telebot.v3"
	"net/http"
	"sync"
	"time"
)

type App struct {
	config    *config.Config
	ErrChan   chan error
	server    *http.Server
	store     *mysql.Store
	bot       *tb.Bot
	startFunc []func()
	stopFunc  []func(ctx context.Context)
}

func New(cfg *config.Config) *App {
	return &App{
		config:  cfg,
		ErrChan: make(chan error, 1),
	}

}

func (s *App) Start() {
	for _, init := range []func() error{
		s.InitDatabase,
		s.InitBot,
		s.InitApi,
	} {
		if err := init(); err != nil {
			s.ErrChan <- err
		}
	}

	logger.Info("App is starting...")

	for _, start := range s.startFunc {
		go start()
	}
}

func (s *App) Shutdown(ctx context.Context) {
	doneCh := make(chan struct{}, 1)
	defer func() {
		close(doneCh)
	}()
	go func() {
		wg := sync.WaitGroup{}
		for _, delegate := range s.stopFunc {
			wg.Add(1)
			handler := delegate
			go func() {
				handler(ctx)
				wg.Done()
			}()
		}
		wg.Wait()
		doneCh <- struct{}{}
	}()
	select {
	case <-doneCh:
		logger.Info("Good bye!")
	case <-ctx.Done():
		logger.Fatal("Shutdown sequence timeout")
	}
}

//////////////////////////////////////////////////////////////////////////////////////////////////////

func (s *App) InitDatabase() error {
	store, err := mysql.New(s.config)
	if err != nil {
		return err
	}

	s.store = store

	s.addStopDelegate(func(ctx context.Context) {
		err := s.store.Close()
		if err != nil {
			logger.Warn("database: error close database", zap.String("err", err.Error()))
			return
		}

		logger.Info("database: close")
	})

	return err
}

func (s *App) InitApi() error {
	mux := http.NewServeMux()
	mux.HandleFunc("v1/message", s.messageHandler)
	mux.HandleFunc("/ping", s.onPing)

	var server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: time.Minute, //https://medium.com/a-journey-with-go/go-understand-and-mitigate-slowloris-attack-711c1b1403f6
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		Addr:              s.config.Server.Address,
	}
	s.server = server

	s.addStartDelegate(func() {
		address := s.config.Server.Address
		logger.Info("App is starting on", zap.String("address", address))

		if err := server.ListenAndServe(); err != nil {
			s.ErrChan <- err
		}
	})

	s.addStopDelegate(func(ctx context.Context) {
		logger.Info("server is shutting down...")
		if err := s.server.Shutdown(ctx); err != nil {
			logger.Error(err)
		} else {
			logger.Info("successful server shutdown")
		}
	})
	return nil
}

func (s *App) onPing(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		logger.Debug("Received health check")
		w.WriteHeader(http.StatusOK)
	}
}

type Message struct {
	Phone string `json:"phone"`
	Text  string `json:"text"`
}

func (s *App) messageHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	apiKey := r.Header.Get("Authorization")
	if apiKey == "" {
		http.Error(w, "API key is missing", http.StatusUnauthorized)
		return
	}

	if apiKey != s.config.Server.Secret {
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if r.Method == http.MethodPost {
		var message Message
		var buf bytes.Buffer
		_, err := buf.ReadFrom(r.Body)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err = json.Unmarshal(buf.Bytes(), &message); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		id, err := s.store.GetChat(ctx, message.Phone)

		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Phone not found", http.StatusNotFound)
			return
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Error(err)
			return
		}

		_, err = s.bot.Send(tb.ChatID(id), message.Text)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Error(err)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func (s *App) InitBot() (err error) {
	ctx := context.Background()

	switch s.config.Telegram.UpdaterKind {
	case config.LongPolling:
		s.bot, err = initLongPoller(s)
	case config.Webhook:
		s.bot, err = initWebhook(s)
	case config.WebhookCustomCert:
		s.bot, err = initWebhookCustomCert(s)
	}

	if err != nil {
		return err
	}

	logger.Info("telegram bot connected:", zap.String("bot_name", s.bot.Me.Username))

	err = s.bot.SetCommands([]tb.Command{
		{Text: "/phone", Description: "Отправить номер телефона"},
	})

	if err != nil {
		logger.WarnWith("failed to set commands, err: ", err)
		return err
	}

	s.bot.Handle("/start", func(c tb.Context) error {
		var chatId = c.Chat().ID
		phone, err := s.store.GetPhone(ctx, chatId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.Send(bot.START_MESSAGE)
				menu := &tb.ReplyMarkup{ResizeKeyboard: true}
				btnProvideContact := menu.Contact(bot.SEND_PHONE_BUTTON)
				menu.Reply(menu.Row(btnProvideContact))

				return c.Send(bot.REQUEST_CONTACT_PHONE_MESSAGE_FIRST, menu)
			}
			return c.Send(bot.ADD_ACCOUNT_MESSAGE_ERROR)
		}
		if phone != "" {
			return c.Send(bot.START_MESSAGE)
		}
		return nil
	})

	s.bot.Handle("/phone", func(c tb.Context) error {
		menu := &tb.ReplyMarkup{ResizeKeyboard: true,
			OneTimeKeyboard: true}
		btnProvideContact := menu.Contact(bot.SEND_PHONE_BUTTON)
		menu.Reply(menu.Row(btnProvideContact))

		return c.Send(bot.REQUEST_CONTACT_PHONE_MESSAGE, menu)
	})

	s.bot.Handle(tb.OnContact, func(c tb.Context) error {
		var chatId = c.Message().Chat.ID
		var contact = c.Message().Contact

		if contact == nil {
			return nil
		}

		err := s.store.SaveMapping(ctx, contact.PhoneNumber, chatId)
		if err != nil {
			logger.Error(err)
			return c.Send(bot.ADD_ACCOUNT_MESSAGE_ERROR)
		}
		return c.Send(bot.PHONE_SUCCED_SAVED)
	})

	s.bot.Handle(tb.OnText, func(c tb.Context) error {
		return c.Send(bot.MESSAGE_UNKNOWN)
	})

	s.addStartDelegate(func() {
		s.bot.Start()
	})

	s.addStopDelegate(func(_ context.Context) {
		s.bot.Stop()
	})

	return nil
}

func initWebhook(s *App) (*tb.Bot, error) {
	var poller = &tb.Webhook{
		Listen:   s.config.Telegram.WebhookListen,
		Endpoint: &tb.WebhookEndpoint{PublicURL: s.config.Telegram.WebhookUrl},
	}

	return tb.NewBot(tb.Settings{
		Token:       s.config.Telegram.Token,
		Poller:      poller,
		Synchronous: true,
		Verbose:     s.config.EnvType == config.Dev,
	})
}

func initWebhookCustomCert(s *App) (*tb.Bot, error) {
	poll := &tb.Webhook{
		Listen: s.config.Telegram.WebhookListen,
		TLS: &tb.WebhookTLS{
			Key:  s.config.Telegram.WebHookTSLKey,
			Cert: s.config.Telegram.WebHookTSLCrt,
		},
		Endpoint: &tb.WebhookEndpoint{
			PublicURL: s.config.Telegram.WebhookUrl,
			Cert:      s.config.Telegram.WebHookTSLCrt,
		},
	}

	return tb.NewBot(tb.Settings{
		Token:       s.config.Telegram.Token,
		Poller:      poll,
		Synchronous: true,
		Verbose:     s.config.EnvType == config.Dev,
	})
}

func initLongPoller(s *App) (*tb.Bot, error) {
	return tb.NewBot(tb.Settings{
		Token:       s.config.Telegram.Token,
		Poller:      &tb.LongPoller{Timeout: 30 * time.Second},
		Synchronous: false,
		Verbose:     s.config.EnvType == config.Dev,
	})
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func (s *App) addStartDelegate(delegate func()) {
	s.startFunc = append(s.startFunc, delegate)
}

func (s *App) addStopDelegate(delegate func(ctx context.Context)) {
	s.stopFunc = append(s.stopFunc, delegate)
}

/////////////////////////////////////////////////////////////////////////////////////////////////////////////////

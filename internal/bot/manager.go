package bot

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"business-pulse/internal/repository"
)

// Manager подключает Telegram в фоне, чтобы недоступный API не задерживал HTTP-сервер.
type Manager struct {
	mu    sync.RWMutex
	bot   *Bot
	token string
	repo  *repository.Repository
	log   *slog.Logger
}

func NewManager(repo *repository.Repository, log *slog.Logger) *Manager {
	return &Manager{repo: repo, log: log}
}

func (m *Manager) Start(ctx context.Context, token string) {
	if token == "" {
		return
	}
	m.mu.Lock()
	m.token = token
	m.mu.Unlock()
	go m.run(ctx)
}

func (m *Manager) run(ctx context.Context) {
	for ctx.Err() == nil {
		b, err := New(m.token, m.repo, m.log)
		if err == nil {
			m.mu.Lock()
			m.bot = b
			m.mu.Unlock()
			b.Run(ctx)
			m.mu.Lock()
			m.bot = nil
			m.mu.Unlock()
		} else {
			m.log.Warn("telegram-бот пока недоступен, повторю подключение", "err", m.redact(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(15 * time.Second):
		}
	}
}

func (m *Manager) Username() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.bot == nil {
		return ""
	}
	return m.bot.Username()
}

func (m *Manager) SendTo(chatID int64, message string) error {
	m.mu.RLock()
	b := m.bot
	m.mu.RUnlock()
	if b == nil {
		return errors.New("telegram-бот ещё не подключен")
	}
	return m.redact(b.SendTo(chatID, message))
}

func (m *Manager) redact(err error) error {
	if err == nil {
		return nil
	}
	if m.token == "" {
		return err
	}
	return errors.New(strings.ReplaceAll(err.Error(), m.token, "[redacted]"))
}

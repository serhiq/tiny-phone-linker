package mysql

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql" // lint: mysql driver.
	"github.com/serhiq/tiny-phone-linker/internal/config"
)

type Store struct {
	conn *sql.DB
}

func New(s *config.Config) (*Store, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci&time_zone=UTC",
		s.DB.Username,
		s.DB.Password,
		s.DB.Host,
		s.DB.Port,
		s.DB.DatabaseName)

	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	err = conn.Ping()
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &Store{
		conn: conn,
	}, nil

}
func (s Store) GetChat(ctx context.Context, phone string) (chatId int64, err error) {
	row := s.conn.QueryRowContext(ctx, `SELECT telegram_id FROM user_phone_mapping WHERE phone_number = ?`, phone)
	err = row.Scan(&chatId)
	return
}

func (s Store) GetPhone(ctx context.Context, chatId int64) (phone string, err error) {
	row := s.conn.QueryRowContext(ctx, `SELECT phone_number FROM user_phone_mapping WHERE telegram_id = ?`, chatId)
	err = row.Scan(&phone)
	return
}

func (s Store) SaveMapping(ctx context.Context, phone string, chatID int64) error {
	_, err := s.conn.ExecContext(ctx, `
        INSERT INTO user_phone_mapping
		(telegram_id, phone_number)
		VALUES (?, ?)
        ON DUPLICATE KEY UPDATE phone_number = VALUES(phone_number)
    `, chatID, phone)
	return err
}

func (s Store) Close() (err error) {
	return s.conn.Close()
}

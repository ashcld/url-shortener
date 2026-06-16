package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// ClickEvent — событие клика по короткой ссылке.
type ClickEvent struct {
	ShortCode string    `json:"short_code"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Referer   string    `json:"referer"`
	ClickedAt time.Time `json:"clicked_at"`
}

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 5 * time.Second,
		// Async: не блокируем редирект если Kafka тормозит
		Async:        true,
		RequiredAcks: kafka.RequireOne,
	}

	return &Producer{writer: writer}
}

// PublishClick публикует событие клика в Kafka.
func (p *Producer) PublishClick(ctx context.Context, event ClickEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal click event: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(event.ShortCode),
		Value: data,
		Time:  event.ClickedAt,
	}

	if err = p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("write kafka message: %w", err)
	}

	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

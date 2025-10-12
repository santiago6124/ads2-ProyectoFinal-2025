package messaging

import (
	"context"
	"github.com/sirupsen/logrus"
)

type Consumer struct {
	logger *logrus.Logger
}

func NewConsumer(logger *logrus.Logger) *Consumer {
	return &Consumer{logger: logger}
}

func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("Messaging consumer started (stub)")
	return nil
}

func (c *Consumer) Stop() error {
	c.logger.Info("Messaging consumer stopped")
	return nil
}

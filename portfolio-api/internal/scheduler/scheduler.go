package scheduler

import (
	"context"
	"github.com/sirupsen/logrus"
)

type Scheduler struct {
	logger *logrus.Logger
}

func NewScheduler(logger *logrus.Logger) *Scheduler {
	return &Scheduler{logger: logger}
}

func (s *Scheduler) Start(ctx context.Context) error {
	s.logger.Info("Scheduler started (stub)")
	return nil
}

func (s *Scheduler) Stop() error {
	s.logger.Info("Scheduler stopped")
	return nil
}

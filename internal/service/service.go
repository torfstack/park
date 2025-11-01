package service

import (
	"context"
	"fmt"

	"github.com/torfstack/park/internal/auth"
	"github.com/torfstack/park/internal/config"
	"google.golang.org/api/drive/v3"
)

type Service struct {
	drv *drive.Service
	cfg config.Config
}

func NewService(ctx context.Context, cfg config.Config) (*Service, error) {
	drv, err := auth.DriveService(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get drive service: %w", err)
	}
	return &Service{drv, cfg}, nil
}

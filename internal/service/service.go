package service

import (
	"github.com/torfstack/park/internal/auth"
	"github.com/torfstack/park/internal/config"
	"google.golang.org/api/drive/v3"
)

type Service struct {
	drv *drive.Service
	cfg config.Config
}

func NewService(cfg config.Config) *Service {
	drv, err := auth.DriveService()
	if err != nil {
		panic(err)
	}
	return &Service{drv, cfg}
}

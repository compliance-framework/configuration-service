package handler

import (
	"github.com/compliance-framework/configuration-service/service"
	"go.uber.org/zap"
)

type ProfileHandler struct {
	service *service.ProfileService
	sugar   *zap.SugaredLogger
}

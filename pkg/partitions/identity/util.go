package identity

import "go.uber.org/zap"

var Logger = zap.L().With(zap.String("logger", "pkg/partitions/identity"))

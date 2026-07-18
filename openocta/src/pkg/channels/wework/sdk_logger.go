package wework

import "github.com/go-sphere/wecom-aibot-go-sdk/aibot"

// sdkLogger satisfies aibot.Logger without emitting application logs.
type sdkLogger struct{}

func newSDKLogger(_ string) aibot.Logger {
	return &sdkLogger{}
}

func (s *sdkLogger) Debug(_ string, _ ...interface{}) {}

func (s *sdkLogger) Info(_ string, _ ...interface{}) {}

func (s *sdkLogger) Warn(_ string, _ ...interface{}) {}

func (s *sdkLogger) Error(_ string, _ ...interface{}) {}

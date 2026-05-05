package kafka

import (
	"log/slog"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
)

func TestNewService(t *testing.T) {
	t.Run("with logger", func(t *testing.T) {
		logger := slog.Default()
		s := NewService(aws.Config{}, logger)
		assert.Equal(t, logger, s.logger)
	})

	t.Run("with nil logger", func(t *testing.T) {
		s := NewService(aws.Config{}, nil)
		assert.NotNil(t, s.logger)
		assert.Equal(t, slog.Default(), s.logger)
	})
}

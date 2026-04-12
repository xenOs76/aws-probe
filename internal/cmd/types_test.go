package cmd

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
)

func TestDerefInt32(t *testing.T) {
	tests := []struct {
		name  string
		input *int32
		want  int32
	}{
		{name: "non-nil", input: aws.Int32(42), want: 42},
		{name: "nil", input: nil, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, derefInt32(tt.input))
		})
	}
}

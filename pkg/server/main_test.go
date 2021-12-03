package server

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGame(t *testing.T) {
	g, err := NewGame(context.Background(), "test_data/asciiImages", 1)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(g.Images))
	assert.Equal(t, "camel", g.Images[0].Name)

	_, err = NewGame(context.Background(), "foo", 1)
	assert.NotNil(t, err)
}

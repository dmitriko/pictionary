package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGame(t *testing.T) {
	conf := GameConfig{
		ImagesPath: "test_data/asciiImages",
	}
	g, err := NewGame(conf)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(g.Images))
	assert.Equal(t, "camel", g.Images[0].Name)

	badConf := GameConfig{"foo"}
	_, err = NewGame(badConf)
	assert.NotNil(t, err)
}

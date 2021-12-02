package server

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewGame(t *testing.T) {
	conf := GameConfig{
		ImagesPath: "test_data/asciiImages",
		TickPeriod: 1,
	}
	g, err := NewGame(conf)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(g.Images))
	assert.Equal(t, "camel", g.Images[0].Name)

	badConf := GameConfig{"foo", 1}
	_, err = NewGame(badConf)
	assert.NotNil(t, err)
}

func TestNewGameSession(t *testing.T) {
	conf := GameConfig{
		ImagesPath: "test_data/asciiImages",
		TickPeriod: 1,
	}
	g, err := NewGame(conf)
	assert.Nil(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Millisecond*500))
	defer cancel()
	guessChan := make(chan Guess)
	outChan := make(chan string)
	s, err := g.NewSession(ctx, guessChan, outChan)
	assert.Nil(t, err)
	var outLine string
	go func() {
		for {
			outLine = <-outChan
		}
	}()
	go s.Start()
	select {
	case <-ctx.Done():
		assert.Nil(t, ctx.Err())
	case done := <-s.Done:
		assert.True(t, done)
	}

}

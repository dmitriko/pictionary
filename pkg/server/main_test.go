package server

import (
	"context"
	"fmt"
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

func TestNobodyWon(t *testing.T) {
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
	go func() {
		for {
			<-outChan
		}
	}()
	go s.Start()
	select {
	case <-ctx.Done():
		assert.Nil(t, ctx.Err())
	case done := <-s.Done:
		assert.True(t, done)
	}
	assert.Equal(t, NOBODY_WON, s.SentLines[len(s.SentLines)-2])
}

func TestUserWins(t *testing.T) {
	conf := GameConfig{
		ImagesPath: "test_data/asciiImages",
		TickPeriod: 1000,
	}
	g, err := NewGame(conf)
	assert.Nil(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Millisecond*500))
	defer cancel()
	guessChan := make(chan Guess)
	outChan := make(chan string)
	s, err := g.NewSession(ctx, guessChan, outChan)
	assert.Nil(t, err)
	go func() {
		for {
			<-outChan
		}
	}()
	go s.Start()
	go func() {
		guessChan <- Guess{"foo", s.Image.Name}
	}()
	select {
	case <-ctx.Done():
		assert.Nil(t, ctx.Err())
	case done := <-s.Done:
		assert.True(t, done)
	}
	resp := fmt.Sprintf(WINNER_TMPL, "foo", s.Image.Name)
	assert.Equal(t, resp, s.SentLines[len(s.SentLines)-2])
}

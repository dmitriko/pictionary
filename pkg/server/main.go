package server

import (
	"context"
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"strings"
	"time"
)

const (
	NotStarted = iota
	Started
	Ended
)

type GameConfig struct {
	ImagesPath string
	TickPeriod int //miliseconds
}

type Image struct {
	Name  string
	Lines []string
}

type Game struct {
	conf   GameConfig
	Images []*Image
}

type GameSession struct {
	ctx          context.Context
	TickDuration time.Duration
	Image        *Image
	Status       int
	Done         chan bool
	In           <-chan Guess
	Out          chan<- string
	CurrentLine  int
}

type Guess struct {
	UserName  string
	ImageName string
}

func NewGame(conf GameConfig) (*Game, error) {
	g := &Game{conf: conf}
	err := g.LoadImages()
	if err != nil {
		return g, err
	}
	return g, nil
}

//Loads images from directory
func (g *Game) LoadImages() error {
	infos, err := ioutil.ReadDir(g.conf.ImagesPath)
	if err != nil {
		return err
	}
	for _, file := range infos {
		path := filepath.Join(g.conf.ImagesPath, file.Name())
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		img := &Image{
			Name:  file.Name(),
			Lines: strings.Split(string(content), "\n"),
		}
		g.Images = append(g.Images, img)
	}
	return nil
}

func (g *Game) NewSession(ctx context.Context, in chan Guess, out chan string) (*GameSession, error) {
	rand.Seed(time.Now().Unix())
	s := &GameSession{
		ctx:          ctx,
		TickDuration: time.Duration(g.conf.TickPeriod),
		Image:        g.Images[rand.Intn(len(g.Images))],
		Done:         make(chan bool),
		In:           in,
		Out:          out,
	}
	return s, nil
}

func (s *GameSession) Start() {
	ticker := time.NewTicker(s.TickDuration)
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if len(s.Image.Lines) <= s.CurrentLine {
				s.Out <- "Nobody wins."
				s.Done <- true
			} else {
				s.Out <- s.Image.Lines[s.CurrentLine]
				s.CurrentLine++
			}

		}
	}
}

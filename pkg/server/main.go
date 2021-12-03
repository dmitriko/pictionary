package server

import (
	"io/ioutil"
	"net"
	"path/filepath"
	"strings"
)

const (
	NOBODY_WON_TMPL  = "Nobody won. It was %s."
	LAST_LINE        = "Game ended!"
	WINNER_TMPL      = "Player %s is the winner! The correct guess: %s."
	WRONG_GUESS_TMPL = "Wrong guess: %s."
)

type GameConfig struct {
	ImagesPath string
	TickPeriod int //miliseconds
}

type Image struct {
	Name  string
	Lines []string
}

type Player struct {
	conn net.Conn
	Name string
}

type Game struct {
	conf    GameConfig
	Images  []*Image
	Players map[string]*Player
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

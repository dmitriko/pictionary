package server

import (
	"io/ioutil"
	"path/filepath"
	"strings"
)

type GameConfig struct {
	ImagesPath string
}

type Image struct {
	Name  string
	Lines []string
}

type Game struct {
	conf   GameConfig
	Images []*Image
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

package server

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"path/filepath"
	"strings"
	"time"
)

const (
	NotStarted = iota
	Started
	Ended
)

const (
	NOBODY_WON       = "Nobody won."
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
	In           chan *Guess
	Out          chan string
	CurrentLine  int
	SentLines    []string
}

type Guess struct {
	UserName  string
	ImageName string
	OK        bool
	Resp      string
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

func (g *Game) NewSession(ctx context.Context, in chan *Guess, out chan string) (*GameSession, error) {
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
		case guess := <-s.In:
			if strings.TrimSpace(guess.ImageName) == s.Image.Name {
				s.Out <- fmt.Sprintf(WINNER_TMPL, guess.UserName, s.Image.Name)
				s.Out <- LAST_LINE
				s.SentLines = append(s.SentLines, fmt.Sprintf(WINNER_TMPL, guess.UserName, s.Image.Name))
				s.SentLines = append(s.SentLines, LAST_LINE)
				s.Done <- true
				return
			} else {
				guess.OK = false
				guess.Resp = fmt.Sprintf(WRONG_GUESS_TMPL, strings.TrimSpace(guess.ImageName))
			}
		case <-ticker.C:
			if len(s.Image.Lines) <= s.CurrentLine {
				s.Out <- NOBODY_WON
				s.Out <- LAST_LINE
				s.SentLines = append(s.SentLines, NOBODY_WON)
				s.SentLines = append(s.SentLines, LAST_LINE)
				s.Done <- true
				return
			} else {
				line := s.Image.Lines[s.CurrentLine]
				s.SentLines = append(s.SentLines, line)
				s.Out <- line
				s.CurrentLine++
			}

		}
	}
}

type UserSession struct {
	ctx      context.Context
	conn     net.Conn
	UserName string
	In       chan string
	Out      chan *Guess
}

func (us *UserSession) Start() {
	fromUser := make(chan string)
	go func(fromUser chan string) {
		msg, _ := bufio.NewReader(us.conn).ReadString('\n')
		fromUser <- msg
	}(fromUser)

	for {
		select {
		case <-us.ctx.Done():
			return
		case msg := <-fromUser:
			if us.UserName == "" && msg != "" {
				us.UserName = msg
			}
			if us.UserName != "" && msg != "" {
				us.Out <- &Guess{UserName: us.UserName, ImageName: msg}
			}
		case toUser := <-us.In:
			us.conn.Write([]byte(toUser + "\n"))
		}

	}
}

type Server struct {
	ctx          context.Context
	Address      string
	ImagesPath   string
	UserSessions []*UserSession
	fromUsers    chan *Guess
	toUsers      chan string
}

//Waits for users and start when there is at least one
func (s *Server) MustWaitAndStartGame() {
	conf := GameConfig{
		ImagesPath: s.ImagesPath,
		TickPeriod: 1000, // 1 sec
	}
	game, err := NewGame(conf)
	if err != nil {
		log.Fatal(err)
	}
	toGame := make(chan *Guess)
	fromGame := make(chan string)

	gameSession, err := game.NewSession(s.ctx, toGame, fromGame)
	for {
		for _, us := range s.UserSessions {
			if us.UserName != "" {
				break
			}
		}
		time.Sleep(1 * time.Second)
	}
	go gameSession.Start()
	for {
		select {
		case <-s.ctx.Done():
			return
		case guess := <-s.fromUsers:
			if guess.ImageName != gameSession.Image.Name {
				s.UserSendMsg(guess.UserName, fmt.Sprintf(WRONG_GUESS_TMPL, guess.ImageName))
			} else {
				gameSession.In <- guess
			}
		case msg := <-gameSession.Out:
			for _, us := range s.UserSessions {
				us.In <- msg
			}
		}
	}
}

func (s *Server) UserSendMsg(userName, imageName string) {

}
func (s *Server) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.Address)
	if err != nil {
		return err
	}
	defer ln.Close()
	go s.MustWaitAndStartGame()
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			return err
		}
		toUser := make(chan string)
		us := &UserSession{
			ctx:  ctx,
			conn: conn,
			Out:  s.fromUsers,
			In:   toUser,
		}
		go us.Start()
		s.UserSessions = append(s.UserSessions, us)
		us.In <- "Welcome to Ascii drawing!"
		us.In <- "Please, enter your name:"
	}

}

func NewServer(address, path string) *Server {
	toUsers := make(chan string)
	fromUsers := make(chan *Guess)
	return &Server{
		ctx:        context.Background(),
		Address:    address,
		ImagesPath: path,
		toUsers:    toUsers,
		fromUsers:  fromUsers,
	}
}

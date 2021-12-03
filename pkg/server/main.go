package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/xid"
)

const (
	NOBODY_WON_TMPL = "Nobody won. It was %s."
	LAST_LINE       = "Game ended!"
	WINNER_TMPL     = "Player %s is the winner! The correct guess: %s."
	WRONG_TMPL      = "Wrong guess: %s."
)

type Image struct {
	Name  string
	Lines []string
}

type Player struct {
	conn io.ReadWriteCloser
	ID   string
	Name string
	To   chan string
	From chan string
	Done chan bool
}

type UserMsg struct {
	UserID string
	Text   string
}

//Creates new Player for net.Conn
func NewPlayer(conn io.ReadWriteCloser) *Player {
	return &Player{
		conn: conn,
		ID:   xid.New().String(),
		From: make(chan string),
		To:   make(chan string),
		Done: make(chan bool),
	}
}

// Sending/Receiving bytes to/from user
func (p *Player) Communicate(ctx context.Context, toGame chan *UserMsg) {
	defer p.conn.Close()
	go func() { // Reading from socket and sending to From channel
		for {
			msg, err := bufio.NewReader(p.conn).ReadString('\n')
			if err != nil {
				close(p.Done)
			}
			p.From <- msg
		}
	}()
	go func() { // Sending to socket text received via To channel
		for {
			select {
			case msg := <-p.To:
				_, err := p.conn.Write([]byte(msg + "\n"))
				if err != nil {
					close(p.Done)
					return
				}
			case <-p.Done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	for {
		select {
		case <-p.Done:
			return
		case msg := <-p.From:
			if p.Name == "" { // If text recived my name is not set assumes it is name
				p.Name = strings.TrimSpace(msg)
			} else { // else send it to game's channel
				toGame <- &UserMsg{UserID: p.ID, Text: msg}
			}
		case <-ctx.Done():
			return
		}

	}
}

type Game struct {
	ctx        context.Context
	imagesPath string
	tickPeriod int
	Images     []*Image
	Players    map[string]*Player
	fromUsers  chan *UserMsg
	currImage  *Image
	currLine   int
	Done       bool
	sync.Mutex
}

// Creates new Game for give path to images dir and period to wait before sending next line in millisecond
func NewGame(ctx context.Context, imagesPath string, tickPeriod int) (*Game, error) {
	g := &Game{
		ctx:        ctx,
		imagesPath: imagesPath,
		tickPeriod: tickPeriod,
		fromUsers:  make(chan *UserMsg),
		Players:    map[string]*Player{},
	}
	err := g.LoadImages()
	if err != nil {
		return nil, err
	}
	if len(g.Images) == 0 {
		return nil, errors.New("Empty images folder.")
	}
	rand.Seed(time.Now().Unix())
	g.currImage = g.Images[rand.Intn(len(g.Images))]
	if err != nil {
		return g, err
	}
	return g, nil
}

//Loads images from directory
func (g *Game) LoadImages() error {
	infos, err := ioutil.ReadDir(g.imagesPath)
	if err != nil {
		return err
	}
	for _, file := range infos {
		path := filepath.Join(g.imagesPath, file.Name())
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

// Handles new tcp socket conntection
// Adds Player record into map, starts communication
func (g *Game) HandleConn(conn io.ReadWriteCloser) {
	log.Print("User connected")
	player := NewPlayer(conn)
	g.Lock()
	g.Players[player.ID] = player
	g.Unlock()
	go player.Communicate(g.ctx, g.fromUsers)
	player.To <- "Welcome to Ascii drawing!"
	player.To <- "Please, enter your name:"
}

// Checks if there at least one player with name set
// wait 1 sec othervise and check again
func waitUserWithName(players map[string]*Player) {
	for {
		for _, player := range players {
			if player.Name != "" {
				return
			}
		}
		time.Sleep(1 * time.Second)
	}
}

// Checks is user sent correct guess
func (g *Game) handleUserMsg(msg *UserMsg) {
	player, exists := g.Players[msg.UserID]
	if !exists {
		return
	}
	text := strings.TrimSpace(msg.Text)
	if text == g.currImage.Name {
		g.sendTextAllPlayers(fmt.Sprintf(WINNER_TMPL, player.Name, g.currImage.Name))
		g.Done = true
	} else {
		player.To <- fmt.Sprintf(WRONG_TMPL, text)
	}

}

func (g *Game) Stop() {
	g.sendTextAllPlayers(LAST_LINE)
	g.Done = true
}

func (g *Game) NobodyWon() {
	g.sendTextAllPlayers(fmt.Sprintf(NOBODY_WON_TMPL, g.currImage.Name))
}

func (g *Game) sendNextLine() {
	if len(g.currImage.Lines) > g.currLine {
		g.sendTextAllPlayers(g.currImage.Lines[g.currLine])
		g.currLine++
	}
}

func (g *Game) sendTextAllPlayers(text string) {
	for _, player := range g.Players {
		if player.Name != "" {
			go func() {
				player.To <- text
			}()
		}
	}
}

//Waits for at least one user and start game
func (g *Game) MustWaitAndStart() {
	waitUserWithName(g.Players)
	ticker := time.NewTicker(time.Duration(g.tickPeriod) * time.Millisecond)
	defer g.Stop()
	for {
		select {
		case <-g.ctx.Done():
			return
		case msg := <-g.fromUsers:
			g.handleUserMsg(msg)
		case <-ticker.C:
			if g.Done {
				return
			}
			if g.currLine >= len(g.currImage.Lines) {
				g.NobodyWon()
				return
			}
			g.sendNextLine()
		}
	}
}

type Server struct {
	Address      string
	ImagesPath   string
	TickDuration int
}

func (s *Server) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.Address)
	if err != nil {
		return err
	}
	defer ln.Close()
	game, err := NewGame(ctx, s.ImagesPath, s.TickDuration)
	go game.MustWaitAndStart()
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			return err
		}
		go game.HandleConn(conn)
	}

}

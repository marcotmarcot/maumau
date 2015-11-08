package main

import (
	"bytes"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"math/big"
	"strconv"
)

var startingCards = flag.Int("starting_cards", 5, "Number of cards each player should start with.")
var numPlayers = flag.Int("num_players", 2, "Number of players of the game.")
var numGames = flag.Int("num_games", 1000, "Number of games that will be played.")
var debug = flag.Bool("debug", false, "Print debug information")

type suit int

const (
	none suit = iota
	spades
	hearts
	diamonds
	clubs
)

type card struct {
	n int
	s suit
}

func (c *card) String() string {
	return strconv.Itoa(c.n) + "/" + strconv.Itoa(int(c.s))
}

type deck struct {
	cs []*card
}

func randInt(max int) int {
	b, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		log.Fatal(err)
	}
	return int(b.Int64())
}

func newDeck() *deck {
	d := &deck{}
	for n := 1; n <= 13; n++ {
		for s := spades; s <= clubs; s++ {
			d.cs = append(d.cs, &card{n, s})
		}
	}
	d.shuffle()
	return d
}

func (d *deck) shuffle() {
	l := len(d.cs)
	s := make([]*card, l)
	for i, c := range d.cs {
		r := randInt(i + 1)
		s[i] = s[r]
		s[r] = c
	}
	d.cs = s
}

func (d *deck) getCard() *card {
	if len(d.cs) == 0 {
		return nil
	}
	c := d.cs[0]
	d.cs = d.cs[1:]
	return c
}

type player struct {
	cs []*card
}

func (p *player) addCard(c *card) {
	p.cs = append(p.cs, c)
}

func (p *player) play(current *card, asked suit, d *deck) (*card, suit) {
	is := p.find(current, asked)
	if len(is) == 0 {
		return nil, none
	}
	i := p.choose(is)
	c := p.cs[i]
	p.cs = append(p.cs[:i], p.cs[i+1:]...)
	if c.n == 11 {
		return c, p.chooseSuit()
	}
	return c, none
}

func (p *player) find(current *card, asked suit) []int {
	var is []int
	for i, c := range p.cs {
		if c.n == 11 {
			is = append(is, i)
			continue
		}
		if asked != none {
			if c.s == asked {
				is = append(is, i)
			}
			continue
		}
		if c.s == current.s || c.n == current.n {
			is = append(is, i)
		}
	}
	return is
}

func (p *player) choose(is []int) int {
	return is[randInt(len(is))]
}

func (p *player) chooseSuit() suit {
	return suit(randInt(4) + 1)
}

func (p *player) String() string {
	return fmt.Sprintf("%v", p.cs)
}

type game struct {
	ps []*player
	ip int
	d *deck
	current *card
	asked suit
	order int
	garbage []*card
}

func newGame() *game {
	g := &game{}
	g.d = newDeck()
	for np := 0; np < *numPlayers; np++ {
		p := &player{}
		for nc := 0; nc < *startingCards; nc++ {
			p.addCard(g.getCard())
		}
		g.ps = append(g.ps, p)
	}
	g.current = g.getCard()
	g.order = 1
	return g
}

func (g *game) play() int {
	for {
		g.garbage = append(g.garbage, g.current)
		current, asked := g.player().play(g.current, g.asked, g.d)
		if *debug {
			fmt.Println(g, current, asked)
		}
		if current == nil {
			g.addCard()
			g.next()
			continue
		}
		check(current, asked, g.current, g.asked)
		if g.end() {
			return g.ip
		}
		g.current, g.asked = current, asked
		switch g.current.n {
		case  1:
			g.next()
		case 7:
			g.next()
			g.addCard()
			g.addCard()
		case 12:
			g.order *= -1
		}
		g.next()
	}
}

func (g *game) player() *player {
	return g.ps[g.ip]
}

func (g *game) next() {
	g.ip = (g.ip + g.order + len(g.ps)) % len(g.ps)
}

func (g *game) getCard() *card {
	c := g.d.getCard()
	if c != nil {
		return c
	}
	if g.garbage == nil {
		log.Fatal("empty garbage")
	}
	g.d.cs = g.garbage
	g.d.shuffle()
	g.garbage = nil
	return g.d.getCard()
}

func (g *game) addCard() {
	g.player().addCard(g.getCard())
}

func (g *game) end() bool {
	return len(g.player().cs) == 0
}

func (g *game) String() string {
	var b bytes.Buffer
	b.WriteString(strconv.Itoa(g.ip))
	b.WriteString(" ")
	b.WriteString(g.current.String())
	b.WriteString(" ")
	b.WriteString(strconv.Itoa(int(g.asked)))
	b.WriteString(" ")
	b.WriteString(strconv.Itoa(g.order))
	b.WriteString(" ")
	for _, p := range g.ps {
		b.WriteString(p.String())
		b.WriteString(" ")
	}
	return b.String()
}

func check(newc *card, newa suit, oldc *card, olda suit) {
	if newc.n == 11 {
		if newa == none {
			log.Fatal("newc.n = 11, newa = none")
		}
		return
	}
	if newa != none {
		log.Fatalf("newc.n = %v, newa = %v", newc.n, newa)
	}
	if olda != none {
		if newc.s != olda {
			log.Fatalf("newc.s = %v, olda = %v", newc.s, olda)
		}
		return
	}
	if newc.n != oldc.n && newc.s != newc.s {
		log.Fatalf("newc = %v, oldc = %v", newc, oldc)
	}
}

func main() {
	flag.Parse()
	w := make([]int, *numPlayers, *numPlayers)
	for i := 0; i < *numGames; i++ {
		g := newGame()
		w[g.play()]++
	}
	for i := 0; i < *numPlayers; i++ {
		fmt.Println(w[i])
	}
}

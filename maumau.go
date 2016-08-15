package main

import (
	"bytes"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"math"
	"math/big"
	"strconv"
	"strings"
)

var (
	startingCards = flag.Int("starting_cards", 5, "Number of cards each player should start with.")
	numGames      = flag.Int("num_games", 100, "Number of games that will be played.")
	numTests      = flag.Int("num_tests", 100, "Number of tests to be performed.")
	ais           = flag.String("ais", "randomAI,randomAI", "AI algorithms to be used by each player separated by comma. The first player is the main one.")
	randomStart   = flag.Bool("random_start", true, "Defines who starts randomly. If false, the first player always starts.")
	decks         = flag.Int("decks", 1, "Number of card decks to be used.")
	debug         = flag.Bool("debug", false, "Print debug information")
)

func main() {
	flag.Parse()
	var results []int
	total := 0
	for i := 0; i < *numTests; i++ {
		res := runGame()
		results = append(results, res)
		total += res
	}
	avg := float64(total) / float64(*numTests)
	variance := 0.0
	for _, res := range results {
		term := float64(res) - avg
		variance += term * term
	}
	variance /= float64(*numTests)
	fmt.Printf("%v+-%v\n", avg, math.Sqrt(variance))
}

func runGame() int {
	w := 0
	for i := 0; i < *numGames; i++ {
		g := newGame()
		if g.play() == 0 {
			w++
		}
	}
	return w
}

type game struct {
	players []*player
	playing int
	deck    *deck
	top     *card
	asked   suit
	order   int
	garbage []*card
}

func newGame() *game {
	g := &game{}
	if *randomStart {
		if *debug {
			fmt.Println("random_start")
		}
		g.playing = randInt(2)
	}
	g.deck = newDeck()
	for _, aiName := range strings.Split(*ais, ",") {
		g.players = append(g.players, newPlayer(aiName))
	}
	for nc := 0; nc < *startingCards; nc++ {
		for _, player := range g.players {
			player.addCard(g.getCard())
		}
	}
	g.top = g.getCard()
	g.order = 1
	return g
}

func (g *game) play() int {
	for {
		top, asked := g.player().play(g.top, g.asked, g.deck)
		if *debug {
			fmt.Println(g, top, asked)
		}
		if top == nil {
			g.addCard()
			g.next()
			continue
		}
		isPlayValid(top, asked, g.top, g.asked)
		if g.end() {
			return g.playing
		}
		g.garbage = append(g.garbage, g.top)
		g.top, g.asked = top, asked
		switch g.top.n {
		case 1:
		case 12:
			if len(g.players) == 2 {
				g.next()
			} else {
				g.order *= -1
			}
		case 7:
			g.next()
			g.addCard()
			g.addCard()
		}
		g.next()
	}
}

func (g *game) player() *player {
	return g.players[g.playing]
}

func (g *game) next() {
	g.playing = (g.playing + g.order + len(g.players)) % len(g.players)
}

func (g *game) getCard() *card {
	c := g.deck.getCard()
	if c != nil {
		return c
	}
	if g.garbage == nil {
		log.Fatal("empty garbage")
	}
	g.deck.cs = g.garbage
	g.deck.shuffle()
	g.garbage = nil
	return g.deck.getCard()
}

func (g *game) addCard() {
	g.player().addCard(g.getCard())
}

func (g *game) end() bool {
	return g.player().end()
}

func (g *game) String() string {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("-> %v %v %v %v %v %v ", len(g.deck.cs), len(g.garbage), g.playing, g.top, g.asked, g.order))
	for _, p := range g.players {
		b.WriteString(p.String())
		b.WriteString(" ")
	}
	return b.String()
}

type deck struct {
	cs []*card
}

func newDeck() *deck {
	d := &deck{}
	for n := 1; n <= 13; n++ {
		for s := spades; s <= clubs; s++ {
			for i := 0; i < *decks; i++ {
				d.cs = append(d.cs, &card{n, s})
			}
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

type card struct {
	n int
	s suit
}

func (c *card) String() string {
	return strconv.Itoa(c.n) + "/" + strconv.Itoa(int(c.s))
}

type suit int

const (
	noSuit suit = iota
	spades
	hearts
	diamonds
	clubs
)

type player struct {
	cs []*card
	ai ai
}

func newPlayer(aiName string) *player {
	return &player{nil, aiImplementation[aiName]}
}

func (p *player) addCard(c *card) {
	p.cs = append(p.cs, c)
}

func (p *player) play(top *card, asked suit, d *deck) (*card, suit) {
	i, s := p.ai(p.cs, top, asked, d)
	if i == -1 {
		return nil, noSuit
	}
	c := p.cs[i]
	p.cs = append(p.cs[:i], p.cs[i+1:]...)
	if c.n != 11 {
		return c, noSuit
	}
	return c, s
}

func (p *player) end() bool {
	return len(p.cs) == 0
}

func (p *player) String() string {
	return fmt.Sprintf("%v", p.cs)
}

// ai returns the index of the card in cs that should be played, -1 if there's
// no card to play, and suit if the card is a 11.
type ai func(cs []*card, top *card, asked suit, d *deck) (int, suit)

var aiImplementation = map[string]ai{
	"randomAI":    randomAI,
	"onlyFirstAI": onlyFirstAI,
	"onlyBuyAI":   onlyBuyAI,
}

func randomAI(cs []*card, top *card, asked suit, d *deck) (int, suit) {
	is := validIndexes(cs, top, asked)
	if len(is) == 0 {
		return -1, noSuit
	}
	return is[randInt(len(is))], suit(randInt(4) + 1)
}

func onlyFirstAI(cs []*card, top *card, asked suit, d *deck) (int, suit) {
	if cs[0].n == 11 {
		return 0, spades
	}
	if asked != noSuit {
		if cs[0].s == asked {
			return 0, noSuit
		}
		return -1, noSuit
	}
	if cs[0].n == top.n || cs[0].s == top.s {
		return 0, noSuit
	}
	return -1, noSuit
}

func onlyBuyAI(cs []*card, top *card, asked suit, d *deck) (int, suit) {
	return -1, noSuit
}

func validIndexes(cs []*card, top *card, asked suit) []int {
	var is []int
	for i, c := range cs {
		if c.n == 11 {
			is = append(is, i)
			continue
		}
		if asked != noSuit {
			if c.s == asked {
				is = append(is, i)
			}
			continue
		}
		if c.s == top.s || c.n == top.n {
			is = append(is, i)
		}
	}
	return is
}

func isPlayValid(newc *card, newa suit, oldc *card, olda suit) {
	if newc.n == 11 {
		if newa == noSuit {
			log.Fatal("newc.n = 11, newa = noSuit")
		}
		return
	}
	if newa != noSuit {
		log.Fatalf("newc.n = %v, newa = %v", newc.n, newa)
	}
	if olda != noSuit {
		if newc.s != olda {
			log.Fatalf("newc.s = %v, olda = %v", newc.s, olda)
		}
		return
	}
	if newc.n != oldc.n && newc.s != newc.s {
		log.Fatalf("newc = %v, oldc = %v", newc, oldc)
	}
}

// randInt returns a random number from 0 to max - 1.
func randInt(max int) int {
	b, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		log.Fatal(err)
	}
	return int(b.Int64())
}

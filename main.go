package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/MagicTheGathering/mtg-sdk-go"
)

var template string = `<head>
<link rel="stylesheet" type="text/css" href="cubetutor.css">
</head>

<script src="https://ajax.googleapis.com/ajax/libs/jquery/1.9.1/jquery.min.js" charset="UTF-8"></script>
<script src="imgPreview.js"></script>

%s

<script type="text/javascript">
$(document).ready(function() {
$('.cardPreview').imgPreview();

});
</script>
`

type (
	rarityMap map[string]colorMap
	colorMap  map[string]cmcMap
	cmcMap    map[int][]*mtg.Card
)

type cardstate uint8

const (
	notyetseen cardstate = iota
	inthelist
	seen
)

var (
	promos  = true
	mRarity = rarityMap{}
	names   = map[string]cardstate{}
	// overrides
	rarities = map[string]string{}
)

// formatting in -file:
// file should be split by newline, one cardname per line
// # is a comment
// [Mythic] Card Name sets Card Name to Mythic even when its a common

func printUsage() {
	fmt.Println("usage: go run main.go -set <SETCODE> [-nopromo]")
	fmt.Println("usage: go run main.go -file <filename> [-nopromo]")
	os.Exit(1)
}

func main() {
	// TODO: use cmd line lib like cobra
	if !(len(os.Args) == 3 || (len(os.Args) == 4 && os.Args[3] == "-nopromo")) {
		printUsage()
	}
	mode := os.Args[1]
	arg := os.Args[2]
	if len(os.Args) > 3 && os.Args[3] == "-nopromo" {
		promos = false
	}
	switch mode {
	case "-set":
		set(arg)
	case "-file":
		file(arg)
	default:
		printUsage()
	}
}

// download a whole set definition and print the overview
func set(code string) {
	query := mtg.NewQuery().Where(mtg.CardSet, code)
	cards, err := query.All()
	if err != nil {
		panic(err)
	}
	for _, card := range cards {
		addToMaps(card, getRarity(card))
	}
	writeHTML()
}

// do the same but instead of all the cards in a set,
// print the overview for all the cards in the file.
// library does encoding of cardnames for us
func file(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	cards := []string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		cardname := scanner.Text()
		if cardname == "" || strings.HasPrefix(cardname, "#") {
			continue
		}
		if strings.HasPrefix(cardname, "[") {
			split := strings.Split(cardname[1:], "] ")
			rarityOverride := split[0]
			cardname = split[1]
			rarities[cardname] = rarityOverride
		}
		cards = append(cards, cardname)
		names[cardname] = inthelist
	}
	for i := 0; i <= len(cards)/10; i++ {
		end := 10 * (i + 1)
		if end > len(cards) {
			end = len(cards)
		}
		normalcards := strings.Join(cards[10*i:end], "|")
		query := mtg.NewQuery().Where(mtg.CardName, normalcards)
		fmt.Println(query)
		resp, err := query.All()
		if err != nil {
			panic(err)
		}
		for _, card := range resp {
			// assumption: cardsets generally have three letters
			// and promos have a P in front (i.e. SOI vs PSOI)
			if !promos && len(card.Set) == 4 { //&& card.Set[0] == 80 {
				continue
			}
			if !promos && card.Set == "PRM" {
				continue
			}
			if names[card.Name] != inthelist {
				continue
			}
			names[card.Name] = seen
			rarity := getRarity(card)
			if value, ok := rarities[card.Name]; ok {
				rarity = value
			}
			addToMaps(card, rarity)
		}
	}
	// multiple exact matches are _broken_ on the API level
	// API is broken for exact match on some cards with special characters in them
	// try to find card without exact match in that case
	// so we'll have to send requests one-by-one for these cards :(
	writeHTML()
}

func writeHTML() {
	f, err := os.Create("out.html")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.WriteString(fmt.Sprintf(template, printMap(mRarity)))
}

func addToMaps(card *mtg.Card, rarity string) {
	c := getColor(card)
	mc, ok := mRarity[rarity]
	if !ok {
		mc = colorMap{}
		mRarity[rarity] = mc
	}
	mcmc, ok := mc[c]
	if !ok {
		mcmc = cmcMap{}
		mc[c] = mcmc
	}
	cmc := int(card.CMC)
	mcmc[cmc] = append(mcmc[cmc], card)
}

var colors = []string{
	"White", "Blue", "Black", "Red", "Green", "Multicolor", "Colorless",
}

func getColor(card *mtg.Card) string {
	if len(card.Colors) > 1 {
		return "Multicolor"
	} else if len(card.Colors) == 1 {
		return card.Colors[0]
	}
	return "Colorless"
}

func getRarity(card *mtg.Card) string {
	r := card.Rarity
	if r == "Common" || r == "Uncommon" || r == "Rare" || r == "Mythic" {
		return r
	}
	return "Other"
}

func printMap(mr rarityMap) string {
	return printRarity(mr["Common"]) + printRarity(mr["Uncommon"]) + printRarity(mr["Rare"]) + printRarity(mr["Mythic"]) + printRarity(mr["Other"])
}

func printRarity(cmap colorMap) string {
	var s string
	s += "<div id=\"listContainer\">\n"
	for _, color := range colors {
		mcmc := cmap[color]
		s += printColor(color, mcmc)
	}
	s += "</div>\n"
	return s
}

func printColor(color string, mcmc cmcMap) string {
	var s string
	s += fmt.Sprintf("<div class=\"viewCubeColumn %sColumn\">\n", strings.ToLower(color))
	keys := []int{}
	var numColor int
	for k := range mcmc {
		keys = append(keys, k)
		numColor += len(mcmc[k])
	}
	s += fmt.Sprintf("<p class=\"bigColumnTitle\">%s (%d)</p>", color, numColor)
	sort.Ints(keys)
	for _, k := range keys {
		list := mcmc[k]
		for _, card := range list {
			switch len(card.Names) {
			case 0:
				name := strings.Replace(strings.ToLower(card.Name), " ", "%20", -1)
				s += fmt.Sprintf("<a rel=\"nofollow\" class=\"cardPreview\" data-image=\"http://d1f83aa4yffcdn.cloudfront.net/%s/%s.jpg\">%s</a>\n", card.Set, name, card.Name)
			case 2:
				name1 := strings.Replace(strings.ToLower(card.Names[0]), " ", "%20", -1)
				// name2 := strings.Replace(strings.ToLower(card.Names[1]), " ", "%20", -1)
				// TODO: at least three formats: name1_flip.jpg, name1name2.jpg, name1name2_flip.jpg
				s += fmt.Sprintf("<a rel=\"nofollow\" class=\"cardPreview\" data-image=\"http://d1f83aa4yffcdn.cloudfront.net/%s/%s_flip.jpg\">%s</a>\n", card.Set, name1, card.Name)
			default:
				fmt.Println("ERR: card with weird number of names: ", card.Names)
			}
		}
		s += "<p class=\"cmcDivider\"></p>\n"
	}
	s += "</div>\n"
	return s
}

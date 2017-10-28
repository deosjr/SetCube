package main

import (
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

var (
	mRarity = rarityMap{}
)

func main() {

	if len(os.Args) != 2 {
		fmt.Println("usage: go run main.go SETCODE(3 characters)")
		os.Exit(1)
	}
	setCode := os.Args[1]

	cards, err := mtg.NewQuery().Where(mtg.CardSet, setCode).All()
	if err != nil {
		panic(err)
	}
	for _, card := range cards {
		addToMaps(card)
	}
	f, err := os.Create("out.html")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.WriteString(fmt.Sprintf(template, printMap(mRarity)))
}

func addToMaps(card *mtg.Card) {
	r := getRarity(card)
	c := getColor(card)
	mc, ok := mRarity[r]
	if !ok {
		mc = colorMap{}
		mRarity[r] = mc
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
	if card.Rarity == "Common" || card.Rarity == "Uncommon" {
		return card.Rarity
	}
	return "Rare"
}

func printMap(mr rarityMap) string {
	return printRarity(mr["Common"]) + printRarity(mr["Uncommon"]) + printRarity(mr["Rare"])
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
	for k := range mcmc {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		list := mcmc[k]
		for _, card := range list {
			name := strings.Replace(strings.ToLower(card.Name), " ", "%20", -1)
			s += fmt.Sprintf("<a rel=\"nofollow\" class=\"cardPreview\" data-image=\"http://d1f83aa4yffcdn.cloudfront.net/%s/%s.jpg\">%s</a>\n", card.Set, name, card.Name)
		}
		s += "<p class=\"cmcDivider\"></p>\n"
	}
	s += "</div>\n"
	return s
}

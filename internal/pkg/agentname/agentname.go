package agentname

import (
	"fmt"
	"math/rand/v2"
)

var adjectives = []string{
	"Swift",
	"Silent",
	"Brave",
	"Bright",
	"Calm",
	"Rapid",
	"Golden",
	"Silver",
	"Clever",
	"Bold",
}

var nouns = []string{
	"Falcon",
	"Tiger",
	"Wolf",
	"Eagle",
	"Dragon",
	"Phoenix",
	"Otter",
	"Panther",
	"Voyager",
	"Comet",
}

func Generate() string {
	return fmt.Sprintf("%s %s",
		adjectives[rand.IntN(len(adjectives))],
		nouns[rand.IntN(len(nouns))],
	)
}

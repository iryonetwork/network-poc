package personaldata

import (
	"crypto/rand"
	"math/big"
)

func newName() string {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(names))))
	if err != nil {
		panic(err)
	}
	return names[n.Int64()]
}

func newSurname() string {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(surnames))))
	if err != nil {
		panic(err)
	}
	return names[n.Int64()]
}

var names = []string{
	"Jane",
	"Jerrica",
	"Dylan",
	"Vashti",
	"Beaulah",
	"Geneva",
	"Winford",
	"Darrell",
	"Jeanette",
	"Vivian",
	"Jackie",
	"Moses",
	"Keely",
	"Ruthie",
	"Hoa",
	"Rheba",
	"Ja",
	"Lacresha",
	"Wilmer",
	"Corazon",
	"Marianne",
	"Shantelle",
	"Sherly",
	"Del",
	"Jason",
	"Solomon",
	"Raeann",
	"Roberta",
	"Nathalie",
	"Carolann",
}

var surnames = []string{
	"Helmuth",
	"Vanasse",
	"Bray",
	"Leclaire",
	"Soukup",
	"Nugent",
	"Lucio",
	"Shirkey",
	"Ostler",
	"Bonaparte",
	"Rinke",
	"Patillo",
	"Patten",
	"Greenhill",
	"Parrales",
	"Leppert",
	"Pando",
	"Spink",
	"Vire",
	"Orr",
	"Humphery",
	"Brew",
	"Jantz",
	"Bordelon",
	"Goodell",
	"Festa",
	"Morison",
	"Gartner",
	"Santi",
	"Dillahunt",
}

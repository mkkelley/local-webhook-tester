package proxy

import (
	"math/rand"
	"strconv"
)

func generateRandomUrlPrefix() string {
	return strconv.Itoa(int(rand.Int31()))
}

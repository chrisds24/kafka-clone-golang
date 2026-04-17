package main // In Go, package name must be main for executables

import (
	"github.com/chrisds24/kafka-clone-golang/internal/broker"
)

func main() {
	// Later on, I can load config here which I could pass to Start
	broker.Start()
}

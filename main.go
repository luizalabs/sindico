package main

import (
	log "github.com/inconshreveable/log15"
	"github.com/luizalabs/sindico/manager"
)

func main() {
	h := log.CallerFileHandler(log.StdoutHandler)
	log.Root().SetHandler(h)
	manager.Run()
}

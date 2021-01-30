package leaf

import (
	"github.com/keekekx/leaf/cluster"
	"github.com/keekekx/leaf/conf"
	"github.com/keekekx/leaf/console"
	"github.com/keekekx/leaf/log"
	"github.com/keekekx/leaf/module"
	"os"
	"os/signal"
)

func Run(mods ...module.Module) {
	// logger
	if conf.LogLevel != "" {
		logger, err := log.New(conf.LogLevel, conf.LogPath, conf.LogFlag)
		if err != nil {
			panic(err)
		}
		log.Export(logger)
		defer logger.Close()
	}

	log.Release("Leaf %v starting up", version)

	// module
	for i := 0; i < len(mods); i++ {
		module.Register(mods[i])
	}
	module.Init()

	// cluster
	cluster.Init()

	// console
	console.Init()

	// close
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	sig := <-c
	log.Release("Leaf closing down (signal: %v)", sig)
	console.Destroy()
	cluster.Destroy()
	module.Destroy()
}

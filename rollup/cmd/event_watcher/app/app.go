package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/database"
	"scroll-tech/common/observability"
	"scroll-tech/common/utils"
	"scroll-tech/common/version"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/controller/watcher"
)

var app *cli.App

func init() {
	// Set up event-watcher app info.
	app = cli.NewApp()
	app.Action = action
	app.Name = "event-watcher"
	app.Usage = "The Scroll Event Watcher"
	app.Version = version.Version
	app.Flags = append(app.Flags, utils.CommonFlags...)
	app.Commands = []*cli.Command{}
	app.Before = func(ctx *cli.Context) error {
		return utils.LogSetup(ctx)
	}
	// Register `event-watcher-test` app for integration-test.
	utils.RegisterSimulation(app, utils.EventWatcherApp)
}

func action(ctx *cli.Context) error {
	// Load config file.
	cfgFile := ctx.String(utils.ConfigFileFlag.Name)
	cfg, err := config.NewConfig(cfgFile)
	if err != nil {
		log.Crit("failed to load config file", "config file", cfgFile, "error", err)
	}

	subCtx, cancel := context.WithCancel(ctx.Context)
	// Init db connection
	db, err := database.InitDB(cfg.DBConfig)
	if err != nil {
		log.Crit("failed to init db connection", "err", err)
	}
	defer func() {
		cancel()
		if err = database.CloseDB(db); err != nil {
			log.Crit("failed to close db connection", "error", err)
		}
	}()

	registry := prometheus.DefaultRegisterer
	observability.Server(ctx, db)
	l1client, err := ethclient.Dial(cfg.L1Config.Endpoint)
	if err != nil {
		log.Crit("failed to connect l1 geth", "config file", cfgFile, "error", err)
	}

	l2client, err := ethclient.Dial(cfg.L2Config.Endpoint)
	if err != nil {
		log.Crit("failed to connect l2 geth", "config file", cfgFile, "error", err)
	}

	l1watcher := watcher.NewL1WatcherClient(ctx.Context, l1client, cfg.L1Config.StartHeight, cfg.L1Config.Confirmations,
		cfg.L1Config.L1MessageQueueAddress, cfg.L1Config.ScrollChainContractAddress, db, registry)

	l2watcher := watcher.NewL2WatcherClient(ctx.Context, l2client, cfg.L2Config.Confirmations, cfg.L2Config.L2MessengerAddress,
		cfg.L2Config.L2MessageQueueAddress, cfg.L2Config.WithdrawTrieRootSlot, db, registry)

	go utils.Loop(subCtx, 10*time.Second, func() {
		if loopErr := l1watcher.FetchContractEvent(); loopErr != nil {
			log.Error("Failed to fetch bridge contract", "err", loopErr)
		}
	})

	// Start l2 watcher process
	go utils.Loop(subCtx, 2*time.Second, l2watcher.FetchContractEvent)
	// Finish start all l2 functions

	log.Info("Start event-watcher successfully")

	// Catch CTRL-C to ensure a graceful shutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt

	return nil
}

// Run event watcher cmd instance.
func Run() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

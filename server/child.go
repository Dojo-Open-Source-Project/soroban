package server

import (
	"context"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	soroban "code.samourai.io/wallet/samourai-soroban"
	"code.samourai.io/wallet/samourai-soroban/ipc"

	log "github.com/sirupsen/logrus"
)

func startChildSoroban(ctx context.Context, options soroban.Options, childID int) {
	executablePath, err := os.Executable()
	if err != nil {
		log.WithError(err).
			Fatal("Failed to get current executable path")
	}

	if !strings.HasPrefix(executablePath, "/") {
		executablePath = path.Join("./", executablePath)
	}
	if !fileExists(executablePath) {
		log.Fatal("Soroban executable not found")
	}

	dhtServerMode := ""
	if options.P2P.DHTServerMode {
		dhtServerMode = "--p2pDHTServerMode"
	}

	go ipc.StartProcessDaemon(ctx, fmt.Sprintf("soroban-child-%d", childID),
		executablePath,
		// "--config", optionsc.Soroban.Config,
		"--ipcChildID", strconv.Itoa(childID),
		"--ipcNatsHost", options.IPC.NatsHost,
		"--ipcNatsPort", strconv.Itoa(options.IPC.NatsPort),
		"--p2pSeed", options.P2P.Seed,
		"--p2pBootstrap", options.P2P.Bootstrap,
		"--p2pRoom", options.P2P.Room,
		"--p2pListenPort", strconv.Itoa(options.P2P.ListenPort+childID),
		"--p2pLowWater", strconv.Itoa(options.P2P.LowWater),
		"--p2pHighWater", strconv.Itoa(options.P2P.HighWater),
		"--p2pPeerstoreFile", options.P2P.PeerstoreFile,
		"--gossipD", strconv.Itoa(options.Gossip.D),
		"--gossipDlo", strconv.Itoa(options.Gossip.Dlo),
		"--gossipDhi", strconv.Itoa(options.Gossip.Dhi),
		"--gossipDout", strconv.Itoa(options.Gossip.Dout),
		"--gossipDscore", strconv.Itoa(options.Gossip.Dscore),
		"--gossipDlazy", strconv.Itoa(options.Gossip.Dlazy),
		"--gossipPrunePeers", strconv.Itoa(options.Gossip.PrunePeers),
		"--gossipLimit", strconv.Itoa(options.Gossip.Limit),
		"--log", log.GetLevel().String(),
		dhtServerMode, // Must be the last flag
	)

}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

/*
* Copyright (C) 2020 The poly network Authors
* This file is part of The poly network library.
*
* The poly network is free software: you can redistribute it and/or modify
* it under the terms of the GNU Lesser General Public License as published by
* the Free Software Foundation, either version 3 of the License, or
* (at your option) any later version.
*
* The poly network is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
* GNU Lesser General Public License for more details.
* You should have received a copy of the GNU Lesser General Public License
* along with The poly network . If not, see <http://www.gnu.org/licenses/>.
 */
package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	//"github.com/elements-studio/poly-starcoin-relayer/cmd"
	"github.com/elements-studio/poly-starcoin-relayer/config"
	"github.com/elements-studio/poly-starcoin-relayer/db"
	"github.com/elements-studio/poly-starcoin-relayer/log"
	"github.com/elements-studio/poly-starcoin-relayer/manager"
	polySdk "github.com/polynetwork/poly-go-sdk"
	stcclient "github.com/starcoinorg/starcoin-go/client"
	"github.com/urfave/cli"
)

var ConfigPath string
var LogDir string
var StartHeight uint64
var PolyStartHeight uint64
var StartForceHeight uint64

func setupApp() *cli.App {
	app := cli.NewApp()
	app.Usage = "Starcoin relayer Service"
	app.Action = startServer
	app.Version = config.Version
	app.Copyright = "Copyright in 2019 The Ontology Authors"
	// app.Flags = []cli.Flag{
	// 	cmd.LogLevelFlag,
	// 	cmd.ConfigPathFlag,
	// 	cmd.EthStartFlag,
	// 	cmd.EthStartForceFlag,
	// 	cmd.PolyStartFlag,
	// 	cmd.LogDir,
	// }
	//todo...
	app.Commands = []cli.Command{}
	app.Before = func(context *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		return nil
	}
	return app
}

func startServer(ctx *cli.Context) {
	// get all cmd flag
	//todo ...
	// logLevel := ctx.GlobalInt(cmd.GetFlagName(cmd.LogLevelFlag))

	// ld := ctx.GlobalString(cmd.GetFlagName(cmd.LogDir))
	// log.InitLog(logLevel, ld, log.Stdout)

	// ConfigPath = ctx.GlobalString(cmd.GetFlagName(cmd.ConfigPathFlag))
	// ethstart := ctx.GlobalUint64(cmd.GetFlagName(cmd.EthStartFlag))
	// if ethstart > 0 {
	// 	StartHeight = ethstart
	// }

	// StartForceHeight = 0
	// ethstartforce := ctx.GlobalUint64(cmd.GetFlagName(cmd.EthStartForceFlag))
	// if ethstartforce > 0 {
	// 	StartForceHeight = ethstartforce
	// }
	// polyStart := ctx.GlobalUint64(cmd.GetFlagName(cmd.PolyStartFlag))
	// if polyStart > 0 {
	// 	PolyStartHeight = polyStart
	// }

	// read config
	servConfig := config.NewServiceConfig(ConfigPath)
	if servConfig == nil {
		log.Errorf("startServer - create config failed!")
		return
	}

	// create poly sdk
	polySdk := polySdk.NewPolySdk()
	err := setUpPoly(polySdk, servConfig.PolyConfig.RestURL)
	if err != nil {
		log.Errorf("startServer - failed to setup poly sdk: %v", err)
		return
	}

	// create ethereum sdk
	// ethereumsdk, err := ethclient.Dial(servConfig.StarcoinConfig.RestURL)
	// if err != nil {
	// 	log.Errorf("startServer - cannot dial sync node, err: %s", err)
	// 	return
	// }
	//todo...
	stcclient := &stcclient.StarcoinClient{}

	var boltDB *db.BoltDB
	if servConfig.BoltDbPath == "" {
		boltDB, err = db.NewBoltDB("boltdb")
	} else {
		boltDB, err = db.NewBoltDB(servConfig.BoltDbPath)
	}
	if err != nil {
		log.Fatalf("db.NewWaitingDB error:%s", err)
		return
	}

	initPolyServer(servConfig, polySdk, stcclient, boltDB)
	initStarcoinServer(servConfig, polySdk, stcclient, boltDB)
	waitToExit()
}

func setUpPoly(poly *polySdk.PolySdk, RpcAddr string) error {
	poly.NewRpcClient().SetAddress(RpcAddr)
	hdr, err := poly.GetHeaderByHeight(0)
	if err != nil {
		return err
	}
	poly.SetChainId(hdr.ChainID)
	return nil
}

func waitToExit() {
	exit := make(chan bool, 0)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for sig := range sc {
			log.Infof("waitToExit - ETH relayer received exit signal:%v.", sig.String())
			close(exit)
			break
		}
	}()
	<-exit
}

func initStarcoinServer(servConfig *config.ServiceConfig, polysdk *polySdk.PolySdk, stcclient *stcclient.StarcoinClient, db db.DB) {
	mgr, err := manager.NewStarcoinManager(servConfig, StartHeight, StartForceHeight, polysdk, stcclient, db)
	if err != nil {
		log.Error("initStarcoinServer - starcoin service start err: %s", err.Error())
		return
	}
	go mgr.MonitorChain()
	go mgr.MonitorDeposit()
	go mgr.CheckDeposit()
}

func initPolyServer(servConfig *config.ServiceConfig, polysdk *polySdk.PolySdk, stcclient *stcclient.StarcoinClient, db db.DB) {
	mgr, err := manager.NewPolyManager(servConfig, uint32(PolyStartHeight), polysdk, stcclient, db)
	if err != nil {
		log.Error("initPolyServer - PolyServer service start failed: %v", err)
		return
	}
	go mgr.MonitorChain()
}

func main() {
	log.Infof("main - Starcoin relayer starting...")
	if err := setupApp().Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

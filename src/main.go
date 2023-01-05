package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"mcmcx.com/mserver/src/database"
	"mcmcx.com/mserver/src/gameserver"
	"mcmcx.com/mserver/src/logout"
	"mcmcx.com/mserver/src/server"
)

func main() {

	logout.LogInit()
	logout.Log("Logout init ...")

	logout.Log("Database (Redis) init ...")
	if !database.RedisInitialize("data/ServerInfo.json") {
		logout.LogError("[Database] Error: ", "initialize redis error.")
		return
	}

	gameserver.GTempUserManager.Initialize(gameserver.USER_TEMP, 100)
	gameserver.GUserManager.Initialize(gameserver.USER_NORMAL, 5000)

	if !server.InitHTTPServer("data/ServerInfo.json", gin.DebugMode) {
		logout.LogError("[HTTP] Error: ", "init http server error.")
		return
	}

	if !server.StartHTTPServer() {
		logout.LogError("[HTTP] Error: ", "starting http server error.")
		return
	}

	if !server.StartHTTPSServer() {
		logout.LogError("[HTTP] Error: ", "starting https server error.")
		return
	}

	logout.Log("Gameserver init ...")
	if !gameserver.LoadGameServer("data/GameServerInfo.json") {
		logout.LogError("[GameServer] Error: ", "loading game server error.")
		return
	}

	sigs := make(chan os.Signal, 1)
	//signal.Ignore(os.Interrupt)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	println("Signal -> %+v", sig)

	//
	println("Exiting ...")

	gameserver.FreeGameServerAll()

	gameserver.GTempUserManager.Release()
	gameserver.GUserManager.Release()

	database.RedisRelease()

	//
	return
}

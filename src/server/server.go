package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"mcmcx.com/mserver/src/logout"
	"mcmcx.com/mserver/src/util"
)

//
type ServerInfo struct {
	HttpPort  int    `json:"http"`
	HttpsPort int    `json:"https"`
	HttpsKey  string `json:"https_key"`
	HttpsCrt  string `json:"https_crt"`
}

//
const LOG_HTTP = "HTTP"

//
var router_instance *gin.Engine = nil
var server_info ServerInfo

//
func load_serverinfo(filename string) bool {

	//
	if !util.LoadJsonFromFile[ServerInfo](filename, &server_info) {
		logout.LogError("[Load] Read server info fail")
		return false
	}
	return true
}

func register_handlers(router *gin.Engine) bool {
	router.GET("/ping", R_handler_ping)
	router.GET("/hello", R_handler_hello)
	router.Any("/auth", R_handler_auth)
	router.GET("/user", R_handler_user)
	return true
}

//
// mode : gin.DebugMode, gin.ReleaseMode

func InitHTTPServer(filename string, mode string) bool {
	gin.SetMode(mode)
	router_instance = gin.Default()

	logout.LogAdd(logout.LogLevel_Info, LOG_HTTP, true, false)
	if !load_serverinfo(filename) {
		return false
	}

	// custom logs
	router_instance.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// your custom format
		var text = fmt.Sprintf("[%s] (%s) | %s \"%s\" (%s, %d, %s)",
			param.TimeStamp.Format(time.RFC3339),
			param.ClientIP,
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			//param.Request.UserAgent(),
			//param.ErrorMessage,
		)
		logout.LogWithName(LOG_HTTP, text)
		if len(param.ErrorMessage) > 0 {
			logout.LogWithName(LOG_HTTP, "Error Message : ", param.ErrorMessage)
		}
		return text + "\n"
	}))

	register_handlers(router_instance)

	return true
}

func start_http_server(address string) bool {
	//IPv6
	//router.Run(":8080") // listen and serve on 0.0.0.0:8080
	var err = http.ListenAndServe(address, router_instance)
	if err != nil {
		logout.LogError("[HTTP] Error: ", err.Error(), "")
		return false
	}
	return true
}

func start_https_server(address string, key string, crt string) bool {
	var err = http.ListenAndServeTLS(address,
		crt, key,
		router_instance)
	if err != nil {
		logout.LogError("[HTTPS] Error: ", err.Error(), "")
		return false
	}
	return true
}

func StartHTTPServer() bool {
	var port = -1
	if server_info.HttpPort > 0 {
		port = server_info.HttpPort
	}
	if port < 0 {
		logout.LogWarn("[HTTP] Server closed")
		return false
	}

	go start_http_server(fmt.Sprintf(":%d", port))
	defer logout.Log("[HTTP] Server starting on ", port)
	return true
}

func StartHTTPSServer() bool {
	// need key, certs, port all true
	var port = -1
	if server_info.HttpsPort > 0 && len(server_info.HttpsCrt) > 0 && len(server_info.HttpsKey) > 0 {
		port = server_info.HttpsPort
	}
	if port < 0 {
		logout.LogWarn("[HTTPS] Server closed")
		return false
	}

	go start_https_server(fmt.Sprintf(":%d", port),
		server_info.HttpsKey, server_info.HttpsCrt)
	defer logout.Log("[HTTPS] Server starting on ", port)
	return true
}

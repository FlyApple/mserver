// Package utils 提供zinx相关工具类函数
// 包括:
//		全局配置
//		配置文件加载
//
// 当前文件描述:
// @Title  global.go
// @Description  相关配置文件定义及加载方式
// @Author  Aceld - Thu Mar 11 10:32:29 CST 2019
package zutils

const (
	ZSERVER_NAME    = "ZinxServer"
	ZSERVER_VERSION = "1.0.1"
)

const (
	ZSERVER_TCP  = "tcp"
	ZSERVER_TCP4 = "tcp4"
	ZSERVER_TCP6 = "tcp6"
)

const (
	ZSERVER_CONNECTIONS_NUM = 100
	ZSERVER_PACKET_SIZE     = 4096
)

//
var ZServer_LogDir = "logs"
var ZServer_ConfFile = "conf/zinx.json"

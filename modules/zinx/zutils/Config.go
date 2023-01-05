package zutils

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"

	"mcmcx.com/mserver/modules/zinx/zlog"
)

//
type TConfig struct {

	//Server
	Type    string `json:"type"`    //tcp版本:tcp,tcp4,tcp6
	Address string `json:"address"` //当前服务器主机监听的IP
	Port    int    `json:"port"`    //当前服务器监听的端口

	//服务器可选配置
	Name    string `json:"name"`    //当前服务器的名称
	Version string `json:"version"` //版本
	//
	ConnectionsMaxNum int32  `json:"connections_maxnum"` //最大连接数量
	PacketSize        uint32 `json:"packet_size"`        //当前框架数据包的最大尺寸
	WorkerPoolSize    int32  //业务工作Worker池的数量
	WorkerTaskMaxLen  int32  //业务工作Worker对应负责的任务队列最大任务存储数量
	MsgChanMaxLen     int32  //SendBuffMsg发送消息的缓冲最大长度
}

type TGlobal struct {
	/*
		logger
	*/
	LogDir   string `json:"log_dir"`   //日志所在文件夹 默认"./logs"
	LogFile  string `json:"log_file"`  //日志文件名称   默认""  --如果没有设置日志文件，打印信息将打印至stderr
	LogDebug bool   `json:"log_debug"` //是否关闭Debug日志级别调试信息 默认false  -- 默认打开debug信息

	//
	List []TConfig `json:"list"`
}

var Global TGlobal

//PathExists 判断一个文件是否存在
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

//
func InitConfig(config *TConfig) {
	if config == nil {
		return
	}

	if len(config.Name) == 0 {
		config.Name = ZSERVER_NAME
	}
	if len(config.Version) == 0 {
		config.Version = ZSERVER_VERSION
	}

	if len(config.Type) == 0 {
		config.Type = ZSERVER_TCP4
	}
	if len(config.Address) == 0 {
		config.Type = ZSERVER_TCP4
		config.Address = "0.0.0.0"
	}
	if config.Port == 0 {
		config.Port = 9000
	}

	if config.PacketSize == 0 {
		config.PacketSize = ZSERVER_PACKET_SIZE
	}
	if config.ConnectionsMaxNum == 0 {
		config.ConnectionsMaxNum = ZSERVER_CONNECTIONS_NUM
	}
	if config.WorkerPoolSize == 0 {
		config.WorkerPoolSize = 10
	}
	if config.WorkerTaskMaxLen == 0 {
		config.WorkerTaskMaxLen = 1024
	}
	if config.MsgChanMaxLen == 0 {
		config.MsgChanMaxLen = 1024
	}
}

//LoadConfig 读取用户的配置文件
func LoadConfigFromFile(filename string, g *TGlobal) error {

	if confFileExists, _ := PathExists(filename); confFileExists != true {
		return errors.New("File " + filename + " is not exist")
	}

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	//将json数据解析到struct中
	err = json.Unmarshal(data, g)
	if err != nil {
		return err
	}

	//
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	if len(g.LogDir) == 0 {
		g.LogDir = cwd + "/" + ZServer_LogDir
	}

	if len(g.LogFile) == 0 {
		g.LogFile = ""
	}

	//Logger 设置
	if g.LogFile != "" {
		zlog.SetLogFile(g.LogDir, g.LogFile)
	}
	if g.LogDebug == false {
		zlog.CloseDebug()
	}
	return nil
}

func init() {
	//
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	filename := cwd + "/" + ZServer_ConfFile

	Global = TGlobal{
		LogDebug: false,
		LogDir:   cwd + "/" + ZServer_LogDir,
		LogFile:  "",
	}

	err = LoadConfigFromFile(filename, &Global)
	if err != nil {
		//println("[ZSERVER] Error : " + err.Error())
		return
	}

	var vlist = Global.List
	if vlist != nil {
		for _, v := range vlist {
			InitConfig(&v)
		}
	}
}

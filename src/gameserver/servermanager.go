package gameserver

import (
	"fmt"
	"strconv"
	"sync"

	"mcmcx.com/mserver/modules/zinx/znet"
	"mcmcx.com/mserver/modules/zinx/zutils"
	"mcmcx.com/mserver/src/logout"
	"mcmcx.com/mserver/src/util"
)

//
const LOG_GAMESERVER = "GAMESERVER"

const (
	STATUS_FREE    = -1
	STATUS_NULL    = 0
	STATUS_INIT    = 1
	STATUS_WORKING = 2
)

const (
	PRIORITY_NORMAL = 0
	PRIORITY_LEVEL1 = 1
	PRIORITY_LEVEL2 = 2
	PRIORITY_LEVEL3 = 3
)

//
type TServerInfo struct {
	ID      int
	Name    string `json:"name"`
	Title   string `json:"title"`
	Version string `json:"version"`

	Type string `json:"type"`
	// Bind Address
	Address string `json:"address"`
	Port    int    `json:"port"`
	// Public Address
	GateAddress string `json:"gate_address"`
	GatePort    int    `json:"gate_port"`

	PacketSize        int `json:"packet_size"`
	ConnectionsMaxNum int `json:"connections_maxnum"`

	//
	PriorityLevel int `json:"priority_level"`
}

type TServerInfoList struct {
	List []TServerInfo `json:"list"`
}
type TPServerInfo *TServerInfo

//
type ServerManager struct {
	servers_info map[int]TPServerInfo

	//
	servers_lock sync.Mutex
	servers_list map[int]*t_server
}

var GServerManager ServerManager

//
func (self *ServerManager) initialize() bool {
	//
	self.servers_info = make(map[int]TPServerInfo)
	self.servers_list = make(map[int]*t_server)
	return true
}

//
func (self *ServerManager) load_serverinfo(filename string) bool {
	self.servers_info = make(map[int]TPServerInfo)

	//
	var server_info_list TServerInfoList
	if !util.LoadJsonFromFile[TServerInfoList](filename, &server_info_list) {
		logout.LogError("[Load] Read server info fail")
		return false
	}

	vlist := server_info_list.List
	for n, _ := range vlist {
		vlist[n].ID = int(util.GenerateIDX(0))
		if len(vlist[n].GateAddress) == 0 {
			vlist[n].GateAddress = "127.0.0.1"
		}
		if vlist[n].GatePort == 0 {
			vlist[n].GatePort = vlist[n].Port
		}
		self.servers_info[vlist[n].ID] = &vlist[n]
	}

	//
	return true
}

//
func (self *ServerManager) add_server(server *t_server) bool {
	if server == nil || server.ID <= 0 {
		return false
	}
	return self.add_server_byid(server.ID, server)
}

func (self *ServerManager) add_server_byid(id int, server *t_server) bool {
	self.servers_lock.Lock()
	_, ok := self.servers_list[id]
	self.servers_lock.Unlock()
	if ok {
		return false
	}

	self.servers_lock.Lock()
	self.servers_list[id] = server
	self.servers_lock.Unlock()
	return true
}

func (self *ServerManager) del_server_by_id(id int) bool {
	if id <= 0 {
		return false
	}

	self.servers_lock.Lock()
	v, ok := self.servers_list[id]
	self.servers_lock.Unlock()
	if !ok {
		return false
	}

	if v.working() {
		self.free_server(v)
	}

	self.servers_lock.Lock()
	delete(self.servers_list, id)
	self.servers_lock.Unlock()

	v.status = STATUS_FREE
	return true
}

func (self *ServerManager) del_server(server *t_server) bool {
	if server == nil || server.ID <= 0 {
		return false
	}
	return self.del_server_by_id(server.ID)
}

func (self *ServerManager) free_server(server *t_server) bool {
	server.status = STATUS_NULL

	server.server.Stop()
	server.release()
	return true
}

func (self *ServerManager) del_server_all() {
	self.servers_lock.Lock()
	for n, _ := range self.servers_list {
		defer self.del_server_by_id(n)
	}
	self.servers_lock.Unlock()
}

func (self *ServerManager) init_server(server *t_server) bool {
	if !server.initialize() {
		return false
	}
	server.server.Start()

	server.status = STATUS_WORKING
	return true
}

func (self *ServerManager) GetServerInfo(id int) TPServerInfo {
	si, ok := self.servers_info[id]
	if !ok {
		return nil
	}
	return si
}

func (self *ServerManager) GetServer(id int) *t_server {
	self.servers_lock.Lock()
	s, ok := self.servers_list[id]
	self.servers_lock.Unlock()
	if !ok {
		return nil
	}
	return s
}

func (self *ServerManager) GetIdleServer() *t_server {
	var s *t_server = nil
	if !self.servers_lock.TryLock() {
		return nil
	}

	var s1, s2, s3 *t_server = nil, nil, nil
	for _, v := range self.servers_list {
		// priority level
		if s1 == nil || (s1 != nil && s1.priority_level < v.priority_level) {
			s1 = v
		}

		// percentage < 80
		percentage := v.SessionsUsedRatio() * 100
		if s2 == nil || (percentage < 80 && v.SessionsUsedRatio() > s2.SessionsUsedRatio()) {
			s2 = v
		}
		//
		if s3 == nil || (percentage >= 80 && v.SessionsUsedRatio() < s3.SessionsUsedRatio()) {
			s3 = v
		}
	}

	s = s1
	if s2 != nil && s2.priority_level >= s.priority_level {
		s = s2
	}
	if s3 != nil && s3.priority_level >= s.priority_level {
		s = s3
	}
	self.servers_lock.Unlock()
	return s
}

func create_gameserver(info TPServerInfo) *t_server {
	var server = &t_server{
		ID:        -1,
		Token:     "",
		timestamp: util.GetTimeStamp64(),

		//
		server: nil,

		//
		priority_level: info.PriorityLevel, //GAMESERVER_PRIORITY_NORMAL
		status:         STATUS_NULL,
	}

	var code = util.GenerateAuthCode(4)

	server.ID = info.ID
	server.Token = util.MD5(strconv.FormatInt(int64(server.ID), 10) + "_" + code)

	if !GServerManager.add_server(server) {
		return nil
	}

	var name = fmt.Sprintf("GameServer_%d", info.ID)
	if len(info.Name) > 0 {
		name = info.Name
	}

	server.server = znet.NewServer(&zutils.TConfig{
		Name:    name,
		Type:    info.Type,
		Address: info.Address,
		Port:    info.Port,

		PacketSize:        uint32(info.PacketSize),
		ConnectionsMaxNum: int32(info.ConnectionsMaxNum),
	})

	//
	server.status = STATUS_INIT

	if !GServerManager.init_server(server) {
		GServerManager.del_server(server)
		return nil
	}

	return server
}

func FreeGameServerAll() {
	GServerManager.del_server_all()
}

func LoadGameServer(filename string) bool {

	zutils.Global.LogDebug = false
	logout.LogAdd(logout.LogLevel_Info, LOG_GAMESERVER, true, true)

	if !GServerManager.initialize() || !GServerManager.load_serverinfo(filename) {
		return false
	}

	vlist := GServerManager.servers_info
	for _, v := range vlist {
		server := create_gameserver(v)
		if server == nil {
			logout.LogWithName(LOG_GAMESERVER, "(Error) Create GameServer failed")
			return false
		}
		logout.LogWithName(LOG_GAMESERVER, "(Info) Create GameServer (ID:", server.ID, "), Max Sessions:",
			(*server).SessionsMaxNum(), " [OK]")
	}

	return true
}

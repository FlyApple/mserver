package gameserver

import (
	"strings"
	"time"

	"mcmcx.com/mserver/modules/zinx/ziface"
	"mcmcx.com/mserver/modules/zinx/znet"
	"mcmcx.com/mserver/modules/zinx/zpack"
	"mcmcx.com/mserver/src/database"
	"mcmcx.com/mserver/src/logout"
	"mcmcx.com/mserver/src/util"
)

//
type HandlerBase struct {
	ServerID    int
	ServerToken string

	//
	LogName string

	//
	SessionID     int
	Session       ziface.IConnection
	SessionUserID int
	SessionUser   i_user
}

func (self *HandlerBase) InitHandle(request ziface.IRequest) bool {
	self.Session = request.GetConnection()
	if self.Session == nil {
		return false
	}
	self.SessionID = int(self.Session.GetConnectionID())

	server_id, _ := self.Session.GetProperty("server_id")
	server_token, _ := self.Session.GetProperty("server_token")
	self.ServerID = server_id.(int)
	self.ServerToken = server_token.(string)

	user_id, _ := self.Session.GetProperty("user_id")
	user_type, _ := self.Session.GetProperty("user_type")

	self.SessionUserID = user_id.(int)
	self.SessionUser = nil
	if user_type.(string) == USER_NORMAL {
		self.LogName = GUserManager.LogName
		self.SessionUser = GUserManager.GetUser(user_id.(int))
	} else {
		self.LogName = GTempUserManager.LogName
		self.SessionUser = GTempUserManager.GetUser(user_id.(int))
	}
	if len(self.LogName) == 0 {
		self.LogName = LOG_USER_TEMP
	}

	if self.SessionUser == nil {
		logout.LogWithName(self.LogName, "[ERROR] (User) Session user NULL",
			", ID:", user_id, " (", user_type, ")", ", SID:", self.SessionID)
		return false
	}

	return true
}

//
type HandlerHello struct {
	znet.BaseRouter
	super HandlerBase
}

//
type HandlerPing struct {
	znet.BaseRouter
	super HandlerBase
}

//
type HandlerAuth struct {
	znet.BaseRouter
	super HandlerBase
}

//
type HandlerUser struct {
	znet.BaseRouter
	super HandlerBase
}

// Handler 00: Hello
func (self *HandlerHello) Handle(request ziface.IRequest) {
	if !self.super.InitHandle(request) {
		return
	}

	var buffer zpack.MessageBuffer
	buffer.WriteUInt32(util.GetTimeStamp())
	buffer.WriteUInt64(util.GetTimeStamp64())
	buffer.WriteStringL(util.DateFormat(time.Now(), 3))

	err := self.super.Session.SendBufferMsg(0, buffer.Data())
	if err != nil {
		return
	}
}

// Handler 01: Ping
func (self *HandlerPing) Handle(request ziface.IRequest) {
	if !self.super.InitHandle(request) {
		return
	}

	var buffer zpack.MessageBuffer
	buffer.WriteUInt32(util.GetTimeStamp())
	buffer.WriteUInt64(util.GetTimeStamp64())

	err := self.super.Session.SendBufferMsg(1, buffer.Data())
	if err != nil {
		return
	}
}

//
func (self *HandlerAuth) ServerAuth(id int, token string, info *TPServerInfo) int {
	if id != self.super.ServerID || token != self.super.ServerToken {
		return -1
	}

	if info != nil {
		*info = GServerManager.GetServerInfo(id)
		if *info != nil {
			return 1
		}
	}
	return 0
}

//
func (self *HandlerAuth) DBUserAuth(idx string, token string, key **database.DBUserKey) int {
	// Get user data
	var db_user_data *database.DBUserData = database.DB_get_user_data(idx)
	if db_user_data == nil || db_user_data.Status < 0 {
		return -1
	}

	if db_user_data.IDX != idx || db_user_data.ServerUserToken != token {
		return -1
	}

	if key == nil {
		return -1
	}

	*key = &database.DBUserKey{
		IDX:      db_user_data.IDX,
		PKey:     db_user_data.PKey,
		PKeyHash: db_user_data.PKeyHash,
	}
	if *key != nil {
		return 1 // OK
	}

	return 0 // FAILED
}

//
func (self *HandlerAuth) UserLoad(user *TUser, idx string, token string,
	timestamp uint32, server_id int, server_token string, server_info TPServerInfo,
	user_addr string, shared_key string) int {
	//
	if user == nil {
		return -1
	}

	if !user.Load(self.super.Session.GetConnectionID(), user_addr) {
		return -1
	}

	user.IDX = idx
	user.Token = token

	user.ClientTimestamp32 = timestamp
	user.ServerTimestamp32 = util.GetTimeStamp()

	user.ServerID = server_id
	user.ServerName = server_info.Title
	user.ServerToken = server_token

	result := user.LoadCrypto(1, shared_key)
	if result {
		return 1 // OK
	}

	//
	return 0 // FAILED
}

// Handler 08: Login
// Handler 09: Auth
// Client Packet:
//   - User IDX (string)
//   - User Timestamp (uint client)
//   - Server ID (int)
//   - Server Token (MD5 16bytes)
//   - User Remote Address (string)
//   - User Authentication Token (MD5 string)
//   - User PublicKey (ECC bytes)
// Server Packet:
//   - Result (int)
//   - User Timestamp (uint server)
//   - User IDX (string, result >= 0)
//   - Server ID (int, result >= 1)
//   - Server Name (string, result >= 1)

func (self *HandlerAuth) Handle(request ziface.IRequest) {
	if !self.super.InitHandle(request) {
		return
	}

	recv_buffer := zpack.NewMessageBuffer(request.GetData())
	// User IDX
	idx := recv_buffer.ReadStringL()
	timestamp := recv_buffer.ReadUInt32()
	if len(idx) == 0 || timestamp == 0 {
		logout.LogWithName(self.super.LogName, "[AUTH] (User) Authentication failed, Result: idx error",
			", ID:", self.super.SessionUserID, ", SID:", self.super.SessionID)

		self.HandleResultFailed(request, -1)
		return
	}
	idx = strings.TrimSpace(idx)

	// Server Auth
	server_id := recv_buffer.ReadInt32()
	server_token := strings.TrimSpace(recv_buffer.ReadStringL())
	var server_info TPServerInfo = nil
	if self.ServerAuth(int(server_id), server_token, &server_info) <= 0 {
		logout.LogWithName(self.super.LogName, "[AUTH] (User) Authentication failed, Result: server error",
			", ID:", self.super.SessionUserID, ", SID:", self.super.SessionID)

		self.HandleResultFailed(request, -1)
		return
	}

	// User address
	user_addr := strings.TrimSpace(recv_buffer.ReadStringL())
	if len(user_addr) == 0 {
		v := strings.Split(self.super.Session.RemoteAddr().String(), ":")
		if len(v) > 0 {
			user_addr = v[0]
		}
	}
	// Not gate address, use remote address
	if self.super.SessionUser.RemoteAddress() != server_info.GateAddress &&
		self.super.SessionUser.RemoteAddress() != user_addr {
		user_addr = self.super.SessionUser.RemoteAddress()
	}

	// User Token
	user_token := strings.TrimSpace(recv_buffer.ReadStringL())

	var user_key *database.DBUserKey
	if self.DBUserAuth(idx, user_token, &user_key) <= 0 {
		logout.LogWithName(self.super.LogName, "[AUTH] (User) Authentication failed, Result: failed",
			", ID:", self.super.SessionUserID, ", SID:", self.super.SessionID, ", IDX:", idx)

		self.HandleResultFailedEx(request, 0, idx)
		return
	}
	var user_skey = util.ECCX509PrivateKeyDecoding(user_key.PKey)

	// User Key
	user_shared_key := ""
	user_pkey_data := recv_buffer.ReadBytesL()
	if user_pkey_data != nil {
		user_pkey := util.ECCPublicKeyParseData(user_pkey_data)
		if user_pkey != nil && user_skey != nil {
			user_shared_key = util.ECCGenSharedKeyEncoding(user_skey, user_pkey)
		}
	}

	// User load
	var user *TUser = &TUser{}
	result := GUserManager.AddUser(user)
	if result && self.UserLoad(user, idx, user_token, timestamp,
		int(server_id), server_token, server_info,
		user_addr, user_shared_key) > 0 {
		result = true
	} else {
		GUserManager.DelUserByID(user.ID())

		result = false
	}

	// SUCCESSED
	if result {
		self.super.Session.SetProperty("user_id", user.ID())
		self.super.Session.SetProperty("user_type", user.Type())
		GTempUserManager.DelUserByID(self.super.SessionUserID)

		logout.LogWithName(self.super.LogName, "[AUTH] (User) Authentication successed, Result: ok",
			", ID:", self.super.SessionUserID, ", SID:", self.super.SessionID,
			", IDX:", idx, ", NewID:", user.ID(), ", Address:", user.RemoteAddress())

		self.HandleResultSuccessed(request, 1, user)
		return
	}

	logout.LogWithName(self.super.LogName, "[AUTH] (User) Authentication failed, Result: failed",
		", ID:", self.super.SessionUserID, ", SID:", self.super.SessionID, ", IDX:", idx)

	//
	self.HandleResultFailedEx(request, 0, idx)
}

func (self *HandlerAuth) HandleResultFailed(request ziface.IRequest, result int32) {
	var buffer zpack.MessageBuffer
	buffer.WriteInt32(result)
	buffer.WriteUInt32(util.GetTimeStamp())

	err := self.super.Session.SendBufferMsg(0x09, buffer.Data())
	if err != nil {
		return
	}
}

func (self *HandlerAuth) HandleResultFailedEx(request ziface.IRequest, result int32, idx string) {
	var buffer zpack.MessageBuffer
	buffer.WriteInt32(result)
	buffer.WriteUInt32(util.GetTimeStamp())
	buffer.WriteStringL(idx)

	err := self.super.Session.SendBufferMsg(0x09, buffer.Data())
	if err != nil {
		return
	}
}

func (self *HandlerAuth) HandleResultSuccessed(request ziface.IRequest, result int32, user *TUser) {
	var buffer zpack.MessageBuffer
	buffer.WriteInt32(result)
	buffer.WriteUInt32(user.ServerTimestamp32)
	// Result >= 0
	buffer.WriteStringL(user.IDX)
	// Result >= 1
	buffer.WriteInt32(int32(user.ServerID))
	buffer.WriteStringL(user.ServerName)

	err := self.super.Session.SendBufferMsg(0x09, buffer.Data())
	if err != nil {
		return
	}
}

// Handler 10: User
func (self *HandlerUser) Handle(request ziface.IRequest) {
	if !self.super.InitHandle(request) {
		return
	}

	recv_buffer := zpack.NewMessageBuffer(request.GetData())
	// User IDX
	idx := recv_buffer.ReadStringL()
	idx = strings.TrimSpace(idx)
	if len(idx) == 0 || self.super.SessionUser == nil {
		return
	}
	user := self.super.SessionUser.(*TUser)
	if user.IDX != idx {
		return
	}

	self.HandleResultUser(request, user)
}

func (self *HandlerUser) HandleResultUser(request ziface.IRequest, user *TUser) {
	var buffer zpack.MessageBuffer
	buffer.WriteStringL(user.IDX)

	err := self.super.Session.SendBufferMsg(0x10, buffer.Data())
	if err != nil {
		return
	}
}

package gameserver

import (
	"strings"

	"mcmcx.com/mserver/modules/zinx/ziface"
	"mcmcx.com/mserver/src/logout"
)

//
type i_server interface {
	initialize() bool
	release()

	SessionsMaxNum() int32
	SessionsNum() int32
}

//
type t_server struct {
	i_server

	ID    int
	Token string

	timestamp uint64

	//
	server ziface.IServer

	//
	priority_level int
	status         int
}

func (self *t_server) SessionsMaxNum() int32 {
	return int32(self.server.GetConnectionManager().MaxLen())
}

func (self *t_server) SessionsNum() int32 {
	return int32(self.server.GetConnectionManager().Len())
}

func (self *t_server) SessionsUsedRatio() float32 {
	num := self.SessionsNum()
	if num == 0 {
		num = 1
	}
	if num >= self.SessionsMaxNum() {
		num = self.SessionsMaxNum()
	}

	percentage := float32(num) / float32(self.SessionsMaxNum())
	if percentage < 1.0 {
		percentage = 1.0
	}
	return percentage
}

func (self *t_server) initialize() bool {

	//
	self.server.SetDataPtr(self)
	self.server.SetOnConnectionStart(handler_session_accept)
	self.server.SetOnConnectionStop(handler_session_closed)

	//
	self.server.AddRouter(0x00, &HandlerHello{})
	self.server.AddRouter(0x01, &HandlerPing{})
	self.server.AddRouter(0x09, &HandlerAuth{})
	self.server.AddRouter(0x10, &HandlerUser{})

	//
	return true
}

func (self *t_server) release() {
	// nothing
}

func (self *t_server) working() bool {
	return self.status >= STATUS_WORKING
}

func (self *t_server) on_session_accept(session ziface.IConnection) {
	logout.LogWithName(LOG_GAMESERVER, "(Accept) Session accept ID: ", session.GetConnectionID(),
		", Address: ", session.RemoteAddr())

	//Add temp user
	user := &TTempUser{}
	if !GTempUserManager.AddUser(user) {
		session.Close()
		return
	}

	address := "127.0.0.1"
	v := strings.Split(session.RemoteAddr().String(), ":")
	if len(v) > 0 {
		address = v[0]
	}

	if !user.Load(session.GetConnectionID(), address) {
		GTempUserManager.DelUser(user)
		session.Close()
		return
	}

	session.SetProperty("user_id", user.ID())
	session.SetProperty("user_type", user.Type())
}

func (self *t_server) on_session_closed(session ziface.IConnection) {

	//
	user_id, _ := session.GetProperty("user_id")
	user_type, _ := session.GetProperty("user_type")
	if user_type.(string) == USER_NORMAL {
		GUserManager.DelUserByID(user_id.(int))
	} else {
		GTempUserManager.DelUserByID(user_id.(int))
	}

	//
	logout.LogWithName(LOG_GAMESERVER, "(Close) Session closed ID: ", session.GetConnectionID(),
		", Address: ", session.RemoteAddr())
}

func handler_session_accept(data any, session ziface.IConnection) {
	server, ok := data.(*t_server)
	if !ok || server == nil {
		logout.LogWithName(LOG_GAMESERVER, "(Error) Session accept failed, ID: ", session.GetConnectionID())
		return
	}

	session.SetProperty("server_id", server.ID)
	session.SetProperty("server_token", server.Token)
	server.on_session_accept(session)
}

func handler_session_closed(data any, session ziface.IConnection) {
	server, ok := data.(*t_server)
	if !ok || server == nil {
		logout.LogWithName(LOG_GAMESERVER, "(Error) Session closed failed, ID: ", session.GetConnectionID())
		return
	}
	server.on_session_closed(session)
}

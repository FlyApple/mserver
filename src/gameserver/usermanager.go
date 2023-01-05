package gameserver

import (
	"sync"

	"mcmcx.com/mserver/src/logout"
)

const (
	USER_ID_NULL = 1000
	USER_ID_MAX  = 1000000000
)

const (
	LOG_USER_TEMP = USER_TEMP
	LOG_USER      = USER_NORMAL
)

//
type UserManager struct {
	Type    string
	MaxNum  int
	LogName string

	//
	idn int

	//
	current_num int
	lock        sync.Mutex
	list        map[int]i_user
}

var GTempUserManager UserManager
var GUserManager UserManager

//
func (self *UserManager) UserMaxNum() int { return self.MaxNum }
func (self *UserManager) UserNum() int    { return self.current_num }

//
func (self *UserManager) Initialize(user_type string, maxnum int) bool {
	if user_type == USER_NORMAL {
	} else {
		user_type = USER_TEMP
	}

	self.Type = user_type
	self.MaxNum = maxnum
	self.current_num = 0

	self.idn = USER_ID_NULL

	//
	self.list = make(map[int]i_user)

	//
	if user_type == USER_NORMAL {
		self.LogName = LOG_USER
		logout.LogAdd(logout.LogLevel_Info, LOG_USER, true, true)
	} else {
		self.LogName = LOG_USER_TEMP
		logout.LogAdd(logout.LogLevel_Info, LOG_USER_TEMP, true, true)
	}
	return true
}

func (self *UserManager) Release() {

	self.del_user_all()
}

//
func (self *UserManager) IDN() int {
	if self.idn >= USER_ID_MAX || self.idn <= 0 {
		self.idn = USER_ID_NULL
	}
	// Not null, has null + 1
	self.idn = self.idn + 1
	return self.idn
}

//
func (self *UserManager) GetUser(id int) i_user {
	user := self.get_user_by_id(id)
	if user == nil {
		return nil
	}
	return user
}

func (self *UserManager) get_user_by_id(id int) i_user {
	//
	self.lock.Lock()
	user, ok := self.list[id]
	self.lock.Unlock()
	if !ok || id != user.ID() {
		return nil
	}

	return user
}

//
func (self *UserManager) AddUser(user i_user) bool {
	if user == nil {
		return false
	}

	if user.ID() == 0 {
		user.parent().ID = self.IDN()
	}
	return self.add_user_impl(user.ID(), user)
}

func (self *UserManager) add_user_impl(id int, user i_user) bool {
	if self.current_num+1 >= self.MaxNum {
		return false
	}

	if id <= 0 || user.ID() != id {
		return false
	}

	self.lock.Lock()
	_, ok := self.list[id]
	self.lock.Unlock()
	if ok {
		return false
	}

	if len(user.Type()) == 0 {
		switch user.(type) {
		case *TTempUser:
			user.parent().Type = USER_TEMP
			break
		case *TUser:
			user.parent().Type = USER_NORMAL
			break
		default:
			return false
		}
	}
	if user.Type() != self.Type {
		return false
	}

	if user.Status() == USER_STATUS_NULL {
		if !user.initialize(id) {
			return false
		}
	} else if user.Status() >= USER_STATUS_ALLOC {
		return false
	}

	self.lock.Lock()
	self.list[id] = user
	self.lock.Unlock()

	user.parent().status = USER_STATUS_ALLOC

	self.current_num += 1
	return true
}

//
func (self *UserManager) DelUser(user i_user) bool {
	if user == nil || user.ID() <= 0 {
		return false
	}
	return self.del_user_impl(user.ID(), user)
}

func (self *UserManager) DelUserByID(id int) bool {
	user := self.get_user_by_id(id)
	if user == nil || user.ID() <= 0 {
		return false
	}
	return self.del_user_impl(user.ID(), user)
}

func (self *UserManager) del_user_impl(id int, user i_user) bool {
	if id <= 0 || id != user.ID() {
		return false
	}

	self.lock.Lock()
	v, ok := self.list[id]
	self.lock.Unlock()
	if !ok || v.ID() != user.ID() {
		return false
	}

	if v.parent().has_alloc() {
		self.free_user_impl(v)
	}

	self.lock.Lock()
	delete(self.list, id)
	num := len(self.list)
	self.lock.Unlock()

	v.parent().status = STATUS_FREE

	self.current_num -= 1
	// fixed total
	if self.current_num <= 0 || self.current_num != num {
		self.current_num = num
	}
	return true
}

func (self *UserManager) free_user_impl(user i_user) bool {
	if user == nil {
		return false
	}

	if user.Status() >= USER_STATUS_INIT {
		user.release()
	}
	return true
}

func (self *UserManager) del_user_all() {
	self.lock.Lock()
	for n, v := range self.list {
		defer self.del_user_impl(n, v)
	}
	self.lock.Unlock()

	self.current_num = 0
}

// Golang does not support class or structure generics,
// nor does it support parent class or structure pointer conversion
// Using `super` pseudo call
package gameserver

const (
	USER_TEMP   = "USER_TEMP"
	USER_NORMAL = "USER"

	USER_STATUS_FREE  = -1
	USER_STATUS_NULL  = 0
	USER_STATUS_INIT  = 1
	USER_STATUS_ALLOC = 2
	USER_STATUS_USING = 3
)

//
type i_user interface {
	parent() *t_user_base

	initialize(id int) bool
	release()

	Type() string
	ID() int
	RemoteAddress() string

	Load(cid uint32, address string) bool

	Status() int
}

type t_user_base struct {
	//
	Type string
	ID   int
	//
	SID           uint32
	RemoteAddress string

	//
	status int
}

func (self *t_user_base) has_alloc() bool {
	return self.status >= USER_STATUS_ALLOC
}

func (self *t_user_base) free() {
	self.SID = 0
	self.RemoteAddress = ""
}

func (self *t_user_base) load(sid uint32, address string) bool {
	self.SID = sid
	self.RemoteAddress = address
	return true
}

//
type TTempUser struct {
	i_user
	super t_user_base
}

//
type TUser struct {
	i_user
	super t_user_base

	IDX   string
	Token string

	//
	ClientTimestamp32 uint32
	ServerTimestamp32 uint32

	//
	ServerID    int
	ServerName  string
	ServerToken string

	//
	crypto_level int
	crypto_key   string
}

//
func (self *TTempUser) parent() *t_user_base  { return &self.super }
func (self *TTempUser) Type() string          { return self.super.Type }
func (self *TTempUser) ID() int               { return self.super.ID }
func (self *TTempUser) RemoteAddress() string { return self.super.RemoteAddress }
func (self *TTempUser) Status() int           { return self.super.status }

//
func (self *TUser) parent() *t_user_base  { return &self.super }
func (self *TUser) Type() string          { return self.super.Type }
func (self *TUser) ID() int               { return self.super.ID }
func (self *TUser) RemoteAddress() string { return self.super.RemoteAddress }
func (self *TUser) Status() int           { return self.super.status }

//
func (self *TTempUser) initialize(id int) bool {
	self.super.Type = USER_TEMP

	self.super.ID = id

	self.super.status = USER_STATUS_INIT
	return true
}

func (self *TTempUser) release() {
	self.super.free()

	self.super.status = USER_STATUS_NULL
	return
}

func (self *TTempUser) Load(sid uint32, address string) bool {
	if !self.super.load(sid, address) {
		return false
	}
	return true
}

//
func (self *TUser) initialize(id int) bool {
	self.super.Type = USER_NORMAL

	self.super.ID = id

	self.IDX = ""
	self.Token = ""

	self.ServerID = 0
	self.ServerName = ""
	self.ServerToken = ""

	self.crypto_level = 0
	self.crypto_key = ""

	self.super.status = USER_STATUS_INIT
	return true
}

func (self *TUser) release() {
	self.super.free()

	self.IDX = ""
	self.Token = ""

	self.ServerID = 0

	self.super.status = USER_STATUS_NULL
	return
}

func (self *TUser) Load(sid uint32, address string) bool {
	if !self.super.load(sid, address) {
		return false
	}
	return true
}

func (self *TUser) LoadCrypto(level int, key string) bool {
	self.crypto_level = level
	self.crypto_key = key
	if len(self.crypto_key) == 0 {
		self.crypto_level = 0 // not encrypt
	} else if len(self.crypto_key) > 0 && self.crypto_level == 0 {
		self.crypto_level = 1
	}
	return true
}

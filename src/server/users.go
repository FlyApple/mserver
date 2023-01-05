package server

import (
	"fmt"
	"time"

	mredis "mcmcx.com/mserver/modules/redis"
	"mcmcx.com/mserver/src/database"
	"mcmcx.com/mserver/src/gameserver"
	"mcmcx.com/mserver/src/util"
)

// WEB Authentication shared data
type DBAuthDataSub struct {
	ID        string `json:"id"`
	Code      string `json:"code"`      //8 lowercase and number
	Timestamp int64  `json:"timestamp"` //
}

type DBAuthData struct {
	IDX       string                   `json:"idx"`       //10 account idx
	Code      string                   `json:"code"`      //6 lowercase and number
	Token     string                   `json:"token"`     //16 hash
	Timestamp int64                    `json:"timestamp"` //
	List      map[string]DBAuthDataSub `json:"list"`
}

// User authentication API (application programming interface ） (idx, token)
// or TLI(temp login interface) (idx, code)
//
type DBUserAuth struct {
	IDX       string `json:"idx"`        //10 account idx
	Code      string `json:"code"`       //8 lowercase and number
	Token     string `json:"token"`      //32 hash
	Timestamp int64  `json:"timestamp"`  //
	TimeLast  string `json:"time_last"`  //(UPDATE AUTO)
	TotalUsed int    `json:"total_used"` //(UPDATE AUTO) Used token auth total
	Expired   float32
	Status    int //0: ok, -1:invalid
}

//
func DB_get_auth_data(idx string) *DBAuthData {
	var data DBAuthData
	result := mredis.GetJson[DBAuthData]("user_"+idx, &data)
	if !result {
		// return nil
	}
	// idx same,
	if idx != data.IDX {
		//return nil
	}
	return &data
}

func DB_get_user_auth(idx string) *DBUserAuth {
	var data DBUserAuth
	result := mredis.GetJson[DBUserAuth]("user_auth_"+idx, &data)
	if !result {
		return nil
	}
	// idx same,
	if idx != data.IDX {
		return nil
	}

	// Expired Time set status : -1
	data.Expired = util.ExpiredTimestamp64(uint64(data.Timestamp), util.TIME_DAY)
	if data.Expired <= 0 {
		data.Status = -1
	}
	return &data
}

func DB_update_user_auth(idx string, user_auth *DBUserAuth) bool {
	if user_auth == nil {
		return false
	}

	user_auth.TotalUsed++
	user_auth.TimeLast = util.DateFormat(time.Now(), 3)

	//
	result := mredis.PushJson[DBUserAuth]("user_auth_"+idx, user_auth, util.TIME_DAY)
	if !result {
		return false
	}
	return true
}

//
func user_auth_init(auth_data *RequestAuthData) *DBUserAuth {

	//
	var code = util.GenerateAuthCode(4)
	var rand = util.GenerateAuthCode(0)
	var token = util.SHA256(auth_data.Code + "_" + code + "_" + rand)

	// Get user auth
	var db_user_auth = DB_get_user_auth(auth_data.IDX)
	if db_user_auth == nil || db_user_auth.Status < 0 {
		db_user_auth = &DBUserAuth{
			IDX:       auth_data.IDX,
			Code:      "",
			Token:     "",
			Timestamp: -1,
			TotalUsed: 0, //
			Status:    0,
		}
	}

	// Must be reinitialized
	db_user_auth.Timestamp = int64(util.GetTimeStamp64())
	db_user_auth.Code = code
	db_user_auth.Token = token
	db_user_auth.TotalUsed = 0
	db_user_auth.Status = 0
	return db_user_auth
}

func user_data_init(auth_data *RequestAuthData) *database.DBUserData {
	// Get user data
	var db_user_data = database.DB_get_user_data(auth_data.IDX)
	if db_user_data == nil || db_user_data.Status < 0 {
		//
		db_user_data = &database.DBUserData{
			IDX:       auth_data.IDX,
			Timestamp: int64(util.GetTimeStamp64()),
			AuthTime:  0,
			PKey:      "",
			Status:    0,
		}
	}

	if db_user_data.AuthTime == 0 ||
		util.ExpiredTimestamp64(uint64(db_user_data.AuthTime), 2*util.TIME_HOUR) <= 0 {

		pkey, _, _ := util.ECCGenkey()
		db_user_data.PKey = util.ECCX509PrivateKeyEncoding(pkey)
		db_user_data.PKeyHash = util.MD5(db_user_data.PKey)

		db_user_data.AuthTime = int64(util.GetTimeStamp64())
	}

	return db_user_data
}

//
func U_auth_token(auth_data *t_auth_data) int {
	if auth_data == nil {
		return -1 // error
	}

	//
	auth_data.result = -1

	// Auth token 32, 64 or 128 bytes
	if len(auth_data.idx) < 10 || (len(auth_data.token) != 32 && len(auth_data.token) != 64) {
		return -1 // error
	}

	var db_user_auth = DB_get_user_auth(auth_data.idx)
	if db_user_auth == nil || db_user_auth.Status < 0 {
		return -1 // internal error
	}
	if db_user_auth.Expired <= 0 {
		auth_data.result = -2
		return -2 // expired time
	}
	if db_user_auth.Token != auth_data.token {
		auth_data.result = 0
		return 0 // failed, not error
	}

	DB_update_user_auth(auth_data.idx, db_user_auth)
	auth_data.result = 1
	return 1 //ok
}

//
func U_user_auth(auth_data *RequestAuthData, result_data *ResponseAuthData) int {
	if len(auth_data.IDX) < 10 || len(auth_data.Code) != 8 || len(auth_data.Token) != 32 {
		return -1
	}

	db_auth_data := DB_get_auth_data(auth_data.IDX)
	if db_auth_data == nil {
		return -1
	}

	if db_auth_data.Token != auth_data.Token {
		//return -1
	}

	//
	result_data.IDX = auth_data.IDX
	result_data.DateTime = util.DateFormat(time.Now(), 3)

	//
	sub, ok := db_auth_data.List[auth_data.Code]
	if !ok || sub.Code != auth_data.Code {
		sub = DBAuthDataSub{}
		//return -1
	}

	// time expired
	if util.ExpiredTimestamp64(uint64(db_auth_data.Timestamp), util.TIME_DAY) <= 0 ||
		util.ExpiredTimestamp64(uint64(sub.Timestamp), util.TIME_DAY) <= 0 {
		//return -2 // expired time
	}

	var db_user_auth = user_auth_init(auth_data)
	if db_user_auth == nil {
		return -3 // internal error
	}

	//
	var db_user_data = database.DB_get_user_data(auth_data.IDX)
	if db_user_data == nil || db_user_data.Status < 0 {
		return -3 // internal error
	}

	// Update auth time
	db_user_data.AuthTime = int64(util.GetTimeStamp64())

	//
	DB_update_user_auth(db_user_auth.IDX, db_user_auth)

	// Result data
	pkey := util.ECCX509PrivateKeyDecoding(db_user_data.PKey)

	result_data.Code = db_user_auth.Code
	result_data.Token = db_user_auth.Token
	result_data.PKey = util.ECCPublicKeyEncoding(&pkey.PublicKey)
	result_data.PKeyHash = util.MD5(result_data.PKey)

	// Server data
	db_user_data.ServerID = 0
	db_user_data.ServerName = ""
	db_user_data.ServerToken = ""
	db_user_data.ServerUserToken = ""

	result_data.ServerID = 0
	result_data.ServerToken = ""
	result_data.ServerAddress = "0.0.0.0"
	result_data.ServerPort = 0
	result_data.ServerUserToken = ""

	var server = gameserver.GServerManager.GetIdleServer()
	if server != nil {
		var server_info = gameserver.GServerManager.GetServerInfo(server.ID)

		db_user_data.ServerID = server.ID
		db_user_data.ServerToken = server.Token

		result_data.ServerID = server.ID
		result_data.ServerToken = server.Token
		result_data.ServerName = ""
		result_data.ServerAddress = "127.0.0.1"
		result_data.ServerPort = 9000
		if server_info != nil {
			db_user_data.ServerName = server_info.Title

			result_data.ServerName = server_info.Title
			result_data.ServerAddress = server_info.GateAddress
			result_data.ServerPort = server_info.GatePort
		}

		var text = fmt.Sprintf("%d_%s_%s_%s", result_data.ServerID, result_data.ServerToken,
			result_data.IDX, result_data.Code)
		db_user_data.ServerUserToken = util.MD5(text)

		result_data.ServerUserToken = db_user_data.ServerUserToken
	}

	database.DB_update_user_data(db_user_data.IDX, db_user_data)

	//
	return 0
}

func U_user_data(auth_data *t_auth_data, result_data *ResponseUserData) int {
	if auth_data == nil {
		return -1
	}

	result_data.IDX = auth_data.idx

	//
	var db_user_data = database.DB_get_user_data(auth_data.idx)
	if db_user_data == nil || db_user_data.Status < 0 {
		return -3 // internal error
	}

	result_data.LastTime = db_user_data.TimeLast

	//
	database.DB_update_user_data(db_user_data.IDX, db_user_data)

	//
	return 0
}

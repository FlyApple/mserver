package server

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"mcmcx.com/mserver/src/util"
)

//
type TAuthToken struct {
	IDX   string `form:"idx"`
	Token string `form:"token"`
}

type t_auth_data struct {
	idx       string
	code      string
	token     string
	timestamp uint64
	ip_client string
	ip_remote string
	result    int
}

// API: auth
type RequestAuthData struct {
	IDX   string `form:"idx"`
	Code  string `form:"code"`
	Token string `form:"token"`
}

type ResponseAuthData struct {
	IDX string `json:"idx"`
	// WEB API
	Code     string `json:"code"`      // TLI pass
	Token    string `json:"token"`     // API Token
	PKey     string `json:"pkey"`      //PublicKey
	PKeyHash string `json:"pkey_hash"` //PublicKey Hash
	// Server
	ServerID      int    `json:"server_id"`
	ServerName    string `json:"server_name"`
	ServerToken   string `json:"server_token"`
	ServerAddress string `json:"server_address"`
	ServerPort    int    `json:"server_port"`
	// Server User
	ServerUserToken string `json:"server_user_token"`
	// Time
	DateTime string `json:"date_time"`
}

// API: user
type ResponseUserData struct {
	IDX      string `json:"idx"`
	LastTime string `json:"last_time"`
}

//
func l_init_data(ctx *gin.Context) util.TMA {
	return util.TMA{
		"address":     ctx.ClientIP(),
		"timestamp":   util.GetTimeStamp(),
		"timestamp64": util.GetTimeStamp64(),
	}
}

func l_init_auth(ctx *gin.Context) (int, *t_auth_data) {
	//
	var auth_data t_auth_data = t_auth_data{
		idx:       "",
		code:      "",
		timestamp: util.GetTimeStamp64(),
		result:    -1,
	}

	auth_data.ip_client = ctx.ClientIP()
	auth_data.ip_remote = ctx.RemoteIP()

	//
	var auth_token TAuthToken
	err := ctx.ShouldBind(&auth_token)
	if err != nil {
		return -1, nil // ERROR
	}

	auth_token.IDX = strings.TrimSpace(auth_token.IDX)
	auth_token.Token = strings.ToUpper(strings.TrimSpace(auth_token.Token))

	//
	auth_data.idx = auth_token.IDX
	auth_data.token = auth_token.Token

	result := U_auth_token(&auth_data)
	if result < 0 {
		return -1, &auth_data // Auth Error
	}
	return auth_data.result, &auth_data
}

func l_init_error_o(data *util.TMA, err error) *util.TMA {
	var code = util.RESULT_ERROR_UNKNOW
	var message = ""
	if err != nil {
		message = err.Error()
	}
	return l_init_result_sx(data, code, util.STATUS_ERROR_UNKNOW, message)
}

func l_init_error_s(data *util.TMA, code int, message string) *util.TMA {
	return l_init_result_sx(data, code, util.STATUS_ERROR_UNKNOW, message)
}

func l_init_result_s(data *util.TMA, code int, status string) *util.TMA {
	return l_init_result_sx(data, code, status, "")
}

func l_init_result_sx(data *util.TMA, code int, status string, message string) *util.TMA {
	var temp = util.TMA{
		"result_code":    code,
		"result_status":  status,
		"result_message": message,
		"result_error":   0, //0: not error, has failed; x < 0: error
	}

	// ERROR
	if code < 0 || status == util.STATUS_ERROR_UNKNOW {
		//Failed not error, the code not set -1
		if status != util.STATUS_FAILED {
			temp["result_error"] = -1
		}
		//
		switch temp["result_code"] {
		case util.RESULT_ERROR_INVALID:
			temp["result_status"] = util.STATUS_ERROR_INVALID
			break
		case util.RESULT_ERROR_INTERNAL:
			temp["result_status"] = util.STATUS_ERROR_INTERNAL
			break
		case util.RESULT_ERROR_NOT_FOUND:
			temp["result_status"] = util.STATUS_ERROR_NOT_FOUND
			break
		case util.RESULT_ERROR_NOT_EXIST:
			temp["result_status"] = util.STATUS_ERROR_NOT_EXIST
			break
		}
	}

	return util.MapConcatPtr(data, &temp)
}

func l_result_data(level int, data *util.TMA, result any) *util.TMA {
	var encrypt_level = level
	(*data)["level"] = encrypt_level

	if encrypt_level == 0 || result == nil {
		(*data)["data"] = result
	} else {
		(*data)["data"] = ""
		buffer, err := json.Marshal(result)
		if err == nil {
			temp := base64.URLEncoding.EncodeToString(buffer)
			(*data)["data"] = temp
		}
	}
	return data
}

func handler_result_error(ctx *gin.Context, err error) {
	data := l_init_data(ctx)
	result := l_init_error_o(&data, err)

	ctx.JSON(200, *result)
}

func handler_result_error_s(ctx *gin.Context, code int, message string) {

	data := l_init_data(ctx)
	result := l_init_error_s(&data, code, message)

	ctx.JSON(200, result)
}

func handler_result_error_n(ctx *gin.Context, code int) {

	data := l_init_data(ctx)
	result := l_init_error_s(&data, code, "")

	ctx.JSON(200, *result)
}

func handler_result_null(ctx *gin.Context) {

	data := l_init_data(ctx)
	result := l_init_result_s(&data, util.RESULT_OK, util.STATUS_OK)
	result = l_result_data(0, result, nil)

	ctx.JSON(200, *result)
}

func handler_result_data(ctx *gin.Context, result_data any) {

	data := l_init_data(ctx)
	result := l_init_result_s(&data, util.RESULT_OK, util.STATUS_OK)
	result = l_result_data(0, result, result_data)

	ctx.JSON(200, *result)
}

func handler_result_ns(ctx *gin.Context, code int, status string) {

	data := l_init_data(ctx)
	result := l_init_result_s(&data, code, status)

	ctx.JSON(200, *result)
}

//
func R_handler_ping(ctx *gin.Context) {
	ctx.JSON(200, gin.H{
		"message":  "pong",
		"time_utc": util.DateFormat(time.Now().UTC(), 9),
		"time_now": util.DateFormat(time.Now(), -1),
	})
}

// default: handler -> hello
func R_handler_hello(ctx *gin.Context) {
	data := l_init_data(ctx)
	ctx.JSON(200, data)
}

func R_handler_auth(ctx *gin.Context) {
	var auth_data RequestAuthData
	if ctx.ShouldBind(&auth_data) != nil {
		handler_result_error_n(ctx, util.RESULT_ERROR_INTERNAL)
		return
	}

	//
	auth_data.IDX = strings.TrimSpace(auth_data.IDX)
	auth_data.Code = strings.ToLower(strings.TrimSpace(auth_data.Code))
	auth_data.Token = strings.ToUpper(strings.TrimSpace(auth_data.Token))
	if len(auth_data.IDX) < 6 || len(auth_data.Code) < 6 ||
		(len(auth_data.Token) != 16 && len(auth_data.Token) != 32) {
		handler_result_error_n(ctx, util.RESULT_ERROR_INVALID)
		return
	}

	var result_data ResponseAuthData
	var result = U_user_auth(&auth_data, &result_data)
	if result < 0 {
		handler_result_ns(ctx, util.RESULT_FAILED, util.STATUS_FAILED)
		return
	}

	handler_result_data(ctx, result_data)
}

func R_handler_user(ctx *gin.Context) {
	// Error or failed
	res, auth := l_init_auth(ctx)
	if res <= 0 {
		handler_result_error_n(ctx, util.RESULT_ERROR_INTERNAL)
		return
	}

	var result_data ResponseUserData
	result := U_user_data(auth, &result_data)
	if result < 0 {
		handler_result_ns(ctx, util.RESULT_FAILED, util.STATUS_FAILED)
		return
	}

	// data
	handler_result_data(ctx, result_data)
}

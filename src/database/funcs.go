package database

import (
	"time"

	mredis "mcmcx.com/mserver/modules/redis"
	"mcmcx.com/mserver/src/util"
)

func DB_get_user_data(idx string) *DBUserData {
	var data DBUserData
	result := mredis.GetJson[DBUserData]("user_data_"+idx, &data)
	if !result {
		return nil
	}
	// idx same,
	if idx != data.IDX {
		return nil
	}

	return &data
}

func DB_update_user_data(idx string, user_data *DBUserData) bool {
	if user_data == nil {
		return false
	}

	user_data.TimeLast = util.DateFormat(time.Now(), 3)

	result := mredis.PushJson[DBUserData]("user_data_"+idx, user_data, util.TIME_KEEPN)
	if !result {
		return false
	}
	return true
}

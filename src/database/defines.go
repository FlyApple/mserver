package database

// (Redis) User data
type DBUserData struct {
	IDX       string `json:"idx"`       //10 account idx
	Timestamp int64  `json:"timestamp"` //create timestamp
	AuthTime  int64  `json:"auth_time"` //Auth time
	TimeLast  string `json:"time_last"` //(UPDATE AUTO)
	// Crypto
	PKey     string `json:"pkey"`
	PKeyHash string `json:"pkey_hash"`
	// Server
	ServerID        int    `json:"server_id"`
	ServerName      string `json:"server_name"`
	ServerToken     string `json:"server_token"`
	ServerUserToken string `json:"server_user_token"`
	Status          int
}

type DBUserKey struct {
	IDX string `json:"idx"` //10 account idx
	// Crypto
	PKey     string `json:"pkey"`
	PKeyHash string `json:"pkey_hash"`
}

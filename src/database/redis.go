package database

import (
	"crypto/tls"
	"crypto/x509"
	"errors"

	"github.com/go-redis/redis/v9"

	mredis "mcmcx.com/mserver/modules/redis"
	"mcmcx.com/mserver/src/logout"
	"mcmcx.com/mserver/src/util"
)

//
const LOG_REDIS = "REDIS"

var redis_info mredis.TInfo
var redis_instance *redis.Client = nil

//
func redis_loadinfo(filename string) bool {

	//
	if !util.LoadJsonFromFile[mredis.TInfo](filename, &redis_info) {
		logout.LogError("[Load] Read redis info fail")
		return false
	}

	if redis_info.Port <= 0 {
		logout.LogError("[Load] Read redis info error")
		return false
	}

	if len(redis_info.TLSKey) > 0 && len(redis_info.TLSCrt) > 0 {
		redis_info.UseTLS = true
		logout.LogWithName(LOG_REDIS, "[Load] Use TLS (OK)")
	}

	return true
}

//
func RedisRelease() int {
	return mredis.Release()
}

//
func RedisInitialize(filename string) bool {

	logout.LogAdd(logout.LogLevel_Info, LOG_REDIS, true, true)

	//
	if !redis_loadinfo(filename) {
		logout.LogError("[Load] redis information server error")
		return false
	}

	//
	var tls_config *tls.Config = nil
	if redis_info.UseTLS {
		pool := util.LoadCertCAFromFile(redis_info.TLSCA)
		if pool == nil {
			logout.LogWithName(LOG_REDIS, "[Load] Load TLS Error : CA file (", redis_info.TLSCA, ")")
			return false
		}
		cert := util.LoadCertFromFiles(redis_info.TLSCrt, redis_info.TLSKey)
		if cert == nil {
			logout.LogWithName(LOG_REDIS, "[Load] Load TLS Error : key file (", redis_info.TLSKey, "), crt file (", redis_info.TLSCrt, ")")
			return false
		}

		tls_config = &tls.Config{
			MinVersion: tls.VersionTLS12,
			//ServerName:         "localhost",
			InsecureSkipVerify: true,
			RootCAs:            pool,
			Certificates: []tls.Certificate{
				*cert,
			},
			ClientAuth: tls.RequireAndVerifyClientCert,
			//ClientCAs:  pool,
			VerifyConnection: func(cs tls.ConnectionState) error {
				if len(cs.PeerCertificates) == 0 {
					return errors.New("The peer certificates not found")
				}
				var cert = cs.PeerCertificates[0]
				_, err := cert.Verify(x509.VerifyOptions{
					DNSName: "",
					Roots:   pool,
				})
				if err != nil {
					return err
				}
				return nil
			},
		}

	}
	redis_info.TLSConfig = tls_config

	redis_instance = mredis.NewAndInitialize(&redis_info)
	if redis_instance == nil {
		logout.LogError("[Load] Connect redis server error")
		return false
	}

	return true
}

package util

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"os"
)

// IO Files
func ReadBytesFromFile(filename string) []byte {
	file, err := os.OpenFile(filename, os.O_RDWR, 0666)
	if err != nil {
		return nil
	}

	length, _ := file.Seek(0, os.SEEK_END)
	file.Seek(0, os.SEEK_SET)
	buffer := make([]byte, length)
	file.Read(buffer)
	file.Close()

	return buffer
}

func WriteBytesToFile(filename string, buffer []byte) bool {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return false
	}

	//
	if len(buffer) > 0 {
		file.Seek(0, os.SEEK_SET)
		file.Write(buffer)
	}
	file.Close()
	return true
}

//JSON Files
func LoadJsonFromFile[T TAny](filename string, data *T) bool {
	buffer := ReadBytesFromFile(filename)
	if buffer == nil {
		println("[Error] Read file (" + filename + ") error.")
		return false
	}

	var err = json.Unmarshal(buffer, data)
	if err != nil {
		println("[Error] Json unmarshal (" + filename + ") fail, Error: " + err.Error())
		return false
	}
	return true
}

func SaveJsonToFile[T TAny](filename string, data *T) bool {
	buffer, err := json.Marshal(data)
	if err != nil {
		println("[Error] Json marshal (" + filename + ") error.")
		return false
	}

	if !WriteBytesToFile(filename, buffer) {
		println("[Error] Save file (" + filename + ") error.")
		return false
	}
	return true
}

//CERT Files
func LoadCertCAFromFile(filename string) *x509.CertPool {
	pem, err := os.ReadFile(filename)
	if err != nil {
		println("[Error] Load cert CA file (" + filename + ") error.")
		return nil
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		println("[Error] Load cert CA file (" + filename + ") error.")
		return nil
	}
	return pool
}

func LoadCertFromFiles(crt_filename string, key_filename string) *tls.Certificate {
	cert, err := tls.LoadX509KeyPair(crt_filename, key_filename)
	if err != nil {
		println("[Error] Load cert files (" + crt_filename + ", " + key_filename + ") error : " + err.Error())
		return nil
	}
	return &cert
}

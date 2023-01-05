package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"mcmcx.com/mserver/modules/zinx/zpack"
	"mcmcx.com/mserver/src/util"
)

//
type RequestAuth struct {
	IDX   string `json:"idx"`
	Code  string `json:"code"`
	Token string `json:"token"`
}

type ResponseResult struct {
	Address     string `json:"address"`
	Timestamp   uint32 `json:"timestamp"`
	Timestamp64 uint64 `json:"timestamp64"`
	Error       int    `json:"result_error"`
	Code        int    `json:"result_code"`
	Message     string `json:"result_message"`
	Status      string `json:"result_status"`
	Data        any    `json:"data"`
}

//
func https_auth(url string) (map[string]interface{}, bool) {

	cert := util.LoadCertFromFiles("certs/https.crt", "certs/https_rsa_2048.pem.unsecure")
	//cert := util.LoadCertFromFiles("certs/client.crt", "certs/client_rsa_2048.pem")
	if cert == nil {
		return nil, false
	}

	tls_config := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true,
		Certificates: []tls.Certificate{
			*cert,
		},
		ClientAuth: tls.RequireAndVerifyClientCert,
		//ClientCAs:  pool,
		VerifyConnection: func(cs tls.ConnectionState) error {
			if len(cs.PeerCertificates) == 0 {
				return errors.New("The peer certificates not found")
			}
			return nil
		},
	}
	tranport := &http.Transport{
		TLSClientConfig: tls_config,
	}
	client := http.Client{
		Transport: tranport,
	}

	// From Web Auth
	idx := "117216368478"
	code := "12095509"
	token := "e10adc3949ba59abbe56e057f20f883e"

	//API
	url = url + "/auth"
	//Parameters
	url = url + "?"
	url = url + "idx=" + idx
	url = url + "&code=" + code
	url = url + "&token=" + token

	// Get or Post, Test using Get
	response, err := client.Get(url)
	if err != nil {
		return nil, false
	}
	defer response.Body.Close()
	//len := response.ContentLength
	buffer, err := ioutil.ReadAll(response.Body)

	var body ResponseResult
	err = json.Unmarshal(buffer, &body)
	if err != nil {
		println("Error : ", err.Error())
		return nil, false
	}

	/*
		var auth = RequestAuth {
			IDX:  idx,
			Code: code,
			Token:  token,
		}
		data, _ := json.Marshal(auth)
		content := bytes.NewBuffer(data)
		content_type := "application/json;charset=utf-8"
		response, err := client.Post(url, content_type, content)
		if err != nil {
			return false
		}
		len := response.ContentLength
		buffer := make([]byte, len)
		defer response.Body.Read(body)
		response.Body.Close()

		var body any
		err = json.Unmarshal(buffer, &data)
		if err != nil {
			return false
		}
	*/

	if body.Error < 0 {
		println("Result : Error ", body.Error, ", Status :", body.Status)
		return nil, false
	}

	if body.Data == nil {
		body.Error = -1
		println("Result : Error ", body.Error, ", Status :", body.Status)
		return nil, false
	}

	//
	result := body.Data.(map[string]interface{})
	result["address"] = body.Address
	return result, true
}

//
func send_hello(conn net.Conn) int {
	dp := zpack.NewDataPack(4096)

	buffer := zpack.NewMessageBuffer(nil)

	pack, _ := dp.Pack(zpack.NewMsgPackage(0x00, buffer.Data()))
	len, err := conn.Write(pack)
	if err != nil {
		fmt.Println("write error err ", err)
		return -1
	}
	return len
}

func send_ping(conn net.Conn) int {
	dp := zpack.NewDataPack(4096)

	buffer := zpack.NewMessageBuffer(nil)
	buffer.WriteUInt32(util.GetTimeStamp())
	buffer.WriteUInt64(util.GetTimeStamp64())

	pack, _ := dp.Pack(zpack.NewMsgPackage(0x01, buffer.Data()))
	len, err := conn.Write(pack)
	if err != nil {
		fmt.Println("write error err ", err)
		return -1
	}
	return len
}

// Handler 09: Auth
// Client Packet:
//   - User IDX (string)
//   - User Timestamp (uint client)
//   - Server ID (int)
//   - Server Token (MD5 16bytes)
//   - User Remote Address (string)
//   - User Authentication Token (MD5 string)
//   - User PublicKey (ECC bytes)
func send_auth(conn net.Conn, idx string, server_id int32, server_token string,
	address string, token string) int {
	dp := zpack.NewDataPack(4096)

	buffer := zpack.NewMessageBuffer(nil)
	buffer.WriteStringL(idx)
	buffer.WriteUInt32(util.GetTimeStamp())

	buffer.WriteInt32(server_id)
	buffer.WriteStringL(server_token)

	buffer.WriteStringL(address)
	buffer.WriteStringL(token)
	buffer.WriteBytesL([]byte(""))

	pack, _ := dp.Pack(zpack.NewMsgPackage(0x09, buffer.Data()))
	len, err := conn.Write(pack)
	if err != nil {
		fmt.Println("write error err ", err)
		return -1
	}
	return len
}

func send_user(conn net.Conn) int {
	dp := zpack.NewDataPack(4096)

	buffer := zpack.NewMessageBuffer(nil)
	buffer.WriteStringL("117216368478")

	pack, _ := dp.Pack(zpack.NewMsgPackage(0x10, buffer.Data()))
	len, err := conn.Write(pack)
	if err != nil {
		fmt.Println("write error err ", err)
		return -1
	}
	return len
}

func recv_data(conn net.Conn, message **zpack.Message, buffer **zpack.MessageBuffer) int {
	dp := zpack.NewDataPack(4096)

	// Set 1 seconds timeout
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	//先读出流中的head部分
	head_data := make([]byte, dp.GetHeadLen())
	_, err := io.ReadFull(conn, head_data) //ReadFull 会把msg填充满为止
	if err != nil {
		e, ok := err.(*net.OpError)
		if ok && e.Timeout() {
			return 0
		}
		fmt.Println("read head error")
		return -1
	}

	//将HeadData字节流 拆包到msg中
	head, err := dp.Unpack(head_data)
	if err != nil {
		fmt.Println("server unpack err:", err)
		return -1
	}

	if head.GetDataLen() == 0 {
		fmt.Println("server message length invalid:")
		return -1
	}

	//msg 是有data数据的，需要再次读取data数据
	msg := head.(*zpack.Message)
	msg.Data = make([]byte, msg.GetDataLen())

	//根据dataLen从io中读取字节流
	len, err := io.ReadFull(conn, msg.Data)
	if err != nil {
		e, ok := err.(*net.OpError)
		if ok && e.Timeout() {
			return 0
		}
		fmt.Println("server unpack data err:", err)
		return -1
	}

	*message = msg
	*buffer = zpack.NewMessageBuffer(msg.Data)
	if *buffer == nil {
		return -1
	}
	return len
}

func main() {

	//
	data, ret := https_auth("https://127.0.0.1:8443")
	if !ret {
		return
	}

	var server_address = fmt.Sprintf("%s:%d",
		data["server_address"].(string), int(data["server_port"].(float64)))

	//
	conn, err := net.Dial("tcp", server_address)
	if err != nil {
		fmt.Println("client start err, exit!", err)
		return
	}

	if conn != nil {
		//发封包message消息
		//send_hello(conn)
		//send_ping(conn)
		send_auth(conn, data["idx"].(string), int32(data["server_id"].(float64)), data["server_token"].(string),
			data["address"].(string), data["server_user_token"].(string))

		var message *zpack.Message
		var buffer *zpack.MessageBuffer
		for recv_data(conn, &message, &buffer) >= 0 {
			if message == nil || buffer == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			switch message.ID {
			case 0x00:
				tm32 := buffer.ReadUInt32()
				tm64 := buffer.ReadUInt64()
				date := buffer.ReadStringL()
				println("(Test) Handler : (Hello) ", tm32, tm64, date)
				break
			case 0x01:
				tm32 := buffer.ReadUInt32()
				tm64 := buffer.ReadUInt64()
				println("(Test) Handler : (Ping) ", tm32, tm64)
				break
			case 0x09:
				result := buffer.ReadInt32()
				tm32 := buffer.ReadUInt32()
				if result >= 0 {
					idx := buffer.ReadStringL()
					if result >= 1 {
						server_id := buffer.ReadInt32()
						server_name := buffer.ReadStringL()
						println("(Test) Handler : (Auth) Result :", result, ", ", tm32,
							"idx:", idx, "Server:", server_id, " - ", server_name)

						send_user(conn)
					} else {
						println("(Test) Handler : (Auth) Result :", result, ", ", tm32,
							"idx:", idx)
					}
				} else {
					println("(Test) Handler : (Auth) Result :", result, ", ", tm32)
				}
				break
			case 0x10:
				idx := buffer.ReadStringL()
				println("(Test) Handler : (Auth) User IDX:", idx)
				break
			}

			message = nil
			buffer = nil
		}
		time.Sleep(1 * time.Second)
		conn.Close()
	}
}

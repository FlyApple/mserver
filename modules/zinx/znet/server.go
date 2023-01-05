package znet

import (
	"errors"
	"fmt"
	"net"

	"mcmcx.com/mserver/modules/zinx/ziface"
	"mcmcx.com/mserver/modules/zinx/zpack"
	"mcmcx.com/mserver/modules/zinx/zutils"
)

var zinxLogo = `                                        
              ██                        
              ▀▀                        
 ████████   ████     ██▄████▄  ▀██  ██▀ 
     ▄█▀      ██     ██▀   ██    ████   
   ▄█▀        ██     ██    ██    ▄██▄   
 ▄██▄▄▄▄▄  ▄▄▄██▄▄▄  ██    ██   ▄█▀▀█▄  
 ▀▀▀▀▀▀▀▀  ▀▀▀▀▀▀▀▀  ▀▀    ▀▀  ▀▀▀  ▀▀▀ 
                                        `
var topLine = `┌──────────────────────────────────────────────────────┐`
var borderLine = `│`
var bottomLine = `└──────────────────────────────────────────────────────┘`

//Server 接口实现，定义一个Server服务类
type TServer struct {
	//服务器的名称
	Name string
	//tcp4 or other
	Type string
	//服务绑定的IP地址
	Address string
	//服务绑定的端口
	Port int

	//
	PacketSize        uint32
	ConnectionsMaxNum int32
	WorkerPoolSize    int32 //业务工作Worker池的数量
	WorkerTaskMaxLen  int32
	MsgChanMaxLen     int32

	//
	//当前Server的消息管理模块，用来绑定MsgID和对应的处理方法
	msgHandler ziface.IMsgHandle
	//当前Server的链接管理器
	connectionManager ziface.IConnectionManager
	//该Server的连接创建时Hook函数
	OnConnectionStart func(data any, connection ziface.IConnection)
	//该Server的连接断开时的Hook函数
	OnConnectionStop func(data any, connection ziface.IConnection)

	exitChan chan struct{}

	packet ziface.IDataPack

	//
	data any
}

//NewServer 创建一个服务器句柄
func NewServer(config *zutils.TConfig, opts ...Option) ziface.IServer {
	//
	zutils.InitConfig(config)

	//
	s := &TServer{
		Name:              config.Name,
		Type:              config.Type,
		Address:           config.Address,
		Port:              config.Port,
		PacketSize:        config.PacketSize,
		ConnectionsMaxNum: config.ConnectionsMaxNum,
		WorkerPoolSize:    config.WorkerPoolSize,
		WorkerTaskMaxLen:  config.WorkerTaskMaxLen,
		MsgChanMaxLen:     config.MsgChanMaxLen,
		msgHandler:        NewMsgHandle(config.WorkerPoolSize, config.WorkerTaskMaxLen),
		connectionManager: NewConnectionManager(config.ConnectionsMaxNum),
		exitChan:          nil,
		packet:            zpack.Factory().NewPack(config.PacketSize, ziface.ZinxDataPack),
		data:              nil,
	}

	//更替打包方式
	for _, opt := range opts {
		opt(s)
	}

	return s
}

//============== 实现 ziface.IServer 里的全部接口方法 ========
func (s *TServer) SetDataPtr(data interface{}) {
	s.data = data
}

//Start 开启网络服务
func (s *TServer) Start() {
	fmt.Printf("[START] Server name: %s,listenner at IP: %s, Port %d is starting\n", s.Name, s.Address, s.Port)
	s.exitChan = make(chan struct{})

	//开启一个go去做服务端Linster业务
	go func() {
		//0 启动worker工作池机制
		s.msgHandler.StartWorkerPool()

		//1 获取一个TCP的Addr
		addr, err := net.ResolveTCPAddr(s.Type, fmt.Sprintf("%s:%d", s.Address, s.Port))
		if err != nil {
			fmt.Println("[WORKING] resolve tcp addr err: ", err)
			return
		}

		//2 监听服务器地址
		listener, err := net.ListenTCP(s.Type, addr)
		if err != nil {
			panic(err)
		}

		//已经监听成功
		fmt.Println("[WORKING] start Zinx server  (", s.Name, ") success, now listenning...")

		//TODO server.go 应该有一个自动生成ID的方法
		var cID uint32
		cID = 0

		go func() {
			//3 启动server网络连接业务
			for {
				//3.1 设置服务器最大连接控制,如果超过最大连接，则等待
				if s.connectionManager.Len() >= int(s.ConnectionsMaxNum) {
					fmt.Println("[WORKING] Exceeded the ConnectionMaxCount:", s.ConnectionsMaxNum, ", Wait:", AcceptDelay.duration)
					AcceptDelay.Delay()
					continue
				}

				//3.2 阻塞等待客户端建立连接请求
				conn, err := listener.AcceptTCP()
				if err != nil {
					//Go 1.16+
					if errors.Is(err, net.ErrClosed) {
						fmt.Println("[WORKING] Listener closed")
						return
					}
					fmt.Println("[WORKING] Accept error: ", err)
					AcceptDelay.Delay()
					continue
				}

				AcceptDelay.Reset()

				//3.3 处理该新连接请求的 业务 方法， 此时应该有 handler 和 conn是绑定的
				dealConn := NewConnection(s, conn, cID, s.WorkerPoolSize, s.MsgChanMaxLen, s.msgHandler)
				cID++

				//3.4 启动当前链接的处理业务
				go dealConn.Start()
			}
		}()

		select {
		case <-s.exitChan:
			err := listener.Close()
			if err != nil {
				fmt.Println("[WORKING] Listener close, error :", err)
			}
		}
	}()
}

//Stop 停止服务
func (s *TServer) Stop() {
	fmt.Println("[STOP] Zinx server , name :", s.Name)

	//将其他需要清理的连接信息或者其他信息 也要一并停止或者清理
	s.connectionManager.ClearAll()
	s.exitChan <- struct{}{}
	close(s.exitChan)
}

//Serve 运行服务
func (s *TServer) Serve() {
	s.Start()

	//TODO Server.Serve() 是否在启动服务的时候 还要处理其他的事情呢 可以在这里添加

	//阻塞,否则主Go退出， listenner的go将会退出
	select {}
}

//AddRouter 路由功能：给当前服务注册一个路由业务方法，供客户端链接处理使用
func (s *TServer) AddRouter(id uint32, router ziface.IRouter) bool {
	return s.msgHandler.AddRouter(id, router)
}

//GetConnectionManager 得到链接管理
func (s *TServer) GetConnectionManager() ziface.IConnectionManager {
	return s.connectionManager
}

//SetOnConnectionStart 设置该Server的连接创建时Hook函数
func (s *TServer) SetOnConnectionStart(hookFunc func(any, ziface.IConnection)) {
	s.OnConnectionStart = hookFunc
}

//SetOnConnectionStop 设置该Server的连接断开时的Hook函数
func (s *TServer) SetOnConnectionStop(hookFunc func(any, ziface.IConnection)) {
	s.OnConnectionStop = hookFunc
}

//CallOnConnectionStart 调用连接OnConnectionStart Hook函数
func (s *TServer) CallOnConnectionStart(connection ziface.IConnection) {
	if s.OnConnectionStart != nil {
		//fmt.Println("---> CallOnConnectionStart....")
		s.OnConnectionStart(s.data, connection)
	}
}

//CallOnConnectionStop 调用连接OnConnectionStop Hook函数
func (s *TServer) CallOnConnectionStop(connection ziface.IConnection) {
	if s.OnConnectionStop != nil {
		//fmt.Println("---> CallOnConnectionStop....")
		s.OnConnectionStop(s.data, connection)
	}
}

func (s *TServer) Packet() ziface.IDataPack {
	return s.packet
}

func printLogo() {
	fmt.Println(zinxLogo)
	fmt.Println(topLine)
	fmt.Println(fmt.Sprintf("%s [Github] https://github.com/aceld                    %s", borderLine, borderLine))
	fmt.Println(fmt.Sprintf("%s [tutorial] https://www.yuque.com/aceld/npyr8s/bgftov %s", borderLine, borderLine))
	fmt.Println(bottomLine)
}

func init() {
	printLogo()
	// nothing
}

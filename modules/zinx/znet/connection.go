package znet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"syscall"
	"time"

	"mcmcx.com/mserver/modules/zinx/ziface"
	"mcmcx.com/mserver/modules/zinx/zpack"
)

//Connection 链接
type Connection struct {
	//当前Conn属于哪个Server
	TCPServer ziface.IServer

	//
	MsgChanMaxLen  int32
	WorkerPoolSize int32

	//当前连接的socket TCP套接字
	Connection *net.TCPConn
	//当前连接的ID 也可以称作为SessionID，ID全局唯一
	ConnectionID uint32
	//消息管理MsgID和对应处理方法的消息管理模块
	MsgHandler ziface.IMsgHandle
	//告知该链接已经退出/停止的channel
	ctx    context.Context
	cancel context.CancelFunc
	//有缓冲管道，用于读、写两个goroutine之间的消息通信
	MsgBufferChan chan []byte

	sync.RWMutex
	//链接属性
	property map[string]interface{}
	////保护当前property的锁
	propertyLock sync.Mutex
	//当前连接的关闭状态
	isClosed bool
}

//NewConnection 创建连接的方法
func NewConnection(server ziface.IServer, connection *net.TCPConn, id uint32,
	workerPoolSize int32, msgChanMaxLen int32,
	msgHandler ziface.IMsgHandle) *Connection {
	//初始化Conn属性
	c := &Connection{
		TCPServer:      server,
		MsgChanMaxLen:  msgChanMaxLen,
		WorkerPoolSize: workerPoolSize,
		Connection:     connection,
		ConnectionID:   id,
		isClosed:       false,
		MsgHandler:     msgHandler,
		MsgBufferChan:  make(chan []byte, msgChanMaxLen),
		property:       nil,
	}

	//将新创建的Conn添加到链接管理中
	c.TCPServer.GetConnectionManager().Add(c)
	return c
}

//StartWriter 写消息Goroutine， 用户将数据发送给客户端
func (c *Connection) StartWriter() {
	fmt.Println("[Writer Goroutine is running]")
	defer fmt.Println(c.RemoteAddr().String(), "[conn Writer exit!]")

	for {
		select {
		case data, ok := <-c.MsgBufferChan:
			if ok {
				//有数据要写给客户端
				if _, err := c.Connection.Write(data); err != nil {
					fmt.Println("Send Buff Data error:, ", err, " Conn Writer exit")
					return
				}
			} else {
				fmt.Println("msgBuffChan is Closed")
				break
			}
		case <-c.ctx.Done():
			return
		}
	}
}

//StartReader 读消息Goroutine，用于从客户端中读取数据
func (c *Connection) StartReader() {
	fmt.Println("[Reader Goroutine is running]")
	defer fmt.Println(c.RemoteAddr().String(), "[conn Reader exit!]")
	defer c.Close()

	// 创建拆包解包的对象
	for {
		select {
		case <-c.ctx.Done():
			return
		default:

			//读取客户端的Msg head
			headData := make([]byte, c.TCPServer.Packet().GetHeadLen())
			if _, err := io.ReadFull(c.Connection, headData); err != nil {
				if errors.Is(err, io.EOF) {
					//nothing
					return
				} else if errors.Is(err, syscall.ECONNRESET) ||
					errors.Is(err, syscall.ECONNREFUSED) {
					//nothing
					return
				} else {
					fmt.Println("[WORKING] (Read) Packet data error: ", err)
					return
				}
			}
			//fmt.Printf("read headData %+v\n", headData)

			//拆包，得到msgID 和 datalen 放在msg中
			msg, err := c.TCPServer.Packet().Unpack(headData)
			if err != nil {
				fmt.Println("[WORKING] (Read) Packet unpack data error: ", err)
				return
			}

			//根据 dataLen 读取 data，放在msg.Data中
			var data []byte
			if msg.GetDataLen() > 0 {
				data = make([]byte, msg.GetDataLen())
				if _, err := io.ReadFull(c.Connection, data); err != nil {
					if errors.Is(err, io.EOF) {
						//nothing
					} else {
						fmt.Println("[WORKING] (Read) Message data error: ", err)
						return
					}
				}
			}
			msg.SetData(data)

			//得到当前客户端请求的Request数据
			req := Request{
				connection: c,
				msg:        msg,
				index:      0,
			}

			if c.WorkerPoolSize > 0 {
				//已经启动工作池机制，将消息交给Worker处理
				c.MsgHandler.SendMsgToTaskQueue(&req)
			} else {
				//从绑定好的消息和对应的处理方法中执行对应的Handle方法
				go c.MsgHandler.DoMsgHandler(&req)
			}
		}
	}
}

//Start 启动连接，让当前连接开始工作
func (c *Connection) Start() {
	c.ctx, c.cancel = context.WithCancel(context.Background())
	//按照用户传递进来的创建连接时需要处理的业务，执行钩子方法
	c.TCPServer.CallOnConnectionStart(c)
	//1 开启用户从客户端读取数据流程的Goroutine
	go c.StartReader()
	//2 开启用于写回客户端数据流程的Goroutine
	go c.StartWriter()

	select {
	case <-c.ctx.Done():
		c.finalizer()
		return
	}
}

//Stop 停止连接，结束当前连接状态M
func (c *Connection) Close() {
	c.cancel()
}

//GetTCPConnection 从当前连接获取原始的socket TCPConn
func (c *Connection) GetTCPConnection() *net.TCPConn {
	return c.Connection
}

//GetConnID 获取当前连接ID
func (c *Connection) GetConnectionID() uint32 {
	return c.ConnectionID
}

//RemoteAddr 获取远程客户端地址信息
func (c *Connection) RemoteAddr() net.Addr {
	return c.Connection.RemoteAddr()
}

//SendMsg 直接将Message数据发送数据给远程的TCP客户端
func (c *Connection) SendMsg(id uint32, data []byte) error {
	c.RLock()
	defer c.RUnlock()
	if c.isClosed == true {
		return errors.New("connection closed when send msg")
	}

	//将data封包，并且发送
	dp := c.TCPServer.Packet()
	msg, err := dp.Pack(zpack.NewMsgPackage(id, data))
	if err != nil {
		fmt.Println("Pack error msg ID = ", id)
		return errors.New("Pack error msg ")
	}

	//写回客户端
	_, err = c.Connection.Write(msg)
	return err
}

//SendBufferMsg  发生BufferMsg
func (c *Connection) SendBufferMsg(id uint32, data []byte) error {
	c.RLock()
	defer c.RUnlock()
	idleTimeout := time.NewTimer(5 * time.Millisecond)
	defer idleTimeout.Stop()

	if c.isClosed == true {
		return errors.New("Connection closed when send buff msg")
	}

	//将data封包，并且发送
	dp := c.TCPServer.Packet()
	msg, err := dp.Pack(zpack.NewMsgPackage(id, data))
	if err != nil {
		fmt.Println("Pack error msg ID = ", id)
		return errors.New("Pack error msg ")
	}

	// 发送超时
	select {
	case <-idleTimeout.C:
		return errors.New("send buff msg timeout")
	case c.MsgBufferChan <- msg:
		return nil
	}
	//写回客户端
	//c.msgBuffChan <- msg

	return nil
}

//SetProperty 设置链接属性
func (c *Connection) SetProperty(key string, value interface{}) {
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()
	if c.property == nil {
		c.property = make(map[string]interface{})
	}

	c.property[key] = value
}

//GetProperty 获取链接属性
func (c *Connection) GetProperty(key string) (interface{}, error) {
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()

	if value, ok := c.property[key]; ok {
		return value, nil
	}

	return nil, errors.New("no property found")
}

//RemoveProperty 移除链接属性
func (c *Connection) RemoveProperty(key string) {
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()

	delete(c.property, key)
}

//返回ctx，用于用户自定义的go程获取连接退出状态
func (c *Connection) Context() context.Context {
	return c.ctx
}

func (c *Connection) finalizer() {
	//如果用户注册了该链接的关闭回调业务，那么在此刻应该显示调用
	c.TCPServer.CallOnConnectionStop(c)

	c.Lock()
	defer c.Unlock()

	//如果当前链接已经关闭
	if c.isClosed == true {
		return
	}

	fmt.Println("Connection Stop()...ConnID = ", c.ConnectionID)

	// 关闭socket链接
	_ = c.Connection.Close()

	//将链接从连接管理器中删除
	c.TCPServer.GetConnectionManager().Remove(c)

	//关闭该链接全部管道
	close(c.MsgBufferChan)
	//设置标志位
	c.isClosed = true
}

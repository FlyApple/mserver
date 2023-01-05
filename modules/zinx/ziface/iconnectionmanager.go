// Package ziface 主要提供zinx全部抽象层接口定义.
// 包括:
//		IServer 服务mod接口
//		IRouter 路由mod接口
//		IConnection 连接mod层接口
//      IMessage 消息mod接口
//		IDataPack 消息拆解接口
//      IMsgHandler 消息处理及协程池接口
//
// 当前文件描述:
// @Title  iconnmanager.go
// @Description    连接管理相关,包括添加、删除、通过一个连接ID获得连接对象，当前连接数量、清空全部连接等方法
// @Author  Aceld - Thu Mar 11 10:32:29 CST 2019
package ziface

/*
	连接管理抽象层
*/
type IConnectionManager interface {
	Add(connection IConnection)         //添加链接
	Remove(connection IConnection)      //删除连接
	Get(id uint32) (IConnection, error) //利用ConnectionID获取链接
	MaxLen() int
	Len() int  //获取当前连接
	ClearAll() //删除并停止所有链接
	ClearOne(id uint32)
}

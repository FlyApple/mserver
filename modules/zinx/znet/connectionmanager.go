package znet

import (
	"errors"
	"fmt"
	"sync"

	"mcmcx.com/mserver/modules/zinx/ziface"
)

//ConnectionManager 连接管理模块
type ConnectionManager struct {
	connections_maxnum int32
	connections        map[uint32]ziface.IConnection
	connections_lock   sync.RWMutex
}

//NewConnManager 创建一个链接管理
func NewConnectionManager(maxnum int32) *ConnectionManager {
	return &ConnectionManager{
		connections_maxnum: maxnum,
		connections:        make(map[uint32]ziface.IConnection),
	}
}

//Add 添加链接
func (m *ConnectionManager) Add(connection ziface.IConnection) {

	m.connections_lock.Lock()
	//将connection连接添加到ConnectionMananger中
	m.connections[connection.GetConnectionID()] = connection
	m.connections_lock.Unlock()

	fmt.Println("connection add to ConnectionManager successfully: connection num = ", m.Len())
}

//Remove 删除连接
func (m *ConnectionManager) Remove(connection ziface.IConnection) {

	m.connections_lock.Lock()
	//删除连接信息
	delete(m.connections, connection.GetConnectionID())
	m.connections_lock.Unlock()
	fmt.Println("connection Remove ConnectionID=", connection.GetConnectionID(), " successfully: connection num = ", m.Len())
}

//Get 利用ID获取链接
func (m *ConnectionManager) Get(id uint32) (ziface.IConnection, error) {
	m.connections_lock.RLock()
	defer m.connections_lock.RUnlock()

	if conn, ok := m.connections[id]; ok {
		return conn, nil
	}

	return nil, errors.New("connection not found")

}

//
func (m *ConnectionManager) MaxLen() int {
	return int(m.connections_maxnum)
}

//Len 获取当前连接
func (m *ConnectionManager) Len() int {
	m.connections_lock.RLock()
	length := len(m.connections)
	m.connections_lock.RUnlock()
	return length
}

//Clear 清除并停止所有连接
func (m *ConnectionManager) ClearAll() {
	m.connections_lock.Lock()

	//停止并删除全部的连接信息
	for _, conn := range m.connections {
		//停止
		conn.Close()
	}
	m.connections_lock.Unlock()

	fmt.Println("Clear All Connections successfully: connection num = ", m.Len())
}

//ClearOneConnection  利用ID获取一个链接 并且删除
func (m *ConnectionManager) ClearOne(id uint32) {
	m.connections_lock.Lock()
	defer m.connections_lock.Unlock()

	connections := m.connections
	if conn, ok := connections[id]; ok {
		//停止
		conn.Close()

		fmt.Println("Clear Connections ID:  ", id, " succeed")
		return
	}

	fmt.Println("Clear Connections ID:  ", id, " error")
}

package zpack

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

const MESSAGE_TEXT_MAXLEN = 0xFFFF
const MESSAGE_BYTES_MAXLEN = 0xFFFF

//Message 消息
type Message struct {
	DataLen uint32 //消息的长度
	ID      uint32 //消息的ID
	Data    []byte //消息的内容
}

//
type MessageBuffer struct {
	bytes.Buffer
}

func NewMessageBuffer(data []byte) *MessageBuffer {
	var buffer MessageBuffer
	if data != nil {
		_, err := buffer.Write(data)
		if err != nil {
			return nil
		}
	}
	return &buffer
}

//
func (mb *MessageBuffer) Length() int {
	return mb.Len()
}

func (mb *MessageBuffer) Data() []byte {
	return mb.Bytes()
}

// 2bytes
func (mb *MessageBuffer) WriteInt16(n int16) {
	a := (n >> 0x00) & 0xFF
	b := (n >> 0x08) & 0xFF
	mb.WriteByte(byte(a))
	mb.WriteByte(byte(b))
}

func (mb *MessageBuffer) WriteUInt16(n uint16) {
	a := (n >> 0x00) & 0xFF
	b := (n >> 0x08) & 0xFF
	mb.WriteByte(byte(a))
	mb.WriteByte(byte(b))
}

func (mb *MessageBuffer) ReadInt16() int16 {
	a, _ := mb.ReadByte()
	b, _ := mb.ReadByte()
	return int16(b)<<0x08 | int16(a)<<0x00
}

func (mb *MessageBuffer) ReadUInt16() uint16 {
	a, _ := mb.ReadByte()
	b, _ := mb.ReadByte()
	return uint16(b)<<0x08 | uint16(a)<<0x00
}

// 4 bytes
// ABCD
// DCBA
func (mb *MessageBuffer) WriteInt32(n int32) {
	a := (n >> 0x00) & 0xFF
	b := (n >> 0x08) & 0xFF
	c := (n >> 0x10) & 0xFF
	d := (n >> 0x18) & 0xFF
	mb.WriteByte(byte(a))
	mb.WriteByte(byte(b))
	mb.WriteByte(byte(c))
	mb.WriteByte(byte(d))
}

func (mb *MessageBuffer) WriteUInt32(n uint32) {
	a := (n >> 0x00) & 0xFF
	b := (n >> 0x08) & 0xFF
	c := (n >> 0x10) & 0xFF
	d := (n >> 0x18) & 0xFF
	mb.WriteByte(byte(a))
	mb.WriteByte(byte(b))
	mb.WriteByte(byte(c))
	mb.WriteByte(byte(d))
}

func (mb *MessageBuffer) ReadInt32() int32 {
	a, _ := mb.ReadByte()
	b, _ := mb.ReadByte()
	c, _ := mb.ReadByte()
	d, _ := mb.ReadByte()
	return int32(d)<<0x18 | int32(c)<<0x10 | int32(b)<<0x08 | int32(a)<<0x00
}

func (mb *MessageBuffer) ReadUInt32() uint32 {
	a, _ := mb.ReadByte()
	b, _ := mb.ReadByte()
	c, _ := mb.ReadByte()
	d, _ := mb.ReadByte()
	return uint32(d)<<0x18 | uint32(c)<<0x10 | uint32(b)<<0x08 | uint32(a)<<0x00
}

func (mb *MessageBuffer) WriteInt64(n uint64) {
	binary.Write(mb, binary.LittleEndian, n)
}

func (mb *MessageBuffer) WriteUInt64(n uint64) {
	binary.Write(mb, binary.LittleEndian, n)
}

func (mb *MessageBuffer) ReadInt64() int64 {
	var value int64 = 0
	err := binary.Read(mb, binary.LittleEndian, &value)
	if err != nil {
		return 0
	}
	return value
}

func (mb *MessageBuffer) ReadUInt64() uint64 {
	var value uint64 = 0
	err := binary.Read(mb, binary.LittleEndian, &value)
	if err != nil {
		return 0
	}
	return value
}

func (mb *MessageBuffer) WriteFloat32(n float32) {
	binary.Write(mb, binary.LittleEndian, n)
}

func (mb *MessageBuffer) WriteFloat64(n float64) {
	binary.Write(mb, binary.LittleEndian, n)
}

func (mb *MessageBuffer) ReadFloat32() float32 {
	var value float32 = 0.0
	err := binary.Read(mb, binary.LittleEndian, &value)
	if err != nil {
		return 0.0
	}
	return value
}

func (mb *MessageBuffer) ReadFloat64() float64 {
	var value float64 = 0.0
	err := binary.Read(mb, binary.LittleEndian, &value)
	if err != nil {
		return 0.0
	}
	return value
}

func (mb *MessageBuffer) WriteStringU(s string) int {
	buffer := bytes.NewBufferString(s)
	if buffer.Len() >= MESSAGE_TEXT_MAXLEN {
		return 0
	}
	n, _ := mb.Write(buffer.Bytes())
	return n
}

func (mb *MessageBuffer) WriteStringL(s string) int {
	buffer := bytes.NewBufferString(s)
	if buffer.Len() >= MESSAGE_TEXT_MAXLEN {
		return 0
	}
	mb.WriteUInt16(uint16(buffer.Len()))
	n, _ := mb.Write(buffer.Bytes())
	return n
}

func (mb *MessageBuffer) ReadStringU() string {
	s, err := mb.ReadString(0x00)
	if err != nil {
		return ""
	}
	return s
}

func (mb *MessageBuffer) ReadStringL() string {
	len := mb.ReadUInt16()
	if len >= MESSAGE_TEXT_MAXLEN {
		return ""
	}

	data := make([]byte, len)
	_, err := mb.Read(data)
	if err != nil {
		return ""
	}

	buffer := bytes.NewBuffer(data)
	text, err := buffer.ReadString(0x00)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			return ""
		}
	}
	return text
}

func (mb *MessageBuffer) WriteBytesL(vb []byte) int {
	buffer := bytes.NewBuffer(vb)
	if buffer.Len() >= MESSAGE_BYTES_MAXLEN {
		return 0
	}

	mb.WriteUInt16(uint16(buffer.Len()))
	n, _ := mb.Write(buffer.Bytes())
	return n
}

func (mb *MessageBuffer) ReadBytesL() []byte {
	len := mb.ReadUInt16()
	if len >= MESSAGE_BYTES_MAXLEN {
		return nil
	}

	data := make([]byte, len)
	_, err := mb.Read(data)
	if err != nil {
		return nil
	}
	return data
}

//NewMsgPackage 创建一个Message消息包
func NewMsgPackage(ID uint32, data []byte) *Message {
	return &Message{
		DataLen: uint32(len(data)),
		ID:      ID,
		Data:    data,
	}
}

func (msg *Message) Init(ID uint32, data []byte) {
	msg.ID = ID
	msg.Data = data
	msg.DataLen = uint32(len(data))
}

//GetDataLen 获取消息数据段长度
func (msg *Message) GetDataLen() uint32 {
	return msg.DataLen
}

//GetMsgID 获取消息ID
func (msg *Message) GetMsgID() uint32 {
	return msg.ID
}

//GetData 获取消息内容
func (msg *Message) GetData() []byte {
	return msg.Data
}

//SetDataLen 设置消息数据段长度
func (msg *Message) SetDataLen(len uint32) {
	msg.DataLen = len
}

//SetMsgID 设计消息ID
func (msg *Message) SetMsgID(msgID uint32) {
	msg.ID = msgID
}

//SetData 设计消息内容
func (msg *Message) SetData(data []byte) {
	msg.Data = data
}

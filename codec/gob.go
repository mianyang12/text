package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

type GobCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
}

var _ Codec = (*GobCodec)(nil)

// NewGobCodec 函数创建一个新的 Codec 实例，用于处理 gob 编码的数据
func NewGobCodec(conn io.ReadWriteCloser) Codec {
	// 创建一个 bufio.NewWriter，用于缓冲数据
	buf := bufio.NewWriter(conn)
	// 返回一个新的 GobCodec 实例，包含了连接、写入器、解码器和编码器
	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(buf),
	}
}

// ReadHeader(*Header) error：读取并解码消息的头部
func (c *GobCodec) ReadHeader(h *Header) error {
	return c.dec.Decode(h)
}

// ReadBody(interface{}) error：读取并解码消息的主体
func (c *GobCodec) ReadBody(body interface{}) error {
	return c.dec.Decode(body)
}

// Write(*Header, interface{}) error：写入编码的消息
func (c *GobCodec) Write(h *Header, body interface{}) (err error) {
	defer func() {
		_ = c.buf.Flush()
		if err != nil {
			_ = c.Close()
		}
	}()
	if err = c.enc.Encode(h); err != nil {
		log.Println("rpc: gob error encoding header:", err)
		return
	}
	if err = c.enc.Encode(body); err != nil {
		log.Println("rpc: gob error encoding body:", err)
		return
	}
	return
}

// Close() error：关闭底层的网络连接。
func (c *GobCodec) Close() error {
	return c.conn.Close()
}

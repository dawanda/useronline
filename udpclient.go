package main

import (
	"fmt"
	"net"
)

type UdpClient struct {
	Conn *net.UDPConn
}

func NewUdpClient(target string) (*UdpClient, error) {
	addr, err := net.ResolveUDPAddr("udp", target)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}

	return &UdpClient{Conn: conn}, nil
}

func (c *UdpClient) Close() {
	if c != nil && c.Conn != nil {
		c.Conn.Close()
	}
}

func (c *UdpClient) Sendf(format string, args ...interface{}) (int, error) {
	return c.Send([]byte(fmt.Sprintf(format, args...)))
}

func (c *UdpClient) Send(buf []byte) (int, error) {
	n, err := c.Conn.Write(buf)
	return n, err
}

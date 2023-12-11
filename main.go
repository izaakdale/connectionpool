package main

import (
	"net"
	"time"
)

const (
	N_CONNS  = 1
	CONN_TTL = 3 * time.Second
)

func main() {
	pool, err := NewConnectionPool(N_CONNS, CONN_TTL)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			conn := pool.WaitForConnection()
			conn.Write([]byte("hello from goroutine\n"))
			time.Sleep(time.Second)
			pool.ReleaseConnection(conn)
		}
	}()

	for {
		conn := pool.WaitForConnection()
		conn.Write([]byte("hello from main\n"))
		time.Sleep(time.Second)
		pool.ReleaseConnection(conn)
	}
}

type Connection struct {
	ID         int
	tcp        net.Conn
	lastActive time.Time
}

func (c *Connection) Write(b []byte) (n int, err error) {
	c.lastActive = time.Now()
	return c.tcp.Write(b)
}

type ConnectionPool struct {
	ConnectionTTL time.Duration
	Connections   chan *Connection
}

func NewConnectionPool(nConns int, connTTL time.Duration) (*ConnectionPool, error) {
	pool := ConnectionPool{
		Connections:   make(chan *Connection, nConns),
		ConnectionTTL: connTTL,
	}

	for i := 0; i < nConns; i++ {
		err := pool.NewConnection(i)
		if err != nil {
			return nil, err
		}
	}
	return &pool, nil
}

func (p *ConnectionPool) NewConnection(id int) error {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		return err
	}
	p.Connections <- &Connection{id, conn, time.Now()}
	return nil
}

func (p *ConnectionPool) WaitForConnection() *Connection {
	conn := <-p.Connections
	if time.Since(conn.lastActive) > p.ConnectionTTL {
		p.NewConnection(conn.ID)
		conn = <-p.Connections
	}
	return conn
}

func (p *ConnectionPool) ReleaseConnection(conn *Connection) {
	p.Connections <- conn
}

package main

import (
	"log"
	"net"
	"time"
)

const (
	N_CONNS  = 3
	CONN_TTL = 3 * time.Second
)

func main() {
	pool, err := NewConnectionPool("tcp", "localhost:8080", N_CONNS, CONN_TTL)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			conn := pool.WaitForConnection()
			_, err := conn.Write([]byte("hello from goroutine\n"))
			if err != nil {
				log.Fatal(err)
			}
			time.Sleep(3 * time.Second)
			pool.ReleaseConnection(conn)
		}
	}()

	for {
		conn := pool.WaitForConnection()
		_, err := conn.Write([]byte("hello from main\n"))
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(time.Second)
		pool.ReleaseConnection(conn)
	}
}

type Connection struct {
	ID         int
	network    string
	address    string
	netStream  net.Conn
	lastActive time.Time
}

func (c *Connection) Write(b []byte) (n int, err error) {
	c.lastActive = time.Now()
	return c.netStream.Write(b)
}

func (c *Connection) Read(b []byte) (n int, err error) {
	c.lastActive = time.Now()
	return c.netStream.Read(b)
}

func (c *Connection) Close() error {
	c.lastActive = time.Now()
	return c.netStream.Close()
}

type ConnectionPool struct {
	ConnectionTTL time.Duration
	Connections   chan *Connection
}

func NewConnectionPool(network string, address string, nConns int, connTTL time.Duration) (*ConnectionPool, error) {
	pool := ConnectionPool{
		Connections:   make(chan *Connection, nConns),
		ConnectionTTL: connTTL,
	}

	for i := 0; i < nConns; i++ {
		err := pool.NewConnection(i, network, address)
		if err != nil {
			return nil, err
		}
	}
	return &pool, nil
}

func (p *ConnectionPool) NewConnection(id int, network string, address string) error {
	conn, err := net.Dial(network, address)
	if err != nil {
		return err
	}
	p.Connections <- &Connection{id, network, address, conn, time.Now()}
	return nil
}

func (p *ConnectionPool) WaitForConnection() *Connection {
	conn := <-p.Connections
	if time.Since(conn.lastActive) > p.ConnectionTTL {
		p.NewConnection(conn.ID, conn.network, conn.address)
		conn = <-p.Connections
	}
	return conn
}

func (p *ConnectionPool) ReleaseConnection(conn *Connection) {
	p.Connections <- conn
}

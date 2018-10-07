package pool

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

// blockingPool implements the Pool interface.
type blockingPool struct {
	address string

	//storage for net.Conn connections
	conns *blockingDeque

	factory Factory

	Config

	//mutex is to make closing the pool and recycling the connection an atomic operation
	sync.RWMutex
}

//Factory is a function to create new connections
//which is provided by the user
type Factory func(address string, dialTimeout time.Duration) (net.Conn, error)

// defaultFactory is base tcp dialer. If user not specify dialer function. this default will be used
var defaultFactory = func(address string, dialTimeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("tcp", address, dialTimeout)
}

//Create a new blocking pool. As no new connections would be made when the pool is busy,
//the number of connections of the pool is kept no more than initCap and maxCap does not
//make sense but the api is reserved. The timeout to block Get() is set to 3 by default
//concerning that it is better to be related with Get() method.
func NewBlockingPool(address string, c Config, factory Factory) (Pool, error) {
	if c.InitCap > c.MaxCap {
		return nil, errors.New("invalid capacity settings")
	}
	c, _ = defaultConfig.Merge(c)

	p := &blockingPool{
		address: address,
		Config:  c,
		factory: factory,
		conns:   newBlockingDeque(c.MaxCap, c.MaxCap-c.InitCap),
	}
	if p.factory == nil {
		p.factory = defaultFactory
	}

	for i := 0; i < c.InitCap; i++ {
		conn, err := p.NewWrapConn()
		if err != nil {
			return nil, fmt.Errorf("create connection fail: %s", err)
		}
		p.conns.prepend(conn)
	}
	go p.checkIdleConn()
	return p, nil
}

// checkIdleConn periodically check and close idle net connections in pool
func (p *blockingPool) checkIdleConn() {
	for {
		time.Sleep(3 * time.Second)
		p.RLock()
		// walk and check whether a wrapped connection is idle.
		p.conns.Walk(func(item interface{}) {
			c := (item).(*WrappedConn)
			c.checkIdle()
		})
		p.RUnlock()
	}
}

func (p *blockingPool) config() Config {
	p.RLock()
	defer p.RUnlock()

	return p.Config
}

//Get blocks for an available connection.
func (p *blockingPool) Get() (*WrappedConn, error) {
	p.RLock()
	defer p.RUnlock()

	if p.conns == nil {
		return nil, ErrClosed
	}
	// new connection is popped from head of Deque
	item, err := p.conns.pop(p.WaitTimeout)
	if err != nil {
		return nil, err
	}
	conn, ok := item.(*WrappedConn)
	if !ok || conn.unusable {
		if conn, err = p.NewWrapConn(); err != nil {
			return nil, err
		}
	}
	return conn, nil
}

func (p *blockingPool) put(conn *WrappedConn) error {
	p.RLock()
	defer p.RUnlock()

	if p.Config.CacheMethod == FIFO {
		return p.putBottom(conn)
	} else {
		return p.putTop(conn)
	}
}

//put puts the connection back to the pool. If the pool is closed, put simply close
//any connections received and return immediately. A nil net.Conn is illegal and will be rejected.
func (p *blockingPool) putTop(conn *WrappedConn) error {
	p.RLock()
	defer p.RUnlock()

	//in case that pool is closed and pool.conns is set to nil
	if p.conns == nil {
		conn.destroy()
		return ErrClosed
	}
	//else append conn to tail of Deque
	p.conns.prepend(conn)
	return nil
}

//put puts the connection back to the pool. If the pool is closed, put simply close
//any connections received and return immediately. A nil net.Conn is illegal and will be rejected.
func (p *blockingPool) putBottom(conn *WrappedConn) error {
	p.RLock()
	defer p.RUnlock()

	//in case that pool is closed and pool.conns is set to nil
	if p.conns == nil {
		conn.destroy()
		return ErrClosed
	}
	//else append conn to tail of Deque
	p.conns.append(conn)
	return nil
}

//TODO
//Close set connection channel to nil and close all the relative connections.
//Yet not implemented.
func (p *blockingPool) Close() {}

//TODO
//Len return the number of current active(in use or available) connections.
func (p *blockingPool) Len() int {
	return 0
}

// Update
func (p *blockingPool) Update(c Config) {
	c, changed := p.Config.Merge(c)
	if changed {
		p.Lock()
		defer p.Unlock()
		if c.MaxCap != p.MaxCap {
			p.conns = newBlockingDeque(c.MaxCap, c.MaxCap-p.MaxCap)
		}
		p.Config = c
	}
}

package pool

import (
	"net"
	"time"
)

//wrappedConn modify the behavior of net.Conn's Write() method and Close() method
//while other methods can be accessed transparently.
type WrappedConn struct {
	net.Conn
	pool       *blockingPool
	unusable   bool
	lastAccess time.Time
}

//TODO
func (c *WrappedConn) Close() error {
	if c.unusable {
		c.destroy()
		c.pool.putBottom(nil)
	} else {
		if err := c.pool.put(c); err != nil {
			c.destroy()
		}
	}

	return nil
}

// getInactiveNetConn
func (c *WrappedConn) checkIdle() bool {
	if time.Since(c.lastAccess) > c.pool.config().IdleTimeout {
		c.unusable = true
		c.destroy()
		return false
	}
	return true
}

// destory
func (c *WrappedConn) destroy() {
	c.Conn.Close()
}

func (c *WrappedConn) MarkUnusable() {
	c.unusable = true
}

//Write checkout the error returned from the origin Write() method.
//If the error is not nil, the connection is marked as unusable.
func (c *WrappedConn) Write(b []byte) (n int, err error) {
	//c.Conn is certainly not nil
	n, err = c.Conn.Write(b)
	if err != nil {
		c.unusable = true
	} else {
		c.lastAccess = time.Now()
	}
	return
}

//Read works the same as Write.
func (c *WrappedConn) Read(b []byte) (n int, err error) {
	//c.Conn is certainly not nil
	n, err = c.Conn.Read(b)
	if err != nil {
		c.unusable = true
	} else {
		c.lastAccess = time.Now()
	}

	return
}

//wrap wraps net.Conn and start a delayClose goroutine
func (p *blockingPool) NewWrapConn() (*WrappedConn, error) {
	p.RLock()
	defer p.RUnlock()

	conn, err := p.factory(p.address, p.DialTimeout)
	if err != nil {
		return nil, err
	}
	c := &WrappedConn{
		Conn:       conn,
		pool:       p,
		lastAccess: time.Now(),
	}
	return c, nil
}

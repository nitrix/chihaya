// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package redis implements the storage interface for a BitTorrent tracker.
package redis

import (
	"time"

	"github.com/garyburd/redigo/redis"

	"github.com/pushrax/chihaya/config"
	"github.com/pushrax/chihaya/storage"
)

type driver struct{}

func (d *driver) New(conf *config.Storage) storage.DS {
	return &DS{
		conf: conf,
		Pool: redis.Pool{
			MaxIdle:      conf.MaxIdleConn,
			IdleTimeout:  conf.IdleTimeout.Duration,
			Dial:         makeDialFunc(conf),
			TestOnBorrow: testOnBorrow,
		},
	}
}

func makeDialFunc(conf *config.Storage) func() (redis.Conn, error) {
	return func() (redis.Conn, error) {
		var (
			conn redis.Conn
			err  error
		)

		if conf.ConnTimeout != nil {
			conn, err = redis.DialTimeout(
				conf.Network,
				conf.Addr,
				conf.ConnTimeout.Duration, // Connect Timeout
				conf.ConnTimeout.Duration, // Read Timeout
				conf.ConnTimeout.Duration, // Write Timeout
			)
		} else {
			conn, err = redis.Dial(conf.Network, conf.Addr)
		}
		if err != nil {
			return nil, err
		}
		return conn, nil
	}
}

func testOnBorrow(c redis.Conn, t time.Time) error {
	_, err := c.Do("PING")
	return err
}

type DS struct {
	conf *config.Storage
	redis.Pool
}

func (ds *DS) FindUser(passkey string) (*storage.User, bool, error) {
	conn := ds.Get()
	defer conn.Close()

	key := ds.conf.Prefix + "User:" + passkey
	reply, err := redis.Values(conn.Do("HGETALL", key))
	if err != nil {
		return nil, true, err
	}

	// If we get nothing back, the user isn't found.
	if len(reply) == 0 {
		return nil, false, nil
	}

	user := &storage.User{}
	err = redis.ScanStruct(reply, user)
	if err != nil {
		return nil, true, err
	}
	return user, true, nil
}

func (ds *DS) FindTorrent(infohash string) (*storage.Torrent, bool, error) {
	conn := ds.Get()
	defer conn.Close()

	key := ds.conf.Prefix + "Torrent:" + infohash
	reply, err := redis.Values(conn.Do("HGETALL", key))
	if err != nil {
		return nil, false, err
	}

	// If we get nothing back, the torrent isn't found.
	if len(reply) == 0 {
		return nil, false, nil
	}

	torrent := &storage.Torrent{}
	err = redis.ScanStruct(reply, torrent)
	if err != nil {
		return nil, true, err
	}
	return torrent, true, nil
}

func (ds *DS) ClientWhitelisted(peerID string) (bool, error) {
	conn := ds.Get()
	defer conn.Close()

	key := ds.conf.Prefix + "Whitelist:" + peerID
	exists, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		return false, err
	}
	return exists, nil
}

type Tx struct {
	conf *config.Storage
	done bool
	redis.Conn
}

func (ds *DS) Begin() (storage.Tx, error) {
	conn := ds.Get()
	err := conn.Send("MULTI")
	if err != nil {
		return nil, err
	}
	return &Tx{
		conf: ds.conf,
		Conn: conn,
	}, nil
}

func (tx *Tx) close() {
	if tx.done {
		panic("redis: transaction closed twice")
	}
	tx.done = true
	tx.Conn.Close()
}

func (tx *Tx) Commit() error {
	if tx.done {
		return storage.ErrTxDone
	}
	_, err := tx.Do("EXEC")
	if err != nil {
		return err
	}

	tx.close()
	return nil
}

// Redis doesn't need to rollback. Exec is atomic.
func (tx *Tx) Rollback() error {
	if tx.done {
		return storage.ErrTxDone
	}
	tx.close()
	return nil
}

func (tx *Tx) Snatch(user *storage.User, torrent *storage.Torrent) error {
	if tx.done {
		return storage.ErrTxDone
	}
	// TODO
	return nil
}

func (tx *Tx) Unprune(t *storage.Torrent) error {
	if tx.done {
		return storage.ErrTxDone
	}
	key := tx.conf.Prefix + "Torrent:" + t.Infohash
	err := tx.Send("HSET " + key + " Status 0")
	if err != nil {
		return err
	}
	return nil
}

func (tx *Tx) NewLeecher(t *storage.Torrent, p *storage.Peer) error {
	if tx.done {
		return storage.ErrTxDone
	}
	// TODO
	return nil
}

func (tx *Tx) RmLeecher(t *storage.Torrent, p *storage.Peer) error {
	if tx.done {
		return storage.ErrTxDone
	}
	// TODO
	return nil
}

func (tx *Tx) NewSeeder(t *storage.Torrent, p *storage.Peer) error {
	if tx.done {
		return storage.ErrTxDone
	}
	// TODO
	return nil
}

func (tx *Tx) RmSeeder(t *storage.Torrent, p *storage.Peer) error {
	if tx.done {
		return storage.ErrTxDone
	}
	// TODO
	return nil
}

func (tx *Tx) IncrementSlots(u *storage.User) error {
	if tx.done {
		return storage.ErrTxDone
	}
	// TODO
	return nil
}

func (tx *Tx) DecrementSlots(u *storage.User) error {
	if tx.done {
		return storage.ErrTxDone
	}
	// TODO
	return nil
}

func init() {
	storage.Register("redis", &driver{})
}

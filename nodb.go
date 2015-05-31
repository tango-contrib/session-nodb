// Copyright 2013 Beego Authors
// Copyright 2014 Unknwon
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package nodbstore

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"
	"time"
	"unsafe"

	"github.com/lunny/log"
	"github.com/lunny/nodb"
	"github.com/lunny/nodb/config"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/lunny/tango"

	"github.com/tango-contrib/session"
)

var _ session.Store = &NodbStore{}

type Options struct {
	Path    string
	DbIndex int
	MaxAge  time.Duration
}

// RedisStore represents a redis session store implementation.
type NodbStore struct {
	Options
	tango.Logger
	db *nodb.DB
}

func preOptions(opts []Options) Options {
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.Path == "" {
		opt.Path = "./nodbstore"
	}
	if opt.MaxAge == 0 {
		opt.MaxAge = session.DefaultMaxAge
	}
	return opt
}

// NewRedisStore creates and returns a redis session store.
func New(opts ...Options) (*NodbStore, error) {
	opt := preOptions(opts)
	cfg := config.NewConfigDefault()
	cfg.DataDir = opt.Path

	ndb, err := nodb.Open(cfg)
	if err != nil {
		return nil, err
	}
	db, err := ndb.Select(opt.DbIndex)
	if err != nil {
		return nil, err
	}

	return &NodbStore{
		Options: opt,
		db:      db,
		Logger:  log.Std,
	}, nil
}

func (c *NodbStore) serialize(value interface{}) ([]byte, error) {
	err := c.registerGobConcreteType(value)
	if err != nil {
		return nil, err
	}

	if reflect.TypeOf(value).Kind() == reflect.Struct {
		return nil, fmt.Errorf("serialize func only take pointer of a struct")
	}

	var b bytes.Buffer
	encoder := gob.NewEncoder(&b)

	err = encoder.Encode(&value)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (c *NodbStore) deserialize(byt []byte) (ptr interface{}, err error) {
	b := bytes.NewBuffer(byt)
	decoder := gob.NewDecoder(b)

	var p interface{}
	err = decoder.Decode(&p)
	if err != nil {
		return
	}

	v := reflect.ValueOf(p)
	if v.Kind() == reflect.Struct {
		var pp interface{} = &p
		datas := reflect.ValueOf(pp).Elem().InterfaceData()

		sp := reflect.NewAt(v.Type(),
			unsafe.Pointer(datas[1])).Interface()
		ptr = sp
	} else {
		ptr = p
	}
	return
}

func (c *NodbStore) registerGobConcreteType(value interface{}) error {
	t := reflect.TypeOf(value)

	switch t.Kind() {
	case reflect.Ptr:
		v := reflect.ValueOf(value)
		i := v.Elem().Interface()
		gob.Register(i)
	case reflect.Struct, reflect.Map, reflect.Slice:
		gob.Register(value)
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Bool, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		// do nothing since already registered known type
	default:
		return fmt.Errorf("unhandled type: %v", t)
	}
	return nil
}

// Set sets value to given key in session.
func (s *NodbStore) Set(id session.Id, key string, val interface{}) error {
	bs, err := s.serialize(val)
	if err != nil {
		return err
	}
	_, err = s.db.HSet([]byte(id), []byte(key), bs)
	if err == nil {
		// when write data, reset maxage
		_, err = s.db.Expire([]byte(id), int64(s.MaxAge))
	}
	return err
}

// Get gets value by given key in session.
func (s *NodbStore) Get(id session.Id, key string) interface{} {
	val, err := s.db.HGet([]byte(id), []byte(key))
	// if not exist
	if err == leveldb.ErrNotFound {
		return nil
	}

	if err != nil {
		s.Logger.Errorf("nodb HGET failed: %s", err)
		return nil
	}

	// when read data, reset maxage
	s.db.Expire([]byte(id), int64(s.MaxAge))

	if len(val) == 0 {
		return nil
	}

	value, err := s.deserialize(val)
	if err != nil {
		s.Logger.Errorf("nodb HGET failed: %v - %v", err, val)
		return nil
	}
	return value
}

// Delete delete a key from session.
func (s *NodbStore) Del(id session.Id, key string) bool {
	_, err := s.db.HDel([]byte(id), []byte(key))
	return err == nil
}

func (s *NodbStore) Clear(id session.Id) bool {
	_, err := s.db.Del([]byte(id))
	return err == nil
}

func (s *NodbStore) Add(id session.Id) bool {
	return true
}

func (s *NodbStore) Exist(id session.Id) bool {
	b, _ := s.db.HLen([]byte(id))
	return b > 0
}

func (s *NodbStore) SetMaxAge(maxAge time.Duration) {
	s.MaxAge = maxAge
}

func (s *NodbStore) SetIdMaxAge(id session.Id, maxAge time.Duration) {
	if s.Exist(id) {
		s.db.Expire([]byte(id), int64(s.MaxAge))
	}
}

func (s *NodbStore) Run() error {
	return nil
}

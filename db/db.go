// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package db

import bolt "go.etcd.io/bbolt"

type DB struct {
	DB *bolt.DB
}

func (d DB) Initialize() error {
	return d.DB.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte("users"))
		return nil
	})
}

func (d DB) GetUser(name string) (*User, error) {
	result := &User{
		Name: name,
	}
	err := d.DB.View(func(tx *bolt.Tx) error {
		users := tx.Bucket([]byte("users"))
		password := users.Get([]byte(name))
		if password == nil {
			return nil
		}
		// bolt byte slices are invalid outside of transaction, copying
		result.Password = make([]byte, len(password))
		copy(result.Password, password)
		return nil
	})
	return result, err
}

func (d DB) UpsertUser(user User) error {
	return d.DB.Update(func(tx *bolt.Tx) error {
		users := tx.Bucket([]byte("users"))
		users.Put([]byte(user.Name), []byte(user.Password))
		return nil
	})
}

func (d DB) DeleteUser(name string) error {
	return d.DB.Update(func(tx *bolt.Tx) error {
		users := tx.Bucket([]byte("users"))
		users.Delete([]byte(name))
		return nil
	})
}

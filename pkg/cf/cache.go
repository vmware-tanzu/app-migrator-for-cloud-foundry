/*
 *  Copyright 2022 VMware, Inc.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *  http://www.apache.org/licenses/LICENSE-2.0
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package cf

import "sync"

type Cache struct {
	sync.RWMutex
	internal map[string]interface{}
}

func NewCache() *Cache {
	return &Cache{
		internal: make(map[string]interface{}),
	}
}

// Load retrieves the item from the cache if exists.
// Returns the val and true if it exists otherwise false.
func (c *Cache) Load(key string) (value interface{}, ok bool) {
	c.RLock()
	value, ok = c.internal[key]
	c.RUnlock()
	return
}

// LoadOrStore atomically returns the already stored object if found and true, otherwise
// it stores the given object, returns val and false
func (c *Cache) LoadOrStore(key string, val interface{}) (actual interface{}, loaded bool) {
	c.Lock()
	// ensure all goroutines use the same val instance
	actual, ok := c.internal[key]
	if ok {
		return actual, true
	}
	actual = val
	c.internal[key] = actual
	c.Unlock()
	return actual, false
}

// Store atomically stores the given object regardless if it exists
func (c *Cache) Store(key string, val interface{}) {
	c.Lock()
	c.internal[key] = val
	c.Unlock()
}

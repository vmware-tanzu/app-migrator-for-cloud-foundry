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

package cache

import (
	"fmt"
)

// notFound follows the opaque error pattern
// https://dave.cheney.net/2016/04/27/dont-just-check-errors-handle-them-gracefully
type notFound interface {
	NotFound() bool
}

func IsNotFound(err error) bool {
	te, ok := err.(notFound)
	return ok && te.NotFound()
}

type AppNotFoundError struct {
	AppName string
	Space   string
	Count   int
}

func (e *AppNotFoundError) Error() string {
	return fmt.Sprintf("Expected to find one app named %s in space %s, but found %d", e.AppName, e.Space, e.Count)
}

func (e *AppNotFoundError) NotFound() bool {
	return true
}

type StackNotFoundError struct {
	StackName string
	Count     int
}

func (e *StackNotFoundError) Error() string {
	return fmt.Sprintf("Expected to find one stack named %s, but found %d", e.StackName, e.Count)
}

func (e *StackNotFoundError) NotFound() bool {
	return true
}

type ServiceInstanceNotFoundError struct {
	ServiceInstanceName string
	Count               int
}

func (e *ServiceInstanceNotFoundError) Error() string {
	return fmt.Sprintf("Expected to find one service instance named %s, but found %d", e.ServiceInstanceName, e.Count)
}

func (e *ServiceInstanceNotFoundError) NotFound() bool {
	return true
}

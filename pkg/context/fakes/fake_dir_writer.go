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

// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"io/fs"
	"sync"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
)

type FakeDirWriter struct {
	IsEmptyStub        func(string) (bool, error)
	isEmptyMutex       sync.RWMutex
	isEmptyArgsForCall []struct {
		arg1 string
	}
	isEmptyReturns struct {
		result1 bool
		result2 error
	}
	isEmptyReturnsOnCall map[int]struct {
		result1 bool
		result2 error
	}
	MkdirStub        func(string) error
	mkdirMutex       sync.RWMutex
	mkdirArgsForCall []struct {
		arg1 string
	}
	mkdirReturns struct {
		result1 error
	}
	mkdirReturnsOnCall map[int]struct {
		result1 error
	}
	MkdirAllStub        func(string, fs.FileMode) error
	mkdirAllMutex       sync.RWMutex
	mkdirAllArgsForCall []struct {
		arg1 string
		arg2 fs.FileMode
	}
	mkdirAllReturns struct {
		result1 error
	}
	mkdirAllReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeDirWriter) IsEmpty(arg1 string) (bool, error) {
	fake.isEmptyMutex.Lock()
	ret, specificReturn := fake.isEmptyReturnsOnCall[len(fake.isEmptyArgsForCall)]
	fake.isEmptyArgsForCall = append(fake.isEmptyArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.IsEmptyStub
	fakeReturns := fake.isEmptyReturns
	fake.recordInvocation("IsEmpty", []interface{}{arg1})
	fake.isEmptyMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeDirWriter) IsEmptyCallCount() int {
	fake.isEmptyMutex.RLock()
	defer fake.isEmptyMutex.RUnlock()
	return len(fake.isEmptyArgsForCall)
}

func (fake *FakeDirWriter) IsEmptyCalls(stub func(string) (bool, error)) {
	fake.isEmptyMutex.Lock()
	defer fake.isEmptyMutex.Unlock()
	fake.IsEmptyStub = stub
}

func (fake *FakeDirWriter) IsEmptyArgsForCall(i int) string {
	fake.isEmptyMutex.RLock()
	defer fake.isEmptyMutex.RUnlock()
	argsForCall := fake.isEmptyArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeDirWriter) IsEmptyReturns(result1 bool, result2 error) {
	fake.isEmptyMutex.Lock()
	defer fake.isEmptyMutex.Unlock()
	fake.IsEmptyStub = nil
	fake.isEmptyReturns = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *FakeDirWriter) IsEmptyReturnsOnCall(i int, result1 bool, result2 error) {
	fake.isEmptyMutex.Lock()
	defer fake.isEmptyMutex.Unlock()
	fake.IsEmptyStub = nil
	if fake.isEmptyReturnsOnCall == nil {
		fake.isEmptyReturnsOnCall = make(map[int]struct {
			result1 bool
			result2 error
		})
	}
	fake.isEmptyReturnsOnCall[i] = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *FakeDirWriter) Mkdir(arg1 string) error {
	fake.mkdirMutex.Lock()
	ret, specificReturn := fake.mkdirReturnsOnCall[len(fake.mkdirArgsForCall)]
	fake.mkdirArgsForCall = append(fake.mkdirArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.MkdirStub
	fakeReturns := fake.mkdirReturns
	fake.recordInvocation("Mkdir", []interface{}{arg1})
	fake.mkdirMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeDirWriter) MkdirCallCount() int {
	fake.mkdirMutex.RLock()
	defer fake.mkdirMutex.RUnlock()
	return len(fake.mkdirArgsForCall)
}

func (fake *FakeDirWriter) MkdirCalls(stub func(string) error) {
	fake.mkdirMutex.Lock()
	defer fake.mkdirMutex.Unlock()
	fake.MkdirStub = stub
}

func (fake *FakeDirWriter) MkdirArgsForCall(i int) string {
	fake.mkdirMutex.RLock()
	defer fake.mkdirMutex.RUnlock()
	argsForCall := fake.mkdirArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeDirWriter) MkdirReturns(result1 error) {
	fake.mkdirMutex.Lock()
	defer fake.mkdirMutex.Unlock()
	fake.MkdirStub = nil
	fake.mkdirReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeDirWriter) MkdirReturnsOnCall(i int, result1 error) {
	fake.mkdirMutex.Lock()
	defer fake.mkdirMutex.Unlock()
	fake.MkdirStub = nil
	if fake.mkdirReturnsOnCall == nil {
		fake.mkdirReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.mkdirReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeDirWriter) MkdirAll(arg1 string, arg2 fs.FileMode) error {
	fake.mkdirAllMutex.Lock()
	ret, specificReturn := fake.mkdirAllReturnsOnCall[len(fake.mkdirAllArgsForCall)]
	fake.mkdirAllArgsForCall = append(fake.mkdirAllArgsForCall, struct {
		arg1 string
		arg2 fs.FileMode
	}{arg1, arg2})
	stub := fake.MkdirAllStub
	fakeReturns := fake.mkdirAllReturns
	fake.recordInvocation("MkdirAll", []interface{}{arg1, arg2})
	fake.mkdirAllMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeDirWriter) MkdirAllCallCount() int {
	fake.mkdirAllMutex.RLock()
	defer fake.mkdirAllMutex.RUnlock()
	return len(fake.mkdirAllArgsForCall)
}

func (fake *FakeDirWriter) MkdirAllCalls(stub func(string, fs.FileMode) error) {
	fake.mkdirAllMutex.Lock()
	defer fake.mkdirAllMutex.Unlock()
	fake.MkdirAllStub = stub
}

func (fake *FakeDirWriter) MkdirAllArgsForCall(i int) (string, fs.FileMode) {
	fake.mkdirAllMutex.RLock()
	defer fake.mkdirAllMutex.RUnlock()
	argsForCall := fake.mkdirAllArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeDirWriter) MkdirAllReturns(result1 error) {
	fake.mkdirAllMutex.Lock()
	defer fake.mkdirAllMutex.Unlock()
	fake.MkdirAllStub = nil
	fake.mkdirAllReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeDirWriter) MkdirAllReturnsOnCall(i int, result1 error) {
	fake.mkdirAllMutex.Lock()
	defer fake.mkdirAllMutex.Unlock()
	fake.MkdirAllStub = nil
	if fake.mkdirAllReturnsOnCall == nil {
		fake.mkdirAllReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.mkdirAllReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeDirWriter) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.isEmptyMutex.RLock()
	defer fake.isEmptyMutex.RUnlock()
	fake.mkdirMutex.RLock()
	defer fake.mkdirMutex.RUnlock()
	fake.mkdirAllMutex.RLock()
	defer fake.mkdirAllMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeDirWriter) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ context.DirWriter = new(FakeDirWriter)

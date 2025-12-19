/*******************************************************************************

Copyright (c) 2024, Perforce Software, Inc.  All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1.  Redistributions of source code must retain the above copyright
    notice, this list of conditions and the following disclaimer.

2.  Redistributions in binary form must reproduce the above copyright
    notice, this list of conditions and the following disclaimer in the
    documentation and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
ARE DISCLAIMED. IN NO EVENT SHALL PERFORCE SOFTWARE, INC. BE LIABLE FOR ANY
DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

*******************************************************************************/

package p4

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)

// #include "p4go.h"
// #include <stdlib.h>
import "C"

type P4ResultType int

const (
	P4RESULTTYPE_STRING P4ResultType = iota
	P4RESULTTYPE_BINARY
	P4RESULTTYPE_TRACK
	P4RESULTTYPE_DICT
	P4RESULTTYPE_MESSAGE
	P4RESULTTYPE_SPEC
)

type P4Result interface {
	ResultType() P4ResultType
}

type P4Data string

func (P4Data) ResultType() P4ResultType { return P4RESULTTYPE_STRING }

type P4Track string

func (P4Track) ResultType() P4ResultType { return P4RESULTTYPE_TRACK }

type progressPtrMap struct {
	items map[*C.P4GoProgress]*P4Progress
	sync.RWMutex
}

func (c *progressPtrMap) Set(k *C.P4GoProgress, v *P4Progress) {
	c.Lock()
	defer c.Unlock()

	c.items[k] = v
}

func (c *progressPtrMap) Get(k *C.P4GoProgress) *P4Progress {
	c.RLock()
	defer c.RUnlock()

	return c.items[k]
}

func (c *progressPtrMap) Delete(k *C.P4GoProgress) {
	c.Lock()
	defer c.Unlock()

	delete(c.items, k)
}

var progress_pointer_map = &progressPtrMap{
	items: make(map[*C.P4GoProgress]*P4Progress),
}

type handlerPtrMap struct {
	items map[*C.P4GoHandler]*P4OutputHandler
	sync.RWMutex
}

func (c *handlerPtrMap) Set(k *C.P4GoHandler, v *P4OutputHandler) {
	c.Lock()
	defer c.Unlock()

	c.items[k] = v
}

func (c *handlerPtrMap) Get(k *C.P4GoHandler) *P4OutputHandler {
	c.RLock()
	defer c.RUnlock()

	return c.items[k]
}

func (c *handlerPtrMap) Delete(k *C.P4GoHandler) {
	c.Lock()
	defer c.Unlock()

	delete(c.items, k)
}

var handler_pointer_map = &handlerPtrMap{
	items: make(map[*C.P4GoHandler]*P4OutputHandler),
}

type ssoHandlerPtrMap struct {
	items map[*C.P4GoSSOHandler]*P4SSOHandler
	sync.RWMutex
}

func (c *ssoHandlerPtrMap) Set(k *C.P4GoSSOHandler, v *P4SSOHandler) {
	c.Lock()
	defer c.Unlock()

	c.items[k] = v
}

func (c *ssoHandlerPtrMap) Get(k *C.P4GoSSOHandler) *P4SSOHandler {
	c.RLock()
	defer c.RUnlock()

	return c.items[k]
}

func (c *ssoHandlerPtrMap) Delete(k *C.P4GoSSOHandler) {
	c.Lock()
	defer c.Unlock()

	delete(c.items, k)
}

var ssohandler_pointer_map = &ssoHandlerPtrMap{
	items: make(map[*C.P4GoSSOHandler]*P4SSOHandler),
}

type resolveHandlerPtrMap struct {
	items map[*C.P4GoResolveHandler]*P4ResolveHandler
	sync.RWMutex
}

func (c *resolveHandlerPtrMap) Set(k *C.P4GoResolveHandler, v *P4ResolveHandler) {
	c.Lock()
	defer c.Unlock()

	c.items[k] = v
}

func (c *resolveHandlerPtrMap) Get(k *C.P4GoResolveHandler) *P4ResolveHandler {
	c.RLock()
	defer c.RUnlock()

	return c.items[k]
}

func (c *resolveHandlerPtrMap) Delete(k *C.P4GoResolveHandler) {
	c.Lock()
	defer c.Unlock()

	delete(c.items, k)
}

var resolvehandler_pointer_map = &resolveHandlerPtrMap{
	items: make(map[*C.P4GoResolveHandler]*P4ResolveHandler),
}

type P4 struct {
	handle         *C.P4GoClientApi
	progresshandle *C.P4GoProgress
	outputhandle   *C.P4GoHandler
	ssohandle      *C.P4GoSSOHandler
	resolvehandle  *C.P4GoResolveHandler
}

func New() *P4 {
	p := C.NewClientApi()
	ret := &P4{handle: p, progresshandle: nil, outputhandle: nil, ssohandle: nil, resolvehandle: nil}
	return ret
}

func (p4 *P4) Close() {
	if p4.progresshandle != nil {
		progress_pointer_map.Delete(p4.progresshandle)
		C.FreeProgress(p4.progresshandle)
	}
	if p4.outputhandle != nil {
		handler_pointer_map.Delete(p4.outputhandle)
		C.FreeHandler(p4.outputhandle)
	}
	if p4.ssohandle != nil {
		ssohandler_pointer_map.Delete(p4.ssohandle)
		C.FreeSSOHandler(p4.ssohandle)
	}
	if p4.resolvehandle != nil {
		resolvehandler_pointer_map.Delete(p4.resolvehandle)
		C.FreeResolveHandler(p4.resolvehandle)
	}
	C.FreeClientApi(p4.handle)
}

func (p4 *P4) Identify() string {
	p := C.P4Identify(p4.handle)
	ret := C.GoString(p)
	return ret
}

func (p4 *P4) Connect() (bool, error) {
	result, err := handleCError(func(e *C.Error) interface{} {
		return C.P4Connect(p4.handle, e) != 0
	})
	return result.(bool), err
}

func (p4 *P4) Connected() bool {
	return C.P4Connected(p4.handle) != 0
}

func (p4 *P4) Disconnect() (bool, error) {
	result, err := handleCError(func(e *C.Error) interface{} {
		return C.P4Disconnect(p4.handle, e) != 0
	})
	return result.(bool), err
}

type Dictionary map[string]interface{}

func (p4 *P4) Run(cmd string, args ...string) ([]P4Result, error) {
	c_cmd := C.CString(cmd)

	argc := len(args)
	argv := make([]*C.char, argc+1)
	for i, arg := range args {
		carg := C.CString(arg)
		defer C.free(unsafe.Pointer(carg))
		argv[i] = carg
	}

	_, run_err := handleCError(func(e *C.Error) interface{} {
		C.Run(p4.handle, c_cmd, C.int(argc), &argv[0], e)
		return true
	})

	C.free(unsafe.Pointer(c_cmd))

	results := []P4Result{}
	rcount := int(C.ResultCount(p4.handle))
	for i := 0; i < rcount; i++ {
		t := C.int(0)
		r := (*C.P4GoResult)(C.malloc(C.size_t(1)))
		if C.ResultGet(p4.handle, C.int(i), &t, &r) != 0 {
			switch P4ResultType(int(t)) {
			case P4RESULTTYPE_STRING:
				s := C.ResultGetString(r)
				results = append(results, P4Data(C.GoString(s)))
				C.free(unsafe.Pointer(s))
			case P4RESULTTYPE_BINARY:
				l := C.int(0)
				s := C.ResultGetBinary(r, &l)
				results = append(results, P4Data(C.GoBytes(unsafe.Pointer(s), l)))
				C.free(unsafe.Pointer(s))
			case P4RESULTTYPE_TRACK:
				s := C.ResultGetString(r)
				results = append(results, P4Track(C.GoString(s)))
				C.free(unsafe.Pointer(s))
			case P4RESULTTYPE_DICT:
				j := 0
				k := (*C.char)(C.malloc(C.size_t(1)))
				v := (*C.char)(C.malloc(C.size_t(1)))
				d := Dictionary{}
				for C.ResultGetKeyPair(r, C.int(j), &k, &v) != 0 {
					j++
					d[C.GoString(k)] = C.GoString(v)
				}
				C.free(unsafe.Pointer(k))
				C.free(unsafe.Pointer(v))
				results = append(results, d)
			case P4RESULTTYPE_SPEC:
				// Convert spec result using SpecData API to properly handle arrays
				spec := C.ResultGetSpec(r)
				d := convertSpecDataToDict(spec)
				results = append(results, d)
			case P4RESULTTYPE_MESSAGE:
				e := C.ResultGetError(r)
				err := P4Message{}
				err.severity = P4MessageSeverity(C.GetErrorSeverity(e))
				ec := int(C.GetErrorCount(e))
				err.lines = []P4MessageLine{}
				err.msgdict = goGetMsgDict(e)
				for j := 0; j < ec; j++ {
					el := P4MessageLine{}
					el.severity = P4MessageSeverity(C.GetErrorSeverityI(e, C.int(j)))
					s := C.FmtError(e, C.int(j))
					el.fmt = C.GoString(s)
					C.free(unsafe.Pointer(s))
					el.code = int(C.GetErrorCode(e, C.int(j)))
					err.lines = append(err.lines, el)
					err.msgdict[fmt.Sprintf("Error %d", j)] = el.fmt
				}
				results = append(results, err)

			default:
				// Unknown result?
			}
		}
	}

	return results, run_err
}

func (p4 *P4) ApiLevel() int {
	return int(C.GetApiLevel(p4.handle))
}

func (p4 *P4) SetApiLevel(apiLevel int) {
	C.SetApiLevel(p4.handle, C.int(apiLevel))
}

func (p4 *P4) Streams() bool {
	return int(C.GetStreams(p4.handle)) != 0
}

func (p4 *P4) SetStreams(enableStreams bool) {
	flag := int8(0)
	if enableStreams {
		flag = 1
	}
	C.SetStreams(p4.handle, C.int(flag))
}

func (p4 *P4) Tagged() bool {
	return int(C.GetTagged(p4.handle)) != 0

}

func (p4 *P4) SetTagged(enableTagged bool) {
	flag := int8(0)
	if enableTagged {
		flag = 1
	}
	C.SetTagged(p4.handle, C.int(flag))
}

func (p4 *P4) Track() bool {
	return int(C.GetTrack(p4.handle)) != 0
}

func (p4 *P4) SetTrack(enableTrack bool) (bool, error) {
	flag := int8(0)
	if enableTrack {
		flag = 1
	}
	result, err := handleCError(func(e *C.Error) interface{} {
		return C.SetTrack(p4.handle, C.int(flag), e) != 0
	})
	return result.(bool), err
}

func (p4 *P4) Graph() bool {
	return int(C.GetGraph(p4.handle)) != 0
}

func (p4 *P4) SetGraph(enableGraph bool) {
	flag := int8(0)
	if enableGraph {
		flag = 1
	}
	C.SetGraph(p4.handle, C.int(flag))
}

func (p4 *P4) Debug() int {
	return int(C.GetDebug(p4.handle))
}

func (p4 *P4) SetDebug(debugLevel int) {
	C.SetDebug(p4.handle, C.int(debugLevel))
}

func (p4 *P4) Charset() string {
	return C.GoString(C.GetCharset(p4.handle))
}

func (p4 *P4) SetCharset(charset string) (bool, error) {
	c_charset := C.CString(charset)
	defer C.free(unsafe.Pointer(c_charset))
	result, err := handleCError(func(e *C.Error) interface{} {
		return C.SetCharset(p4.handle, c_charset, e) != 0
	})
	return result.(bool), err
}

func (p4 *P4) Cwd() string {
	return C.GoString(C.GetCwd(p4.handle))
}

func (p4 *P4) SetCwd(cwd string) {
	c_cwd := C.CString(cwd)
	C.SetCwd(p4.handle, c_cwd)
	C.free(unsafe.Pointer(c_cwd))
}

func (p4 *P4) Client() string {
	return C.GoString(C.GetClient(p4.handle))
}

func (p4 *P4) SetClient(client string) {
	c_client := C.CString(client)
	C.SetClient(p4.handle, c_client)
	C.free(unsafe.Pointer(c_client))
}

func (p4 *P4) Env(env string) string {
	c_env := C.CString(env)
	res := C.GoString(C.GetEnv(p4.handle, c_env))
	C.free(unsafe.Pointer(c_env))
	return res
}

func (p4 *P4) SetEnv(env string, value string) (bool, error) {
	c_env := C.CString(env)
	c_value := C.CString(value)
	defer C.free(unsafe.Pointer(c_env))
	defer C.free(unsafe.Pointer(c_value))
	result, err := handleCError(func(e *C.Error) interface{} {
		return C.SetEnv(p4.handle, c_env, c_value, e) != 0
	})
	return result.(bool), err
}

func (p4 *P4) EnviroFile() string {
	return C.GoString(C.GetEnviroFile(p4.handle))
}

func (p4 *P4) SetEnviroFile(enviroFile string) {
	c_enviroFile := C.CString(enviroFile)
	C.SetEnviroFile(p4.handle, c_enviroFile)
	C.free(unsafe.Pointer(c_enviroFile))
}

func (p4 *P4) EVar(evar string) string {
	c_evar := C.CString(evar)
	res := C.GoString(C.GetEVar(p4.handle, c_evar))
	C.free(unsafe.Pointer(c_evar))
	return res
}

func (p4 *P4) SetEVar(evar string, value string) {
	c_evar := C.CString(evar)
	c_value := C.CString(value)
	C.SetEVar(p4.handle, c_evar, c_value)
	C.free(unsafe.Pointer(c_evar))
	C.free(unsafe.Pointer(c_value))
}

func (p4 *P4) Host() string {
	return C.GoString(C.GetHost(p4.handle))
}

func (p4 *P4) SetHost(host string) {
	c_host := C.CString(host)
	C.SetHost(p4.handle, c_host)
	C.free(unsafe.Pointer(c_host))
}

func (p4 *P4) IgnoreFile() string {
	return C.GoString(C.GetIgnoreFile(p4.handle))
}

func (p4 *P4) SetIgnoreFile(ignoreFile string) {
	c_ignoreFile := C.CString(ignoreFile)
	C.SetIgnoreFile(p4.handle, c_ignoreFile)
	C.free(unsafe.Pointer(c_ignoreFile))
}

func (p4 *P4) Ignored(path string) bool {
	c_path := C.CString(path)
	res := int(C.IsIgnored(p4.handle, c_path)) != 0
	C.free(unsafe.Pointer(c_path))
	return res
}

func (p4 *P4) Language() string {
	return C.GoString(C.GetLanguage(p4.handle))
}

func (p4 *P4) SetLanguage(language string) {
	c_language := C.CString(language)
	C.SetLanguage(p4.handle, c_language)
	C.free(unsafe.Pointer(c_language))
}

func (p4 *P4) GetP4ConfigFile() string {
	return C.GoString(C.GetP4ConfigFile(p4.handle))
}

func (p4 *P4) Password() string {
	return C.GoString(C.GetPassword(p4.handle))
}

func (p4 *P4) SetPassword(password string) {
	c_password := C.CString(password)
	C.SetPassword(p4.handle, c_password)
	C.free(unsafe.Pointer(c_password))
}

func (p4 *P4) Port() string {
	return C.GoString(C.GetPort(p4.handle))
}

func (p4 *P4) SetPort(port string) {
	c_port := C.CString(port)
	C.SetPort(p4.handle, c_port)
	C.free(unsafe.Pointer(c_port))
}

func (p4 *P4) Prog() string {
	return C.GoString(C.GetProg(p4.handle))
}

func (p4 *P4) SetProg(prog string) {
	c_prog := C.CString(prog)
	C.SetProg(p4.handle, c_prog)
	C.free(unsafe.Pointer(c_prog))
}

func (p4 *P4) SetProtocol(protocol string, value string) {
	c_protocol := C.CString(protocol)
	c_value := C.CString(value)
	C.SetProtocol(p4.handle, c_protocol, c_value)
	C.free(unsafe.Pointer(c_protocol))
	C.free(unsafe.Pointer(c_value))
}

func (p4 *P4) SetVar(variable string, value string) {
	c_variable := C.CString(variable)
	c_value := C.CString(value)
	C.SetVar(p4.handle, c_variable, c_value)
	C.free(unsafe.Pointer(c_variable))
	C.free(unsafe.Pointer(c_value))
}

func (p4 *P4) TicketFile() string {
	return C.GoString(C.GetTicketFile(p4.handle))
}

func (p4 *P4) SetTicketFile(ticketFile string) {
	c_ticketFile := C.CString(ticketFile)
	C.SetTicketFile(p4.handle, c_ticketFile)
	C.free(unsafe.Pointer(c_ticketFile))
}

func (p4 *P4) TrustFile() string {
	return C.GoString(C.GetTrustFile(p4.handle))
}

func (p4 *P4) SetTrustFile(trustFile string) {
	c_trustFile := C.CString(trustFile)
	C.SetTrustFile(p4.handle, c_trustFile)
	C.free(unsafe.Pointer(c_trustFile))
}

func (p4 *P4) User() string {
	return C.GoString(C.GetUser(p4.handle))
}

func (p4 *P4) SetUser(user string) {
	c_user := C.CString(user)
	C.SetUser(p4.handle, c_user)
	C.free(unsafe.Pointer(c_user))
}

func (p4 *P4) Version() string {
	return C.GoString(C.GetP4Version(p4.handle))
}

func (p4 *P4) SetVersion(version string) {
	c_version := C.CString(version)
	C.SetP4Version(p4.handle, c_version)
	C.free(unsafe.Pointer(c_version))
}

func (p4 *P4) MaxResults() int {
	return int(C.GetMaxResults(p4.handle))
}

func (p4 *P4) SetResults(maxResults int) {
	C.SetMaxResults(p4.handle, C.int(maxResults))
}

func (p4 *P4) MaxScanRows() int {
	return int(C.GetMaxScanRows(p4.handle))
}

func (p4 *P4) SetMaxScanRows(maxScanRows int) {
	C.SetMaxScanRows(p4.handle, C.int(maxScanRows))
}

func (p4 *P4) MaxLockTime() int {
	return int(C.GetMaxLockTime(p4.handle))
}

func (p4 *P4) SetMaxLockTime(maxLockTime int) {
	C.SetMaxLockTime(p4.handle, C.int(maxLockTime))
}

func (p4 *P4) SetInput(input ...string) {
	C.ResetInput(p4.handle)
	for _, in := range input {
		c_in := C.CString(in)
		C.AppendInput(p4.handle, c_in)
		C.free(unsafe.Pointer(c_in))
	}
}

// convertSpecDataToDict converts a P4GoSpecData pointer to a Dictionary
// handling both scalar values and arrays properly
func convertSpecDataToDict(spec *C.P4GoSpecData) Dictionary {
	dict := Dictionary{}

	if spec == nil {
		return dict
	}

	// Get count of all variables (including array elements)
	varCount := int(C.SpecDataGetVarCount(spec))

	// Collect all raw key-value pairs
	rawPairs := make(map[string]string)
	for i := 0; i < varCount; i++ {
		var c_key *C.char
		var c_val *C.char
		if C.SpecDataGetKeyPair(spec, C.int(i), &c_key, &c_val) != 0 {
			key := C.GoString(c_key)
			val := C.GoString(c_val)
			rawPairs[key] = val
			// Don't free c_key and c_val - they're internal pointers managed by SpecData
		}
	}

	// Group keys by base name
	baseKeys := make(map[string][]string) // baseKey -> list of full keys
	scalarKeys := make(map[string]string) // scalar keys

	for key := range rawPairs {
		if idx := strings.Index(key, "["); idx != -1 {
			// Array element: "View[0]" -> base="View"
			baseKey := key[:idx]
			baseKeys[baseKey] = append(baseKeys[baseKey], key)
		} else {
			// Check if this is a base key that has array elements
			hasArrayElements := false
			for fullKey := range rawPairs {
				if strings.HasPrefix(fullKey, key+"[") {
					hasArrayElements = true
					break
				}
			}
			if !hasArrayElements {
				scalarKeys[key] = rawPairs[key]
			}
		}
	}

	// Add scalar values to dict
	for key, val := range scalarKeys {
		dict[key] = val
	}

	// Add array values to dict
	for baseKey, keys := range baseKeys {
		// Sort keys by index
		indices := make(map[int]string)
		maxIndex := -1

		for _, key := range keys {
			// Extract index from "BaseKey[123]"
			start := strings.Index(key, "[")
			end := strings.Index(key, "]")
			if start != -1 && end != -1 {
				indexStr := key[start+1 : end]
				index, err := strconv.Atoi(indexStr)
				if err == nil {
					indices[index] = rawPairs[key]
					if index > maxIndex {
						maxIndex = index
					}
				}
			}
		}

		// Build array in order
		values := make([]string, 0, maxIndex+1)
		for i := 0; i <= maxIndex; i++ {
			if val, exists := indices[i]; exists {
				values = append(values, val)
			}
		}

		dict[baseKey] = values
	}

	return dict
}

func (p4 *P4) ParseSpec(spec string, form string) (Dictionary, error) {
	c_spec := C.CString(spec)
	c_form := C.CString(form)
	defer C.free(unsafe.Pointer(c_spec))
	defer C.free(unsafe.Pointer(c_form))
	result, err := handleCError(func(e *C.Error) interface{} {
		return C.ParseSpec(p4.handle, c_spec, c_form, e)
	})

	s := result.(*C.P4GoSpecData)
	defer C.FreeSpecData(s)

	dict := convertSpecDataToDict(s)

	return dict, err
}

// splitNumberedKey checks if a key is in the format "FieldN" (e.g., "View0", "Paths1")
// Returns the base key and index if it's a numbered key, otherwise returns ("", -1)
func splitNumberedKey(key string) (string, int) {
	// Find where digits start from the end
	i := len(key) - 1
	for i >= 0 && key[i] >= '0' && key[i] <= '9' {
		i--
	}

	// If no digits found or all digits, not a numbered key
	if i < 0 || i == len(key)-1 {
		return "", -1
	}

	// Extract base and number
	baseKey := key[:i+1]
	numStr := key[i+1:]

	// Parse the number
	var index int
	_, err := fmt.Sscanf(numStr, "%d", &index)
	if err != nil {
		return "", -1
	}

	return baseKey, index
}

// setArrayInStrDict sets an array of values in a StrDict with proper P4 API format
// It sets the base key to empty string (required by P4GoSpecData::GetLine)
// and then sets each element with [index] notation
func setArrayInStrDict(d *C.StrDict, baseKey string, values []string) {
	// Set dummy value for base key so GetLine doesn't bail early
	c_baseKey := C.CString(baseKey)
	c_dummy := C.CString("")
	C.StrDictSetKeyPair(d, c_baseKey, c_dummy)
	C.free(unsafe.Pointer(c_baseKey))
	C.free(unsafe.Pointer(c_dummy))

	// Set each array element with [index] notation
	for i, item := range values {
		arrayKey := fmt.Sprintf("%s[%d]", baseKey, i)
		c_ak := C.CString(arrayKey)
		c_v := C.CString(item)
		C.StrDictSetKeyPair(d, c_ak, c_v)
		C.free(unsafe.Pointer(c_ak))
		C.free(unsafe.Pointer(c_v))
	}
}

func (p4 *P4) FormatSpec(spec string, dict Dictionary) (string, error) {

	d := C.NewStrDict()

	// First pass: identify which base keys have numbered variants
	// e.g., if we see "View0", mark "View" as having numbered keys
	numberedBaseKeys := make(map[string]bool)

	for k := range dict {
		baseKey, index := splitNumberedKey(k)
		if index >= 0 {
			numberedBaseKeys[baseKey] = true
		}
	}

	// Second pass: process all keys
	processedKeys := make(map[string]bool)

	for k, v := range dict {
		// Check if this is a numbered key
		_, index := splitNumberedKey(k)
		if index >= 0 {
			// This is a numbered key (e.g., "View0")
			// Skip it for now - we'll process all numbered keys together below
			processedKeys[k] = true
			continue
		}

		// Check if this key has numbered variants (e.g., "View" when "View0" exists)
		if numberedBaseKeys[k] {
			// Skip the base key if numbered variants exist
			// The numbered variants will be processed below
			processedKeys[k] = true
			continue
		}

		c_k := C.CString(k)

		switch val := v.(type) {
		case string:
			// Handle scalar string values
			c_v := C.CString(val)
			C.StrDictSetKeyPair(d, c_k, c_v)
			C.free(unsafe.Pointer(c_v))
			processedKeys[k] = true
		case []string:
			// Handle array values - convert to P4 API format
			setArrayInStrDict(d, k, val)
			processedKeys[k] = true
		default:
			// Unsupported type, skip
			processedKeys[k] = true
		}

		C.free(unsafe.Pointer(c_k))
	}

	// Third pass: process grouped numbered keys for backward compatibility
	// e.g., "View0", "View1", "View2" -> internal "View[0]", "View[1]", "View[2]"
	for baseKey := range numberedBaseKeys {
		// Collect all numbered values into an array
		var values []string
		i := 0
		for {
			numberedKey := fmt.Sprintf("%s%d", baseKey, i)
			val, exists := dict[numberedKey]
			if !exists {
				break
			}
			if strVal, ok := val.(string); ok {
				values = append(values, strVal)
			}
			i++
		}

		// Use the same helper function to set the array
		if len(values) > 0 {
			setArrayInStrDict(d, baseKey, values)
		}
	}

	c_spec := C.CString(spec)
	defer C.free(unsafe.Pointer(c_spec))
	defer C.FreeStrDict(d)
	result, err := handleCError(func(e *C.Error) interface{} {
		return C.FormatSpec(p4.handle, c_spec, d, e)
	})

	f := result.(*C.char)
	form := C.GoString(f)
	C.free(unsafe.Pointer(f))

	return form, err
}

func (p4 *P4) ServerLevel() (int, error) {
	result, err := handleCError(func(e *C.Error) interface{} {
		return int(C.P4ServerLevel(p4.handle, e))
	})
	return result.(int), err
}

func (p4 *P4) ServerUnicode() (bool, error) {
	result, err := handleCError(func(e *C.Error) interface{} {
		return C.P4ServerUnicode(p4.handle, e) != 0
	})
	return result.(bool), err
}

func (p4 *P4) ServerCaseSensitive() (bool, error) {

	result, err := handleCError(func(e *C.Error) interface{} {
		return C.P4ServerCaseSensitive(p4.handle, e) != 0
	})
	return result.(bool), err
}

// Generic Fetch methods
func (p4 *P4) RunFetch(spec string, args ...string) (Dictionary, error) {
	if len(args) == 0 {
		args = []string{}
	}

	args = append([]string{"-o"}, args...)
	raw, run_err := p4.Run(spec, args...)

	for _, r := range raw {
		switch v := r.(type) {
		case P4Message:
			if v.Severity() == P4MESSAGE_FAILED || v.Severity() == P4MESSAGE_FATAL {
				return nil, errors.Join(run_err, &v)
			}
		case Dictionary:
			return v, run_err
		}
	}

	return nil, run_err
}

// Generic Save methods
func (p4 *P4) RunSave(spec string, specdict Dictionary, args ...string) (*P4Message, error) {
	formattedSpec, err := p4.FormatSpec(spec, specdict)
	if err != nil {
		return nil, err
	}
	p4.SetInput(formattedSpec)

	args = append([]string{"-i"}, args...)
	raw, run_err := p4.Run(spec, args...)

	// Process the results
	for _, r := range raw {
		if msg, ok := r.(P4Message); ok {
			switch msg.Severity() {
			case P4MESSAGE_FAILED, P4MESSAGE_FATAL:
				return nil, errors.Join(run_err, &msg)
			case P4MESSAGE_INFO, P4MESSAGE_WARN:
				return &msg, run_err
			}
		}
	}

	return nil, run_err
}

// Generic Submit method
func (p4 *P4) RunSubmit(args ...interface{}) ([]Dictionary, error) {
	var result []Dictionary
	var specDict Dictionary
	var finalArgs []string

	// Process arguments
	for _, arg := range args {
		switch v := arg.(type) {
		case Dictionary:
			specDict = v
		case string:
			finalArgs = append(finalArgs, v)
		case int:
			finalArgs = append(finalArgs, strconv.Itoa(v))
		}
	}

	// If a dict is provided, set it as input and add "-i" to arguments
	if specDict != nil {
		formattedSpec, err := p4.FormatSpec("change", specDict)
		if err != nil {
			return nil, err
		}
		p4.SetInput(formattedSpec)
		finalArgs = append(finalArgs, "-i")
	}

	raw, run_err := p4.Run("submit", finalArgs...)

	for _, r := range raw {
		switch v := r.(type) {
		case P4Message:
			if v.Severity() == P4MESSAGE_FAILED || v.Severity() == P4MESSAGE_FATAL {
				return nil, errors.Join(run_err, &v)
			}
		case Dictionary:
			result = append(result, v)
		}
	}

	return result, run_err
}

// Generic Shelve method
func (p4 *P4) RunShelve(args ...interface{}) ([]Dictionary, error) {
	var result []Dictionary
	var specDict Dictionary
	var finalArgs []string

	// Process arguments
	for _, arg := range args {
		switch v := arg.(type) {
		case Dictionary:
			specDict = v
		case string:
			finalArgs = append(finalArgs, v)
		case int:
			finalArgs = append(finalArgs, strconv.Itoa(v))
		}
	}

	// If a dict is provided, set it as input and add "-i" to arguments
	if specDict != nil {
		formattedSpec, err := p4.FormatSpec("change", specDict)
		if err != nil {
			return nil, err
		}
		p4.SetInput(formattedSpec)
		finalArgs = append(finalArgs, "-i")
	}

	raw, run_err := p4.Run("shelve", finalArgs...)

	for _, r := range raw {
		switch v := r.(type) {
		case P4Message:
			if v.Severity() == P4MESSAGE_FAILED || v.Severity() == P4MESSAGE_FATAL {
				return nil, errors.Join(run_err, &v)
			}
		case Dictionary:
			result = append(result, v)
		}
	}

	return result, run_err
}

// Generic Delete methods
func (p4 *P4) RunDelete(spec string, args ...string) (*P4Message, error) {
	if len(args) == 0 {
		args = []string{}
	}

	if spec == "shelve" {
		// Ensure "-c" is included as the first argument if not already present
		hasChangeFlag := false
		for _, arg := range args {
			if arg == "-c" {
				hasChangeFlag = true
				break
			}
		}
		if !hasChangeFlag {
			args = append([]string{"-c"}, args...)
		}
	}

	args = append([]string{"-d"}, args...)
	raw, run_err := p4.Run(spec, args...)

	for _, r := range raw {
		if msg, ok := r.(P4Message); ok {
			switch msg.Severity() {
			case P4MESSAGE_FAILED, P4MESSAGE_FATAL:
				return nil, errors.Join(run_err, &msg)
			case P4MESSAGE_INFO, P4MESSAGE_WARN:
				return &msg, run_err
			}
		}
	}

	return nil, run_err
}

func (p4 *P4) SpecIterator(spec string, args ...string) ([]Dictionary, error) {
	// Mapping for specs Iterator
	specTypes := map[string][2]string{
		"clients":  {"client", "client"},
		"labels":   {"label", "label"},
		"branches": {"branch", "branch"},
		"changes":  {"change", "change"},
		"streams":  {"stream", "Stream"},
		"jobs":     {"job", "Job"},
		"users":    {"user", "User"},
		"groups":   {"group", "group"},
		"depots":   {"depot", "name"},
		"servers":  {"server", "ServerID"},
		"ldaps":    {"ldap", "Name"},
		"remotes":  {"remote", "RemoteID"},
		"repos":    {"repo", "Repo"},
	}
	var results []Dictionary

	// Check if method exists in specTypes
	if _, exists := specTypes[spec]; !exists {
		return nil, fmt.Errorf("Not a P4 Spectype: %s", spec)
	}

	specs, run_err := p4.Run(spec, args...)

	c := specTypes[spec][0]
	k := specTypes[spec][1]

	for _, spec := range specs {
		if _, ok := spec.(Dictionary); !ok {
			return nil, errors.Join(run_err, fmt.Errorf("No such spec: %s", spec))
		}
		specDict := spec.(Dictionary)
		if k == "" {
			res, _ := p4.Run(c, "-o")
			results = append(results, (res[0]).(Dictionary))
		} else {
			if keyStr, ok := specDict[k].(string); ok {
				res, _ := p4.Run(c, "-o", keyStr)
				results = append(results, (res[0]).(Dictionary))
			} else {
				return nil, errors.Join(run_err, fmt.Errorf("specDict[%s] is not a string", k))
			}
		}

	}
	return results, run_err

}

func (p4 *P4) RunPassword(old string, new string) (*P4Message, error) {
	if len(old) > 0 {
		p4.SetInput(old, new, new)
	} else {
		p4.SetInput(new, new)
	}

	raw, run_err := p4.Run("password")

	for _, r := range raw {
		if msg, ok := r.(P4Message); ok {
			switch msg.Severity() {
			case P4MESSAGE_FAILED, P4MESSAGE_FATAL:
				return nil, errors.Join(run_err, &msg)
			case P4MESSAGE_INFO, P4MESSAGE_WARN:
				return &msg, run_err
			}
		}
	}

	return nil, run_err
}

func (p4 *P4) RunLogin(args ...string) (Dictionary, error) {
	p4.SetInput(p4.Password())
	raw, run_err := p4.Run("login", args...)

	for _, r := range raw {
		switch v := r.(type) {
		case P4Message:
			switch v.Severity() {
			case P4MESSAGE_FAILED, P4MESSAGE_FATAL:
				return nil, errors.Join(run_err, &v)
			case P4MESSAGE_INFO, P4MESSAGE_WARN:
				// Convert map[string]string to Dictionary (map[string]interface{})
				dict := Dictionary{}
				for k, val := range v.GetMsgDict() {
					dict[k] = val
				}
				return dict, nil
			}
		case Dictionary:
			return v, run_err
		}
	}

	return nil, run_err
}

func (p4 *P4) RunTickets() ([]map[string]string, error) {
	//  return an empty array if the file doesnt exist
	//  or is a directory.

	var results []map[string]string
	re := regexp.MustCompile(`([^=]*)=(.*):([^:]*)$`)

	path := p4.TicketFile()
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !fileInfo.IsDir() {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			res := re.FindStringSubmatch(line)
			if res != nil {
				tickets := map[string]string{
					"Host":   res[1],
					"User":   res[2],
					"Ticket": res[3],
				}
				results = append(results, tickets)
			}

		}

		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}
	return results, nil
}

// RunFilelog processes raw filelog data and returns a slice of DepotFiles.
func (p4 *P4) RunFilelog(args ...string) ([]*P4DepotFile, error) {
	var result []*P4DepotFile

	raw, run_err := p4.Run("filelog", args...)

	for _, h := range raw {
		var df *P4DepotFile
		var err error

		if hDict, ok := h.(Dictionary); ok {
			hMap := make(map[string]interface{})
			for key, value := range hDict {
				hMap[key] = value
			}
			df, err = ProcessFilelog(hMap)
			if err != nil {
				return nil, errors.Join(run_err, err)
			}
		} else {
			return nil, errors.Join(run_err, errors.New("unexpected filelog data type"))
		}

		result = append(result, df)
	}

	return result, run_err
}

func (Dictionary) ResultType() P4ResultType { return P4RESULTTYPE_DICT }

type P4MessageSeverity int

const (
	P4MESSAGE_EMPTY P4MessageSeverity = iota
	P4MESSAGE_INFO
	P4MESSAGE_WARN
	P4MESSAGE_FAILED
	P4MESSAGE_FATAL
)

func (s P4MessageSeverity) String() string {
	switch s {
	case P4MESSAGE_EMPTY:
		return "empty"
	case P4MESSAGE_INFO:
		return "info"
	case P4MESSAGE_WARN:
		return "warn"
	case P4MESSAGE_FAILED:
		return "failed"
	case P4MESSAGE_FATAL:
		return "fatal"
	}
	return "unknown"
}

type P4MessageLine struct {
	severity P4MessageSeverity
	code     int
	fmt      string
}

type P4Message struct {
	severity P4MessageSeverity
	lines    []P4MessageLine
	msgdict  map[string]string
}

func (P4Message) ResultType() P4ResultType { return P4RESULTTYPE_MESSAGE }

func (e *P4Message) Count() int {
	return len(e.lines)
}

func (e *P4Message) Severity() P4MessageSeverity {
	return e.severity
}

func (e *P4Message) Id(i int) int {
	return e.lines[i].code
}

func (e *P4Message) SubCode(i int) int   { return (e.lines[i].code >> 0) & 0x3ff }
func (e *P4Message) Subsystem(i int) int { return (e.lines[i].code >> 10) & 0x3f }
func (e *P4Message) Generic(i int) int   { return (e.lines[i].code >> 16) & 0xff }
func (e *P4Message) ArgCount(i int) int  { return (e.lines[i].code >> 24) & 0x0f }
func (e *P4Message) LineSeverity(i int) P4MessageSeverity {
	return P4MessageSeverity((e.lines[i].code >> 28) & 0x0f)
}

func (e *P4Message) UniqueCode(i int) int { return e.lines[i].code & 0xffff }

func (e *P4Message) GetLine(i int) P4MessageLine {
	return e.lines[i]
}

func (e *P4Message) String() string {
	var sb strings.Builder
	for i := 0; i < len(e.lines); i++ {
		if i != 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(e.lines[i].fmt)
	}
	return sb.String()
}

func (e P4Message) Error() string {
	return e.String()
}

func (el *P4MessageLine) String() string {
	return el.fmt
}

// public getter for the msg dict
func (e *P4Message) GetMsgDict() map[string]string {
	return e.msgdict
}

// function to populate the msgdict field
func goGetMsgDict(e *C.Error) map[string]string {

	dict := make(map[string]string)
	c_dict := C.GetDict(e)
	if c_dict != nil {
		i := 0
		var k, v *C.char
		for C.StrDictGetKeyPair(c_dict, C.int(i), &k, &v) != 0 {
			i++
			dict[C.GoString(k)] = C.GoString(v)
		}
	}
	return dict
}

// handleCError is a wrapper function for handling C function calls that return different types.
func handleCError(call func(e *C.Error) interface{}) (interface{}, error) {
	e := C.MakeError()
	defer C.FreeError(e)

	result := call(e)

	if int(C.GetErrorCount(e)) == 0 {
		return result, nil // No error occurred
	}

	// Process the error details
	err := P4Message{}
	err.severity = P4MessageSeverity(C.GetErrorSeverity(e))
	ec := int(C.GetErrorCount(e))
	err.lines = []P4MessageLine{}
	for j := 0; j < ec; j++ {
		el := P4MessageLine{}
		el.severity = P4MessageSeverity(C.GetErrorSeverityI(e, C.int(j)))
		s := C.FmtError(e, C.int(j))
		el.fmt = C.GoString(s)
		C.free(unsafe.Pointer(s))
		el.code = int(C.GetErrorCode(e, C.int(j)))
		err.lines = append(err.lines, el)
	}

	return result, err // Return nil for the result and the error details
}

// Callbacks

type P4Progress interface {
	Init(progress_type int)
	Description(desc string, units int)
	Total(total int64)
	Update(position int64)
	Done(failed bool)
}

func (p4 *P4) SetProgress(progress P4Progress) {
	p := C.GetProgress(p4.handle)
	if p != nil {
		progress_pointer_map.Delete(p)
		C.FreeProgress(p)
	}
	p4.progresshandle = nil
	if progress != nil {
		p4.progresshandle = C.NewProgress()
		progress_pointer_map.Set(p4.progresshandle, &progress)
		C.SetProgress(p4.handle, p4.progresshandle)
	} else {
		C.SetProgress(p4.handle, nil)
	}
}

//export goCallProgressInitFunction
func goCallProgressInitFunction(ctx unsafe.Pointer, t C.int) {
	progress := progress_pointer_map.Get((*C.P4GoProgress)(ctx))
	if progress != nil {
		(*progress).Init(int(t))
	}
}

//export goCallProgressDescFunction
func goCallProgressDescFunction(ctx unsafe.Pointer, d *C.char, u C.int) {
	progress := progress_pointer_map.Get((*C.P4GoProgress)(ctx))
	if progress != nil {
		(*progress).Description(C.GoString(d), int(u))
	}

}

//export goCallProgressTotalFunction
func goCallProgressTotalFunction(ctx unsafe.Pointer, t C.long) {
	progress := progress_pointer_map.Get((*C.P4GoProgress)(ctx))
	if progress != nil {
		(*progress).Total(int64(t))
	}
}

//export goCallProgressUpdateFunction
func goCallProgressUpdateFunction(ctx unsafe.Pointer, p C.long) {
	progress := progress_pointer_map.Get((*C.P4GoProgress)(ctx))
	if progress != nil {
		(*progress).Update(int64(p))
	}
}

//export goCallProgressDoneFunction
func goCallProgressDoneFunction(ctx unsafe.Pointer, t C.int) {
	progress := progress_pointer_map.Get((*C.P4GoProgress)(ctx))
	if progress != nil {
		(*progress).Done(bool(int(t) != 0))
	}
}

type P4OutputHandlerResult int

const (
	P4OUTPUTHANDLER_REPORT  P4OutputHandlerResult = iota // Let Run return the output
	P4OUTPUTHANDLER_HANDLED                              // Skip adding the output to the Run result
	P4OUTPUTHANDLER_CANCEL                               // Abort the command, invalidating the connection
)

func (s P4OutputHandlerResult) String() string {
	switch s {
	case P4OUTPUTHANDLER_REPORT:
		return "report"
	case P4OUTPUTHANDLER_HANDLED:
		return "handled"
	case P4OUTPUTHANDLER_CANCEL:
		return "cancel"
	}
	return "unknown"
}

type P4OutputHandler interface {
	HandleBinary(data []byte) P4OutputHandlerResult
	HandleMessage(msg P4Message) P4OutputHandlerResult
	HandleStat(dict Dictionary) P4OutputHandlerResult
	HandleText(data string) P4OutputHandlerResult
	HandleTrack(data string) P4OutputHandlerResult
	HandleSpec(dict Dictionary) P4OutputHandlerResult
}

func (p4 *P4) SetHandler(handler P4OutputHandler) {
	p := C.GetHandler(p4.handle)
	if p != nil {
		handler_pointer_map.Delete(p)
		C.FreeHandler(p)
	}
	p4.outputhandle = nil
	if handler != nil {
		p4.outputhandle = C.NewHandler()
		handler_pointer_map.Set(p4.outputhandle, &handler)
		C.SetHandler(p4.handle, p4.outputhandle)
	} else {
		C.SetHandler(p4.handle, nil)
	}
}

//export goCallHandleBinaryFunction
func goCallHandleBinaryFunction(ctx unsafe.Pointer, t *C.char, l C.int) C.int {
	handler := handler_pointer_map.Get((*C.P4GoHandler)(ctx))
	if handler != nil {
		return C.int((*handler).HandleBinary(C.GoBytes(unsafe.Pointer(t), l)))
	}
	return C.int(0)
}

//export goCallHandleMessageFunction
func goCallHandleMessageFunction(ctx unsafe.Pointer, e *C.Error) C.int {
	handler := handler_pointer_map.Get((*C.P4GoHandler)(ctx))
	if handler != nil {
		err := P4Message{}
		err.severity = P4MessageSeverity(C.GetErrorSeverity(e))
		ec := int(C.GetErrorCount(e))
		err.lines = []P4MessageLine{}
		for j := 0; j < ec; j++ {
			el := P4MessageLine{}
			el.severity = P4MessageSeverity(C.GetErrorSeverityI(e, C.int(j)))
			s := C.FmtError(e, C.int(j))
			el.fmt = C.GoString(s)
			C.free(unsafe.Pointer(s))
			el.code = int(C.GetErrorCode(e, C.int(j)))
			err.lines = append(err.lines, el)
		}
		return C.int((*handler).HandleMessage(err))
	}
	return C.int(0)
}

//export goCallHandleStatFunction
func goCallHandleStatFunction(ctx unsafe.Pointer, t *C.StrDict) C.int {
	handler := handler_pointer_map.Get((*C.P4GoHandler)(ctx))
	if handler != nil {
		dict := Dictionary{}
		i := 0
		k := (*C.char)(C.malloc(C.size_t(1)))
		v := (*C.char)(C.malloc(C.size_t(1)))
		for C.StrDictGetKeyPair(t, C.int(i), &k, &v) != 0 {
			i++
			dict[C.GoString(k)] = C.GoString(v)
		}
		C.free(unsafe.Pointer(k))
		C.free(unsafe.Pointer(v))
		return C.int((*handler).HandleStat(dict))
	}
	return C.int(0)
}

//export goCallHandleSpecFunction
func goCallHandleSpecFunction(ctx unsafe.Pointer, t *C.P4GoSpecData) C.int {
	handler := handler_pointer_map.Get((*C.P4GoHandler)(ctx))
	if handler != nil {
		dict := Dictionary{}
		i := 0
		k := (*C.char)(C.malloc(C.size_t(1)))
		v := (*C.char)(C.malloc(C.size_t(1)))
		for C.SpecDataGetKeyPair(t, C.int(i), &k, &v) != 0 {
			i++
			dict[C.GoString(k)] = C.GoString(v)
		}
		C.free(unsafe.Pointer(k))
		C.free(unsafe.Pointer(v))
		return C.int((*handler).HandleSpec(dict))
	}
	return C.int(0)
}

//export goCallHandleTextFunction
func goCallHandleTextFunction(ctx unsafe.Pointer, t *C.char) C.int {
	handler := handler_pointer_map.Get((*C.P4GoHandler)(ctx))
	if handler != nil {
		return C.int((*handler).HandleText(C.GoString(t)))
	}
	return C.int(0)
}

//export goCallHandleTrackFunction
func goCallHandleTrackFunction(ctx unsafe.Pointer, t *C.char) C.int {
	handler := handler_pointer_map.Get((*C.P4GoHandler)(ctx))
	if handler != nil {
		return C.int((*handler).HandleTrack(C.GoString(t)))
	}
	return C.int(0)
}

type P4SSOResult int

const (
	P4SSO_PASS  P4SSOResult = iota // SSO succeeded (result is an authentication token)
	P4SSO_FAIL                     // SSO failed (result will be logged as error message)
	P4SSO_UNSET                    // Client has no SSO support
	P4SSO_EXIT                     // Stop login process
	P4SSO_SKIP                     // Fall back to default P4API behavior
)

func (s P4SSOResult) String() string {
	switch s {
	case P4SSO_PASS:
		return "pass"
	case P4SSO_FAIL:
		return "fail"
	case P4SSO_UNSET:
		return "unset"
	case P4SSO_EXIT:
		return "exit"
	case P4SSO_SKIP:
		return "skip"
	}
	return "unknown"
}

type P4SSOHandler interface {
	Authorize(vars Dictionary, maxLength int) (P4SSOResult, string)
}

func (p4 *P4) SetSSOHandler(handler P4SSOHandler) {
	p := C.GetSSOHandler(p4.handle)
	if p != nil {
		ssohandler_pointer_map.Delete(p)
		C.FreeSSOHandler(p)
	}
	p4.ssohandle = nil
	if handler != nil {
		p4.ssohandle = C.NewSSOHandler()
		ssohandler_pointer_map.Set(p4.ssohandle, &handler)
		C.SetSSOHandler(p4.handle, p4.ssohandle)
	} else {
		C.SetSSOHandler(p4.handle, nil)
	}
}

//export goCallSSOAuthorizeFunction
func goCallSSOAuthorizeFunction(ctx unsafe.Pointer, d *C.StrDict, l C.int, r **C.char) C.int {
	ssohandler := ssohandler_pointer_map.Get((*C.P4GoSSOHandler)(ctx))
	if ssohandler != nil {
		dict := Dictionary{}
		i := 0
		k := (*C.char)(C.malloc(C.size_t(1)))
		v := (*C.char)(C.malloc(C.size_t(1)))
		for C.StrDictGetKeyPair(d, C.int(i), &k, &v) != 0 {
			i++
			dict[C.GoString(k)] = C.GoString(v)
		}
		C.free(unsafe.Pointer(k))
		C.free(unsafe.Pointer(v))

		status, ret := (*ssohandler).Authorize(dict, int(l))
		*r = C.CString(ret)
		return C.int(status)
	}
	return C.int(P4SSO_SKIP)
}

type P4MergeStatus int

const (
	P4MD_QUIT   P4MergeStatus = iota // user wants to quit
	P4MD_SKIP                        // skip the integration record
	P4MD_MERGED                      // accepted merged theirs and yours
	P4MD_EDIT                        // accepted edited merge
	P4MD_THEIRS                      // accepted theirs
	P4MD_YOURS                       // accepted yours
)

type P4ResolveHandler interface {
	Resolve(md P4MergeData) P4MergeStatus
}

func (p4 *P4) SetResolveHandler(handler P4ResolveHandler) {
	p := C.GetResolveHandler(p4.handle)
	if p != nil {
		resolvehandler_pointer_map.Delete(p)
		C.FreeResolveHandler(p)
	}
	p4.outputhandle = nil
	if handler != nil {
		p4.resolvehandle = C.NewResolveHandler()
		resolvehandler_pointer_map.Set(p4.resolvehandle, &handler)
		C.SetResolveHandler(p4.handle, p4.resolvehandle)
	} else {
		C.SetResolveHandler(p4.handle, nil)
	}
}

//export goCallResolveFunction
func goCallResolveFunction(ctx unsafe.Pointer, t *C.P4GoMergeData) C.int {
	handler := resolvehandler_pointer_map.Get((*C.P4GoResolveHandler)(ctx))
	if handler != nil {
		return C.int((*handler).Resolve(P4MergeData{handle: t}))
	}
	return C.int(P4MD_QUIT)
}

//
// MapApi
//

type P4MapDirection int

const (
	P4MAP_LEFT_RIGHT P4MapDirection = iota
	P4MAP_RIGHT_LEFT
)

type P4MapType int

const (
	P4MAP_INCLUDE P4MapType = iota
	P4MAP_EXCLUDE
	P4MAP_OVERLAY
	P4MAP_ONETOMANY
)

type P4MapCaseSensitivity int

const (
	P4MAP_CASE_SENSITIVE P4MapCaseSensitivity = iota
	P4MAP_CASE_INSENSITIVE
)

type P4Map struct {
	handle *C.MapApi
}

func NewMap() *P4Map {
	p := C.NewMapApi()
	ret := &P4Map{handle: p}
	return ret
}

func (mapapi *P4Map) Close() {
	C.FreeMapApi(mapapi.handle)
}

func JoinMap(m1 *P4Map, m2 *P4Map) *P4Map {
	p := C.JoinMapApi(m1.handle, m2.handle)
	ret := &P4Map{handle: p}
	return ret
}

func (mapapi *P4Map) Insert(lhs string, rhs string, flag P4MapType) {
	c_lhs := C.CString(lhs)
	c_rhs := C.CString(rhs)
	C.MapApiInsert(mapapi.handle, c_lhs, c_rhs, C.int(flag))
	C.free(unsafe.Pointer(c_lhs))
	C.free(unsafe.Pointer(c_rhs))
}

func (mapapi *P4Map) String() string {
	return strings.Join(mapapi.Array(), "\n")
}

func (mapapi *P4Map) Array() []string {
	a := []string{}
	c := mapapi.Count()
	for i := 0; i < c; i++ {
		l := mapapi.Lhs(i)
		r := mapapi.Rhs(i)
		f := mapapi.Type(i)

		s := ""
		switch f {
		case P4MAP_INCLUDE:
			break
		case P4MAP_EXCLUDE:
			s += "-"
		case P4MAP_OVERLAY:
			s += "+"
		case P4MAP_ONETOMANY:
			s += "&"
		}
		s += l + " " + r
		a = append(a, s)
	}
	return a
}

func (mapapi *P4Map) Clear() {
	C.MapApiClear(mapapi.handle)
}

func (mapapi *P4Map) Count() int {
	return int(C.MapApiCount(mapapi.handle))
}

func (mapapi *P4Map) Reverse() {
	mapapi.handle = C.MapApiReverse(mapapi.handle)
}

func (mapapi *P4Map) Translate(input string, dir P4MapDirection) string {
	s := C.CString(input)
	res := C.MapApiTranslate(mapapi.handle, s, C.int(dir))
	C.free(unsafe.Pointer(s))

	if res == nil {
		return ""
	}
	rstr := C.GoString(res)
	C.free(unsafe.Pointer(res))
	return rstr
}

func (mapapi *P4Map) TranslateArray(input string, dir P4MapDirection) []string {
	s := C.CString(input)
	l := C.int(0)
	res := C.MapApiTranslateArray(mapapi.handle, s, C.int(dir), &l)
	C.free(unsafe.Pointer(s))

	rarr := []string{}
	if res == nil {
		return rarr
	}
	for _, v := range unsafe.Slice(res, l) {
		rarr = append(rarr, C.GoString(v))
		C.free(unsafe.Pointer(v))
	}
	C.free(unsafe.Pointer(res))
	return rarr
}

func (mapapi *P4Map) Lhs(i int) string {
	res := C.MapApiLhs(mapapi.handle, C.int(i))
	if res == nil {
		return ""
	}
	return C.GoString(res)
}

func (mapapi *P4Map) Rhs(i int) string {
	res := C.MapApiRhs(mapapi.handle, C.int(i))
	if res == nil {
		return ""
	}
	return C.GoString(res)
}
func (mapapi *P4Map) Type(i int) P4MapType {
	return (P4MapType)(int(C.MapApiType(mapapi.handle, C.int(i))))
}

//
// P4ResolveData
//

type P4MergeData struct {
	handle      *C.P4GoMergeData
	actionType  *P4Message
	yourAction  *P4Message
	theirAction *P4Message
	mergeAction *P4Message
}

func (md *P4MergeData) YourName() string {
	return C.GoString(C.MergeDataGetYourName(md.handle))
}

func (md *P4MergeData) TheirName() string {
	return C.GoString(C.MergeDataGetTheirName(md.handle))
}

func (md *P4MergeData) BaseName() string {
	return C.GoString(C.MergeDataGetBaseName(md.handle))
}

func (md *P4MergeData) YourPath() string {
	return C.GoString(C.MergeDataGetYourPath(md.handle))
}

func (md *P4MergeData) TheirPath() string {
	return C.GoString(C.MergeDataGetTheirPath(md.handle))
}

func (md *P4MergeData) BasePath() string {
	return C.GoString(C.MergeDataGetBasePath(md.handle))
}

func (md *P4MergeData) ResultPath() string {
	return C.GoString(C.MergeDataGetResultPath(md.handle))
}

func (md *P4MergeData) MergeHint() P4MergeStatus {
	return P4MergeStatus(int(C.MergeDataGetMergeHint(md.handle)))
}

func (md *P4MergeData) RunMerge() bool {
	return int(C.MergeDataRunMergeTool(md.handle)) != 0
}

func (md *P4MergeData) IsActionResolve() bool {
	return int(C.MergeDataGetActionResolveStatus(md.handle)) != 0

}

func (md *P4MergeData) IsContentResolve() bool {
	return int(C.MergeDataGetContentResolveStatus(md.handle)) != 0

}

func (md *P4MergeData) ActionType() *P4Message {
	if md.actionType != nil {
		return md.actionType
	}

	e := C.MergeDataGetType(md.handle)
	md.actionType = &P4Message{}
	md.actionType.severity = P4MessageSeverity(C.GetErrorSeverity(e))
	ec := int(C.GetErrorCount(e))
	md.actionType.lines = []P4MessageLine{}
	for j := 0; j < ec; j++ {
		el := P4MessageLine{}
		el.severity = P4MessageSeverity(C.GetErrorSeverityI(e, C.int(j)))
		s := C.FmtError(e, C.int(j))
		el.fmt = C.GoString(s)
		C.free(unsafe.Pointer(s))
		el.code = int(C.GetErrorCode(e, C.int(j)))
		md.actionType.lines = append(md.actionType.lines, el)
	}
	return md.actionType
}

func (md *P4MergeData) Info() int {
	//ToDo: should be the last 2 stat/text results?
	return 0
}

func (md *P4MergeData) MergeAction() *P4Message {
	if md.mergeAction != nil {
		return md.mergeAction
	}

	e := C.MergeDataGetMergeAction(md.handle)
	md.mergeAction = &P4Message{}
	md.mergeAction.severity = P4MessageSeverity(C.GetErrorSeverity(e))
	ec := int(C.GetErrorCount(e))
	md.mergeAction.lines = []P4MessageLine{}
	for j := 0; j < ec; j++ {
		el := P4MessageLine{}
		el.severity = P4MessageSeverity(C.GetErrorSeverityI(e, C.int(j)))
		s := C.FmtError(e, C.int(j))
		el.fmt = C.GoString(s)
		C.free(unsafe.Pointer(s))
		el.code = int(C.GetErrorCode(e, C.int(j)))
		md.mergeAction.lines = append(md.mergeAction.lines, el)
	}
	return md.mergeAction
}

func (md *P4MergeData) TheirAction() *P4Message {
	if md.theirAction != nil {
		return md.theirAction
	}

	e := C.MergeDataGetTheirAction(md.handle)
	md.theirAction = &P4Message{}
	md.theirAction.severity = P4MessageSeverity(C.GetErrorSeverity(e))
	ec := int(C.GetErrorCount(e))
	md.theirAction.lines = []P4MessageLine{}
	for j := 0; j < ec; j++ {
		el := P4MessageLine{}
		el.severity = P4MessageSeverity(C.GetErrorSeverityI(e, C.int(j)))
		s := C.FmtError(e, C.int(j))
		el.fmt = C.GoString(s)
		C.free(unsafe.Pointer(s))
		el.code = int(C.GetErrorCode(e, C.int(j)))
		md.theirAction.lines = append(md.theirAction.lines, el)
	}
	return md.theirAction
}

func (md *P4MergeData) YourAction() *P4Message {
	if md.yourAction != nil {
		return md.yourAction
	}

	e := C.MergeDataGetYoursAction(md.handle)
	md.yourAction = &P4Message{}
	md.yourAction.severity = P4MessageSeverity(C.GetErrorSeverity(e))
	ec := int(C.GetErrorCount(e))
	md.yourAction.lines = []P4MessageLine{}
	for j := 0; j < ec; j++ {
		el := P4MessageLine{}
		el.severity = P4MessageSeverity(C.GetErrorSeverityI(e, C.int(j)))
		s := C.FmtError(e, C.int(j))
		el.fmt = C.GoString(s)
		C.free(unsafe.Pointer(s))
		el.code = int(C.GetErrorCode(e, C.int(j)))
		md.yourAction.lines = append(md.yourAction.lines, el)
	}
	return md.yourAction
}

func (md *P4MergeData) String() string {
	s := C.MergeDataGetString(md.handle)
	r := C.GoString(s)
	C.free(unsafe.Pointer(s))
	return r
}

// P4DepotFile represents a file in the depot.
type P4DepotFile struct {
	Name      string
	Revisions []*P4Revision
}

// P4Revision represents a single revision of a P4DepotFile.
type P4Revision struct {
	Rev          int
	Change       int
	Action       string
	Type         string
	Time         time.Time
	User         string
	Client       string
	Desc         string
	Digest       string
	FileSize     string
	Integrations []P4Integration
}

// P4Integration represents an integration record for a revision.
type P4Integration struct {
	How  string
	File string
	SRev int
	ERev int
}

// NewDepotFile creates a new P4DepotFile.
func NewDepotFile(name string) *P4DepotFile {
	return &P4DepotFile{Name: name, Revisions: []*P4Revision{}}
}

// NewRevision creates a new Revision for a DepotFile.
func (df *P4DepotFile) NewRevision() *P4Revision {
	rev := &P4Revision{}
	df.Revisions = append(df.Revisions, rev)
	return rev
}

// AddIntegration adds an integration record to a Revision.
func (r *P4Revision) AddIntegration(how, file string, srev, erev int) {
	r.Integrations = append(r.Integrations, P4Integration{How: how, File: file, SRev: srev, ERev: erev})
}

func ProcessFilelog(h map[string]interface{}) (*P4DepotFile, error) {
	depotFile, ok := h["depotFile"].(string)
	if !ok {
		return nil, errors.New("not a filelog object: missing depotFile")
	}

	df := NewDepotFile(depotFile)

	// Parse revisions from the map
	revisionCount := 0
	for key := range h {
		if strings.HasPrefix(key, "rev") {
			revisionCount++
		}
	}

	for n := 0; n < revisionCount; n++ {
		r := df.NewRevision()

		// Parse scalar attributes for the revision
		if rev, ok := h[fmt.Sprintf("rev%d", n)].(string); ok {
			r.Rev, _ = strconv.Atoi(rev)
		}
		if change, ok := h[fmt.Sprintf("change%d", n)].(string); ok {
			r.Change, _ = strconv.Atoi(change)
		}
		if action, ok := h[fmt.Sprintf("action%d", n)].(string); ok {
			r.Action = action
		}
		if fileType, ok := h[fmt.Sprintf("type%d", n)].(string); ok {
			r.Type = fileType
		}
		if timeStr, ok := h[fmt.Sprintf("time%d", n)].(string); ok {
			timeInt, _ := strconv.ParseInt(timeStr, 10, 64)
			r.Time = time.Unix(timeInt, 0)
		}
		if user, ok := h[fmt.Sprintf("user%d", n)].(string); ok {
			r.User = user
		}
		if client, ok := h[fmt.Sprintf("client%d", n)].(string); ok {
			r.Client = client
		}
		if desc, ok := h[fmt.Sprintf("desc%d", n)].(string); ok {
			r.Desc = desc
		}
		if digest, ok := h[fmt.Sprintf("digest%d", n)].(string); ok {
			r.Digest = digest
		}
		if fileSize, ok := h[fmt.Sprintf("fileSize%d", n)].(string); ok {
			r.FileSize = fileSize
		}

		// Parse integration records for the revision
		for m := 0; ; m++ {
			howKey := fmt.Sprintf("how%d,%d", n, m)
			fileKey := fmt.Sprintf("file%d,%d", n, m)
			srevKey := fmt.Sprintf("srev%d,%d", n, m)
			erevKey := fmt.Sprintf("erev%d,%d", n, m)

			how, howOk := h[howKey].(string)
			file, fileOk := h[fileKey].(string)
			srev, srevOk := h[srevKey].(string)
			erev, erevOk := h[erevKey].(string)

			if !howOk || !fileOk || !srevOk || !erevOk {
				break
			}

			srevInt := parseRevision(srev)
			erevInt := parseRevision(erev)

			r.AddIntegration(how, file, srevInt, erevInt)
		}
	}

	return df, nil
}

// parseRevision parses a revision string and converts it to an integer.
func parseRevision(rev string) int {
	if rev == "none" || rev == "#none" {
		return 0
	}
	rev = strings.TrimPrefix(rev, "#")
	parsedRev, _ := strconv.Atoi(rev)
	return parsedRev
}

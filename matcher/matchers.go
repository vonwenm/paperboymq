/*
Copyright 2015 Dariusz Górecki <darek.krk@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package matcher

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/canni/paperboymq/amq"
)

// Direct matcher matches if both message routing key and binding key are equal.
var Direct = amq.MatchFunc(func(msg amq.Message, binding amq.Binding) bool {
	return msg.RoutingKey() == binding.BindingKey()
})

// Fanout matcher always matches
var Fanout = amq.MatchFunc(func(msg amq.Message, binding amq.Binding) bool {
	return true
})

var patternCache = struct {
	cache map[string]*regexp.Regexp
	sync.RWMutex
}{
	cache: make(map[string]*regexp.Regexp),
}

// Topic matcher matches when routing key matches binding pattern, see AMQP
// specification for detailed information. There is no need to rewrite all
// rules here.
var Topic = amq.MatchFunc(func(msg amq.Message, binding amq.Binding) bool {
	patternCache.RLock()
	defer patternCache.RUnlock()

	if matcher, found := patternCache.cache[binding.BindingKey()]; found {
		return matcher.MatchString(msg.RoutingKey())
	} else {
		pattern := strings.Replace(binding.BindingKey(), ".", `\.`, -1)
		pattern = strings.Replace(pattern, "*", `[0-9A-z]+`, -1)
		pattern = strings.Replace(pattern, `\.#\.`, `(\.|\.[0-9A-z\.]*\.)`, -1)
		pattern = strings.Replace(pattern, `\.#`, `(\.[0-9A-z\.]*)?`, -1)
		pattern = strings.Replace(pattern, `#\.`, `([0-9A-z\.]*\.)?`, -1)
		pattern = strings.Replace(pattern, "#", `[0-9A-z\.]*`, -1)

		matcher := regexp.MustCompile(fmt.Sprintf("^%s$", pattern))

		go func() {
			patternCache.Lock()
			defer patternCache.Unlock()

			patternCache.cache[binding.BindingKey()] = matcher
		}()

		return matcher.MatchString(msg.RoutingKey())
	}
})
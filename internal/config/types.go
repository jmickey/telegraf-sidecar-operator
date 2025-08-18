/*
Copyright 2024 Josh Michielsen.

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

package config

import (
	"fmt"
	"strconv"
)

type OptionalInt64 struct {
	value *int64
}

func (o *OptionalInt64) String() string {
	if o.value == nil {
		return ""
	}
	return fmt.Sprintf("%d", *o.value)
}

func (o *OptionalInt64) Set(s string) error {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	o.value = &v
	return nil
}

func (o *OptionalInt64) Value() *int64 {
	return o.value
}

type OptionalBool struct {
	value *bool
}

func (o *OptionalBool) String() string {
	if o.value == nil {
		return ""
	}
	return fmt.Sprintf("%t", *o.value)
}

func (o *OptionalBool) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	o.value = &v
	return nil
}

func (o *OptionalBool) Value() *bool {
	return o.value
}

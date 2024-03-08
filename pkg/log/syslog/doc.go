// Copyright 2023 Hedgehog
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// The implementation for the encoder has been taken and adopted from https://github.com/imperfectgo/zap-syslog
// which at the time of this writing (3/9/23) was under the MIT license. As it looks like that
// the author is not maintaining the library, and because this is not a go module yet, it is the
// best course of action to import the necessary code into this package here.
package syslog

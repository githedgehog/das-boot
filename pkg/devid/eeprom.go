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

package devid

import "fmt"

// Tlv represents an ONIE TLV
type Tlv uint8

const (
	TlvSerial Tlv = 0x23
)

func (t Tlv) String() string {
	return fmt.Sprintf("0x%2x", uint8(t))
}

# Copyright Splunk Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

function _panic() {
  >&2 echo "$1"
  exit 1
}

export ROLE_NAME=$(whoami)-extension-demo
export BUFFERED_FUNCTION_NAME=$(whoami)-signalfx-extension-demo-buffered
export REAL_TIME_FUNCTION_NAME=$(whoami)-signalfx-extension-demo-realm-time

export FUNCTION_RUNTIME=${FUNCTION_RUNTIME:-nodejs12.x}
export FUNCTION_CODE=${FUNCTION_CODE:-UEsDBBQAAAAIAJRsYVJ3m87PcAAAAHgAAAAIABwAaW5kZXguanNVVAkAA9jfPGDZ3zxgdXgLAAEE9QEAAAQUAAAAS60oyC8qKdbLSMxLyUktUrBVSCyuzEtW0NBUsLVTqOZSUChKLSktygMzFRSKSxJLSoud81NSrRSMDAx0wIJJ+SmVVgpewf5+esUlRZl56ZlplRrqHqk5OfkKaUX5uQo+iblJKYmK6pog9bXWXEAEAFBLAQIeAxQAAAAIAJRsYVJ3m87PcAAAAHgAAAAIABgAAAAAAAEAAACkgQAAAABpbmRleC5qc1VUBQAD2N88YHV4CwABBPUBAAAEFAAAAFBLBQYAAAAAAQABAE4AAACyAAAAAAA=}

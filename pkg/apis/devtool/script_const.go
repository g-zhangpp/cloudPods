// Copyright 2019 Yunion
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

package devtool

const (
	SCRIPT_APPLY_STATUS_APPLYING     = "applying"
	SCRIPT_APPLY_STATUS_APPLY_FAILED = "apply_failed"
	SCRIPT_APPLY_STATUS_READY        = "ready"

	SCRIPT_APPLY_RECORD_APPLYING = "applying"
	SCRIPT_APPLY_RECORD_SUCCEED  = "succeed"
	SCRIPT_APPLY_RECORD_FAILED   = "failed"

	SCRIPT_APPLY_RECORD_FAILCODE_SSHABLE  = "ServerNotSshable"
	SCRIPT_APPLY_RECORD_FAILCODE_INFLUXDB = "NoReachInfluxdb"
	SCRIPT_APPLY_RECORD_FAILCODE_OTHERS   = "Others"

	SCRIPT_NAME  = "monitor agent"
	SERVICE_TYPE = "devtool"

	SCRIPT_STATUS_READY = "ready"
)

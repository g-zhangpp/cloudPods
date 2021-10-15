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

package shell

import (
	"yunion.io/x/onecloud/pkg/multicloud/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type DBInstanceListOptions struct {
		Id     string
		Offset int
		Limit  int
	}
	shellutils.R(&DBInstanceListOptions{}, "dbinstance-list", "List rds intances", func(cli *aws.SRegion, args *DBInstanceListOptions) error {
		instances, err := cli.GetDBInstances(args.Id)
		if err != nil {
			return err
		}
		printList(instances, 0, args.Offset, args.Limit, []string{})
		return nil
	})

	type DBInstanceIdOptions struct {
		ID string
	}

	shellutils.R(&DBInstanceIdOptions{}, "dbinstance-show", "Show rds intance", func(cli *aws.SRegion, args *DBInstanceIdOptions) error {
		instance, err := cli.GetDBInstance(args.ID)
		if err != nil {
			return err
		}
		printObject(instance)
		return nil
	})

}

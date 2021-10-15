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

package models

import (
	"testing"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
)

func TestSSnapshotPolicy_Key(t *testing.T) {
	cases := []struct {
		in   *SSnapshotPolicy
		want uint64
	}{
		{
			&SSnapshotPolicy{
				RepeatWeekdays: 0,
				TimePoints:     0,
				RetentionDays:  7,
				IsActivated:    tristate.True,
			},
			15 + 2,
		},
		{
			&SSnapshotPolicy{
				RepeatWeekdays: 11,
			},
			11<<56 + 2,
		},
		{
			&SSnapshotPolicy{
				TimePoints: 234,
			},
			234<<24 + 2,
		},
		{
			&SSnapshotPolicy{
				RetentionDays: -1,
			},
			0,
		},
		{
			&SSnapshotPolicy{
				RepeatWeekdays: 13,
				TimePoints:     123,
				RetentionDays:  7,
			},
			936748724556660752,
		},
	}

	for i, c := range cases {
		g := c.in.Key()
		if c.want != g {
			log.Errorf("the %d case, want %d, get %d", i, c.want, g)
		}
	}
}

func TestSSnapshotPolicy_ComputeNextSyncTime(t *testing.T) {
	timeStr := "2006-01-02 15:04:05"
	t.Run("base test", func(t *testing.T) {
		cases := []struct {
			in   *SSnapshotPolicy
			base string
			want string
		}{
			{
				in: &SSnapshotPolicy{
					RepeatWeekdays: SnapshotPolicyManager.RepeatWeekdaysParseIntArray([]int{2}),
					TimePoints:     SnapshotPolicyManager.TimePointsParseIntArray([]int{4}),
				},
				base: "2020-10-31 00:00:00",
				want: "2020-11-03 04:00:00",
			},
			{
				in: &SSnapshotPolicy{
					RepeatWeekdays: SnapshotPolicyManager.RepeatWeekdaysParseIntArray([]int{5, 7}),
					TimePoints:     SnapshotPolicyManager.TimePointsParseIntArray([]int{2, 6}),
				},
				base: "2020-10-31 00:00:00",
				want: "2020-11-01 02:00:00",
			},
		}
		for _, c := range cases {
			base, _ := time.Parse(timeStr, c.base)
			want, _ := time.Parse(timeStr, c.want)
			real := c.in.ComputeNextSyncTime(base)
			if want != real {
				t.Fatalf("want: %s, real: %s", want, real)
			}
		}
	})
	t.Run("same day", func(t *testing.T) {
		cases := []struct {
			in   *SSnapshotPolicy
			base string
			want string
		}{
			{
				in: &SSnapshotPolicy{
					RepeatWeekdays: SnapshotPolicyManager.RepeatWeekdaysParseIntArray([]int{6, 7}),
					TimePoints:     SnapshotPolicyManager.TimePointsParseIntArray([]int{2}),
				},
				base: "2020-10-31 00:00:00",
				want: "2020-10-31 02:00:00",
			},
			{
				in: &SSnapshotPolicy{
					RepeatWeekdays: SnapshotPolicyManager.RepeatWeekdaysParseIntArray([]int{6, 7}),
					TimePoints:     SnapshotPolicyManager.TimePointsParseIntArray([]int{2}),
				},
				base: "2020-10-31 02:00:00",
				want: "2020-11-01 02:00:00",
			},
			{
				in: &SSnapshotPolicy{
					RepeatWeekdays: SnapshotPolicyManager.RepeatWeekdaysParseIntArray([]int{6, 7}),
					TimePoints:     SnapshotPolicyManager.TimePointsParseIntArray([]int{2}),
				},
				base: "2020-10-31 01:00:00",
				want: "2020-10-31 02:00:00",
			},
			{
				in: &SSnapshotPolicy{
					RepeatWeekdays: SnapshotPolicyManager.RepeatWeekdaysParseIntArray([]int{6}),
					TimePoints:     SnapshotPolicyManager.TimePointsParseIntArray([]int{2, 4}),
				},
				base: "2020-10-31 05:00:00",
				want: "2020-11-07 02:00:00",
			},
			{
				in: &SSnapshotPolicy{
					RepeatWeekdays: SnapshotPolicyManager.RepeatWeekdaysParseIntArray([]int{1, 2, 3, 4, 5, 6, 7}),
					TimePoints:     SnapshotPolicyManager.TimePointsParseIntArray([]int{0}),
				},
				base: "2020-10-31 05:00:00",
				want: "2020-11-01 00:00:00",
			},
		}
		for _, c := range cases {
			base, _ := time.Parse(timeStr, c.base)
			want, _ := time.Parse(timeStr, c.want)
			real := c.in.ComputeNextSyncTime(base)
			if want != real {
				t.Fatalf("want: %s, real: %s", want, real)
			}
		}
	})
	t.Run("retentionday", func(t *testing.T) {
		cases := []struct {
			in   *SSnapshotPolicy
			base string
			want string
		}{
			{
				in: &SSnapshotPolicy{
					RepeatWeekdays: SnapshotPolicyManager.RepeatWeekdaysParseIntArray([]int{5}),
					TimePoints:     SnapshotPolicyManager.TimePointsParseIntArray([]int{4}),
					RetentionDays:  2,
				},
				base: "2020-10-31 04:00:00",
				want: "2020-11-01 04:00:00",
			},
			{
				in: &SSnapshotPolicy{
					RepeatWeekdays: SnapshotPolicyManager.RepeatWeekdaysParseIntArray([]int{6}),
					TimePoints:     SnapshotPolicyManager.TimePointsParseIntArray([]int{4}),
					RetentionDays:  8,
				},
				base: "2020-10-31 04:00:00",
				want: "2020-11-01 04:00:00",
			},
			{
				in: &SSnapshotPolicy{
					RepeatWeekdays: SnapshotPolicyManager.RepeatWeekdaysParseIntArray([]int{1, 6}),
					TimePoints:     SnapshotPolicyManager.TimePointsParseIntArray([]int{4}),
					RetentionDays:  4,
				},
				base: "2020-10-31 04:00:00",
				want: "2020-11-02 04:00:00",
			},
		}
		for _, c := range cases {
			base, _ := time.Parse(timeStr, c.base)
			want, _ := time.Parse(timeStr, c.want)
			real := c.in.ComputeNextSyncTime(base)
			if want != real {
				t.Fatalf("want: %s, real: %s", want, real)
			}
		}
	})
}

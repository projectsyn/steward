package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestProcessOpenshiftVersion(t *testing.T) {
	tcs := map[string]struct {
		in   OpenshiftVersion
		out  OpenshiftVersionFact
		fail bool
	}{
		"standard": {
			in: OpenshiftVersion{Status: OpenshiftVersionStatus{
				Desired: OpenshiftVersionDesired{
					Version: "4.8.11",
				},
				History: []OpenshiftVersionHistory{
					{
						State:          "Completed",
						Verified:       true,
						Version:        "4.8.11",
						CompletionTime: mustTime("2021-10-15T08:41:44Z"),
					},
				},
			}},
			out: OpenshiftVersionFact{
				Version: SemanticVersion{
					Major: "4",
					Minor: "8",
					Patch: "11",
				},
				DesiredVersion: SemanticVersion{
					Major: "4",
					Minor: "8",
					Patch: "11",
				},
			},
			fail: false,
		},
		"notUpdated": {
			in: OpenshiftVersion{Status: OpenshiftVersionStatus{
				Desired: OpenshiftVersionDesired{
					Version: "4.8.11",
				},
				History: []OpenshiftVersionHistory{
					{
						State:          "Partial",
						Verified:       true,
						Version:        "4.8.4",
						CompletionTime: mustTime("2021-10-13T08:41:44Z"),
					},
					{
						State:          "Completed",
						Verified:       false,
						Version:        "4.8.2",
						CompletionTime: mustTime("2021-10-12T08:41:44Z"),
					},
					{
						State:          "Completed",
						Verified:       true,
						Version:        "4.7.4",
						CompletionTime: mustTime("2021-10-11T08:41:44Z"),
					},
				},
			}},
			out: OpenshiftVersionFact{
				Version: SemanticVersion{
					Major: "4",
					Minor: "7",
					Patch: "4",
				},
				DesiredVersion: SemanticVersion{
					Major: "4",
					Minor: "8",
					Patch: "11",
				},
			},
			fail: false,
		},
		"notInOrder": {
			in: OpenshiftVersion{Status: OpenshiftVersionStatus{
				Desired: OpenshiftVersionDesired{
					Version: "4.8.11",
				},
				History: []OpenshiftVersionHistory{
					{
						State:          "Completed",
						Verified:       true,
						Version:        "4.8.2",
						CompletionTime: mustTime("2021-10-12T08:41:44Z"),
					},
					{
						State:          "Completed",
						Verified:       true,
						Version:        "4.8.4",
						CompletionTime: mustTime("2021-10-13T08:41:44Z"),
					},
					{
						State:          "Completed",
						Verified:       true,
						Version:        "4.7.4",
						CompletionTime: mustTime("2021-10-11T08:41:44Z"),
					},
				},
			}},
			out: OpenshiftVersionFact{
				Version: SemanticVersion{
					Major: "4",
					Minor: "8",
					Patch: "4",
				},
				DesiredVersion: SemanticVersion{
					Major: "4",
					Minor: "8",
					Patch: "11",
				},
			},
			fail: false,
		},
		"missingCurrent": {
			in: OpenshiftVersion{Status: OpenshiftVersionStatus{
				Desired: OpenshiftVersionDesired{
					Version: "4.8.11",
				},
				History: []OpenshiftVersionHistory{},
			}},
			out: OpenshiftVersionFact{
				Version: SemanticVersion{},
				DesiredVersion: SemanticVersion{
					Major: "4",
					Minor: "8",
					Patch: "11",
				},
			},
			fail: true,
		},
		"invalidVersion": {
			in: OpenshiftVersion{Status: OpenshiftVersionStatus{
				Desired: OpenshiftVersionDesired{
					Version: "a.8.11",
				},
				History: []OpenshiftVersionHistory{
					{
						State:    "Partial",
						Verified: true,
						Version:  "4.8.4",
					},
					{
						State:    "Completed",
						Verified: false,
						Version:  "4.8.2",
					},
					{
						State:    "Completed",
						Verified: true,
						Version:  "4.7.4",
					},
				},
			}},
			out:  OpenshiftVersionFact{},
			fail: true,
		},
	}
	for k, tc := range tcs {
		t.Run(k, func(t *testing.T) {
			v, err := processOpenshiftVersion(tc.in)
			if tc.fail {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.out, *v)
		})
	}
}

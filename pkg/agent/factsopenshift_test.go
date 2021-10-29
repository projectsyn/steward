package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)


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
						State:    "Completed",
						Verified: true,
						Version:  "4.8.11",
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

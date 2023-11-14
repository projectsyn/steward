package facts

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		out  SemanticVersion
		fail bool
	}{
		"standard": {
			in: OpenshiftVersion{Status: OpenshiftVersionStatus{
				Desired: OpenshiftVersionDesired{
					Version: "4.8.13",
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
			out: SemanticVersion{
				Major: "4",
				Minor: "8",
				Patch: "11",
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
			out: SemanticVersion{
				Major: "4",
				Minor: "7",
				Patch: "4",
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
			out: SemanticVersion{
				Major: "4",
				Minor: "8",
				Patch: "4",
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
			out: SemanticVersion{
				Major: "4",
				Minor: "8",
				Patch: "11",
			},
			fail: false,
		},
		"invalidDesiredVersion": {
			in: OpenshiftVersion{Status: OpenshiftVersionStatus{
				Desired: OpenshiftVersionDesired{
					Version: "a.8.11",
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
			out: SemanticVersion{
				Major: "4",
				Minor: "8",
				Patch: "4",
			},
			fail: false,
		},
		"invalidVersions": {
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
						Version:  "s4.l.4",
					},
				},
			}},
			out:  SemanticVersion{},
			fail: true,
		},
		"empty": {
			in: OpenshiftVersion{Status: OpenshiftVersionStatus{
				Desired: OpenshiftVersionDesired{},
				History: []OpenshiftVersionHistory{},
			}},
			out:  SemanticVersion{},
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
			require.NotNil(t, v)
			assert.Equal(t, tc.out, *v)
		})
	}
}

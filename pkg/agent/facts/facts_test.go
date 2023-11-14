package facts

import (
	"testing"

	"k8s.io/apimachinery/pkg/version"

	"github.com/stretchr/testify/assert"
)

func TestProcessKubernetesVersion(t *testing.T) {
	tcs := map[string]struct {
		in   version.Info
		out  version.Info
		fail bool
	}{
		"standard": {
			in:   version.Info{Major: "1", Minor: "20"},
			out:  version.Info{Major: "1", Minor: "20"},
			fail: false,
		},
		"eks/gce": {
			in:   version.Info{Major: "1", Minor: "20+"},
			out:  version.Info{Major: "1", Minor: "20"},
			fail: false,
		},
		"strange major": {
			in:   version.Info{Major: "1+", Minor: "20"},
			out:  version.Info{Major: "1", Minor: "20"},
			fail: false,
		},
		"invalid major": {
			in:   version.Info{Major: "one", Minor: "20"},
			fail: true,
		},
		"invalid minor": {
			in:   version.Info{Major: "1", Minor: "latest"},
			fail: true,
		},
	}

	for k, tc := range tcs {
		t.Run(k, func(t *testing.T) {
			v, err := processKubernetesVersion(tc.in)
			if tc.fail {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.out, v)
		})
	}
}

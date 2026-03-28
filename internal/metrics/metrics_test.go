package metrics

import "testing"

func TestRouteTemplate(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"/healthz", "/healthz"},
		{"/api/v1/skills", "/api/v1/skills"},
		{"/api/v1/skills/foo", "/api/v1/skills/{name}"},
		{"/api/v1/skills/foo/versions/1.0.0", "/api/v1/skills/{name}/versions/{version}"},
		{"/downloads/abc.tar.gz", "/downloads/{file}"},
		{"/api/v1/other", "/api/unknown"},
	}
	for _, tc := range cases {
		if got := RouteTemplate(tc.path); got != tc.want {
			t.Errorf("RouteTemplate(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

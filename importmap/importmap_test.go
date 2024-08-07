package importmap

import (
	"net/url"
	"testing"
)

func TestResolveImportMap(t *testing.T) {
	baseUrlRaw := "https://site.com"
	baseUrl, _ := url.Parse(baseUrlRaw)
	m, _ := New(WithMapUrl(baseUrl), WithMap(Data{
		Imports: Imports{
			"test":                       "/test-map.js",
			"https://another.com/url.js": "/url-map.js",
		},
		Scopes: Scopes{
			"https://another.com/": {
				"/url.js": "/scoped-map.js",
			},
		},
	}))

	assertUrlsEqualsU(m, "test", baseUrl, "https://site.com/test-map.js", t)
	assertUrlsEquals(m, "/url.js", "https://another.com/", "https://site.com/url-map.js", t)
	assertUrlsEquals(m, "https://site.com/url.js", "https://another.com/x", "https://site.com/scoped-map.js", t)
	assertUrlsEqualsU(m, "https://another.com/url.js", baseUrl, "https://site.com/url-map.js", t)
}

func assertUrlsEquals(m IImportMap, inputUrl string, baseUrl string, expectedUrl string, t *testing.T) {
	t.Helper()

	b, ue := url.Parse(baseUrl)

	if ue != nil {
		panic(ue)
	}

	assertUrlsEqualsU(m, inputUrl, b, expectedUrl, t)
}

func assertUrlsEqualsU(m IImportMap, inputUrl string, baseUrl *url.URL, expectedUrl string, t *testing.T) {
	t.Helper()

	result, err := m.ResolveWithParent(inputUrl, baseUrl)

	if err != nil {
		panic(err)
	}

	if result != expectedUrl {
		t.Errorf("expected %s, got %s", expectedUrl, result)
	}
}

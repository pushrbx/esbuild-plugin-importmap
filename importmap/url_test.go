package importmap

import (
	"net/url"
	"testing"
)

func TestRebaseUrl(t *testing.T) {
	const testUrl = "file:///test/"
	mapUrl, _ := url.Parse("file:///test/a/")
	rootUrl, _ := url.Parse("file:///test/a/")

	sut, err := rebase(testUrl, mapUrl, rootUrl)

	if err != nil {
		t.Fatal(err)
	}

	if sut != "file:///test/" {
		t.Errorf("expected %s, got %s", "file:///test/", sut)
	}
}

func TestRebaseUrlNested(t *testing.T) {
	const testUrl = "file:///test/"
	mapUrl, _ := url.Parse("file:///test/a/")
	rootUrl, _ := url.Parse("file:///test/a/b/")

	sut, err := rebase(testUrl, mapUrl, rootUrl)

	if err != nil {
		t.Fatal(err)
	}

	if sut != "file:///test/" {
		t.Errorf("expected %s, got %s", "file:///test/", sut)
	}
}

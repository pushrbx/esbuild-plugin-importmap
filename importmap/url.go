package importmap

import (
	"errors"
	"net/url"
	"strings"
)

func resolve(inputUrl string, mapUrl *url.URL, rootUrl *url.URL) (string, error) {
	if strings.HasPrefix(inputUrl, "/") {
		if rootUrl != nil {
			var tempUrl string
			if inputUrl[1] == '/' {
				tempUrl = inputUrl[1:]
			} else {
				tempUrl = inputUrl
			}

			return url.JoinPath(rootUrl.String(), ".", tempUrl)
		} else {
			return inputUrl, nil
		}
	}

	u, err := url.Parse(inputUrl)
	if err != nil {
		return "", err
	}

	if mapUrl != nil {
		return mapUrl.ResolveReference(u).String(), nil
	} else {
		return u.String(), nil
	}
}

func rebase(inputUrl string, baseUrl *url.URL, rootUrl *url.URL) (string, error) {
	if baseUrl == nil {
		return "", errors.New("baseUrl is nil; it must be set")
	}

	u, err := url.Parse(inputUrl)

	if err != nil {
		return "", err
	}

	var resolved *url.URL

	if strings.HasPrefix(inputUrl, "/") || strings.HasPrefix(inputUrl, "//") {
		if rootUrl == nil {
			return inputUrl, nil
		}

		resolved = rootUrl.ResolveReference(u)
	} else {
		resolved = baseUrl.ResolveReference(u)
	}

	if rootUrl != nil && strings.HasPrefix(resolved.String(), rootUrl.String()) {
		return resolved.String()[len(rootUrl.String())-1:], nil
	}

	if rootUrl != nil && strings.HasPrefix(rootUrl.String(), resolved.String()) {
		return "/" + rootUrl.ResolveReference(resolved).String(), nil
	}

	if sameOrigin(resolved, baseUrl) {
		return baseUrl.ResolveReference(resolved).String(), nil
	}

	return resolved.String(), nil
}

func sameOrigin(inputUrl *url.URL, baseUrl *url.URL) bool {
	if inputUrl == nil || baseUrl == nil {
		return false
	}
	return inputUrl.Scheme == baseUrl.Scheme && inputUrl.Host == baseUrl.Host && inputUrl.Port() == baseUrl.Port()
}

func isUrl(inputUrl string) bool {
	_, err := url.ParseRequestURI(inputUrl)
	return err == nil
}

func isRelative(specifier string) bool {
	return strings.HasPrefix(specifier, "./") || strings.HasPrefix(specifier, "../") || strings.HasPrefix(specifier, "/")
}

func isPlain(specifier string) bool {
	return !isRelative(specifier) && !isUrl(specifier)
}

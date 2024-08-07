package importmap

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
)

type Scope map[string]string

type Scopes map[string]Scope

type Imports map[string]string

type Integrity map[string]string

type IImportMap interface {
	// Resolve performs a module resolution against the import map.
	//
	// Parameters:
	//   - specified: Specifier to resolve
	// Returns the resolved URL string.
	Resolve(specifier string) (string, error)

	// ResolveWithParent performs a module resolution against the import map.
	//
	// Parameters:
	//   - specified: Specifier to resolve
	//   - parentUrl: Parent URL to resolve against
	// Returns the resolved URL string.
	ResolveWithParent(specifier string, parentUrl *url.URL) (string, error)

	// Rebase will rebase the entire import map to a new mapUrl and rootUrl
	//
	// Parameters:
	//   - mapUrl: The new map URL to use
	//   - rootUrl: The new root URL to use
	// Returns IImportMap for chaining
	Rebase(mapUrl *url.URL, rootUrl *url.URL) error

	// Flatten groups the import map scopes to shared URLs to reduce duplicate mappings.
	//
	// For two given scopes, "https://site.com/x/" and "https://site.com/y/",
	// a single scope will be constructed for "https://site.com/" including
	// their shared mappings, only retaining the scopes if they have differences.
	//
	// In the case where the scope is on the same origin as the mapUrl, the grouped
	// scope is determined based on determining the common baseline over all local scopes.
	// Returns IImportMap for chaining
	Flatten() IImportMap

	// CombineSubPaths groups subpath mappings into path mappings when multiple exact subpaths
	// exist under the same path.
	//
	// For two mappings like { "base/a.js": "/a.js", "base/b.js": "/b.js" },
	// these will be replaced with a single path mapping { "base/": "/" }.
	// Groupings are done throughout all import scopes individually.
	//
	// Returns IImportMap for chaining
	CombineSubPaths() IImportMap

	// Replace will replace URLs in the import map in bulk
	// Provide a URL ending in "/" to perform path replacements.
	//
	// Parameters:
	//   - url: URL to replace
	//   - newUrl: URL to replace with
	//
	// Returns IImportMap for chaining
	Replace(url url.URL, newUrl url.URL) IImportMap

	GetIntegrity() Integrity

	GetIntegrityValue(target string, integrity string) (string, error)

	SetIntegrityValue(target string, integrity string) error

	// Set will set a specific entry in the import map.
	Set(name string, target string) IImportMap

	// SetWithParent will set a specific entry in the import map.
	SetWithParent(name string, target string, parent string) IImportMap

	Extend(importMap IImportMap, overrideScopes bool) (IImportMap, error)

	Clone() IImportMap

	GetScopes() Scopes

	GetImports() Imports
}

type Options struct {
	Map     Data
	MapUrl  *url.URL
	RootUrl *url.URL
}

type Option func(options *Options)

type Data struct {
	Imports   Imports   `json:"imports,omitempty"`
	Scopes    Scopes    `json:"scopes,omitempty"`
	Integrity Integrity `json:"integrity,omitempty"`
}

type importMap struct {
	imports   Imports
	scopes    Scopes
	integrity Integrity
	mapUrl    *url.URL
	rootUrl   *url.URL
}

// New creates a new IImportMap instance
func New(opts ...Option) (IImportMap, error) {
	options := &Options{}

	for _, opt := range opts {
		opt(options)
	}

	obj := &importMap{
		imports:   options.Map.Imports,
		scopes:    options.Map.Scopes,
		integrity: options.Map.Integrity,
		mapUrl:    options.MapUrl,
		rootUrl:   options.RootUrl,
	}

	if obj.mapUrl == nil {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		obj.mapUrl, err = url.Parse(fmt.Sprintf("file://%s/", cwd))
		if err != nil {
			return nil, err
		}
	}

	if obj.rootUrl == nil && (obj.mapUrl.Scheme == "http" || obj.mapUrl.Scheme == "https") {
		obj.rootUrl = obj.mapUrl.ResolveReference(&url.URL{Path: "/"})
	}

	return obj, nil
}

func WithMap(importMap Data) Option {
	return func(options *Options) {
		options.Map = importMap
	}
}

func WithMapUrl(mapUrl *url.URL) Option {
	return func(options *Options) {
		options.MapUrl = mapUrl
	}
}

func WithRootUrl(rootUrl *url.URL) Option {
	return func(options *Options) {
		options.RootUrl = rootUrl
	}
}

func (i *importMap) Clone() IImportMap {
	return &importMap{
		imports:   i.imports,
		scopes:    i.scopes,
		integrity: i.integrity,
		mapUrl:    i.mapUrl,
		rootUrl:   i.rootUrl,
	}
}

func (i *importMap) Extend(importMap IImportMap, overrideScopes bool) (IImportMap, error) {
	for k, v := range importMap.GetImports() {
		i.imports[k] = v
	}

	if overrideScopes {
		for k, v := range importMap.GetScopes() {
			i.scopes[k] = v
		}
	} else if importMap.GetScopes() != nil {
		for scopeKey, scope := range importMap.GetScopes() {
			if _, ok := i.scopes[scopeKey]; !ok {
				i.scopes[scopeKey] = make(Scope)
			}

			for k, v := range scope {
				i.scopes[scopeKey][k] = v
			}
		}
	}

	for k, v := range importMap.GetIntegrity() {
		i.integrity[k] = v
	}
	err := i.Rebase(i.mapUrl, nil)
	if err != nil {
		return nil, err
	}
	return i, nil
}

func (i *importMap) Set(name string, target string) IImportMap {
	i.imports[name] = target
	return i
}

func (i *importMap) SetWithParent(name string, target string, parent string) IImportMap {
	if i.scopes[parent] == nil {
		i.scopes[parent] = make(Scope)
	}
	i.scopes[parent][name] = target
	return i
}

func (i *importMap) GetScopes() Scopes {
	return i.scopes
}

func (i *importMap) GetImports() Imports {
	return i.imports
}

func (i *importMap) GetIntegrityValue(target string, _ string) (string, error) {
	targetRebased, err := rebase(target, i.mapUrl, i.rootUrl)
	if err != nil {
		return "", err
	}

	if v, ok := i.integrity[targetRebased]; ok {
		return v, nil
	}

	if v, ok := i.integrity[targetRebased[2:]]; ok {
		return v, nil
	}
	return "", errors.New("integrity not found")
}

func (i *importMap) SetIntegrityValue(target string, integrity string) error {
	i.integrity[target] = integrity
	targetRebased, err := rebase(target, i.mapUrl, i.rootUrl)
	if err != nil {
		return err
	}

	if targetRebased != target {
		if _, ok := i.integrity[targetRebased]; ok {
			delete(i.integrity, targetRebased)
		}
	}
	if strings.HasPrefix(targetRebased, "./") && target != targetRebased[2:] {
		if _, ok := i.integrity[targetRebased[2:]]; ok {
			delete(i.integrity, targetRebased)
		}
	}
	return nil
}

// Rebase is an implementation of the IImportMap interface.
func (i *importMap) Rebase(mapUrl *url.URL, rootUrl *url.URL) error {
	if mapUrl == nil {
		return errors.New("invalid argument: mapUrl is nil")
	}
	if rootUrl == nil && i.mapUrl != nil {
		if mapUrl.String() == i.mapUrl.String() {
			rootUrl = i.mapUrl
		} else {
			if i.rootUrl == nil || (mapUrl.Scheme != "https" && mapUrl.Scheme != "http") {
				rootUrl = nil
			} else {
				rootUrl = mapUrl.ResolveReference(&url.URL{Path: "/"})
			}
		}
	}

	for importKey, target := range i.imports {
		resolvedTarget, err := resolve(target, i.mapUrl, i.rootUrl)
		if err != nil {
			return err
		}
		i.imports[importKey], err = rebase(resolvedTarget, i.mapUrl, i.rootUrl)
		if err != nil {
			return err
		}

		if !isPlain(importKey) {
			resolvedImportKey, resolvErr := resolve(importKey, i.mapUrl, i.rootUrl)
			if resolvErr != nil {
				return resolvErr
			}
			newImport, rebaseErr := rebase(resolvedImportKey, mapUrl, rootUrl)
			if rebaseErr != nil {
				return rebaseErr
			}

			if newImport != importKey {
				i.imports[newImport] = i.imports[importKey]
				delete(i.imports, importKey)
			}
		}
	}

	for scopeKey, scopeImports := range i.scopes {
		changedScopeImportProps := false
		for importKey, target := range scopeImports {
			resolvedTarget, err := resolve(target, i.mapUrl, i.rootUrl)
			if err != nil {
				return err
			}
			scopeImports[importKey], err = rebase(resolvedTarget, i.mapUrl, i.rootUrl)
			if err != nil {
				return err
			}
			if !isPlain(importKey) {
				resolvedImportKey, resolvErr := resolve(importKey, i.mapUrl, i.rootUrl)
				if resolvErr != nil {
					return resolvErr
				}
				newImport, rebaseErr := rebase(resolvedImportKey, mapUrl, rootUrl)
				if rebaseErr != nil {
					return rebaseErr
				}

				if newImport != importKey {
					changedScopeImportProps = true
					scopeImports[newImport] = scopeImports[importKey]
					delete(scopeImports, importKey)
				}
			}
		}

		if changedScopeImportProps {
			i.scopes[scopeKey] = scopeImports
		}

		resolvedScopeKey, err := resolve(scopeKey, i.mapUrl, i.rootUrl)
		if err != nil {
			return err
		}
		newScope, rebaseErr := rebase(resolvedScopeKey, mapUrl, rootUrl)
		if rebaseErr != nil {
			return rebaseErr
		}

		if newScope != scopeKey {
			i.scopes[newScope] = i.scopes[scopeKey]
			delete(i.scopes, scopeKey)
		}
	}

	for integrityKey, integrityValue := range i.integrity {
		resolvedIntegrityValue, err := resolve(integrityValue, i.mapUrl, i.rootUrl)
		if err != nil {
			return err
		}
		i.integrity[integrityKey], err = rebase(resolvedIntegrityValue, i.mapUrl, i.rootUrl)
		if err != nil {
			return err
		}
		if integrityKey != integrityValue {
			i.integrity[integrityKey] = i.integrity[integrityValue]
			delete(i.integrity, integrityValue)
		}
	}

	i.mapUrl = mapUrl
	i.rootUrl = rootUrl
	return nil
}

// Flatten is an implementation of the IImportMap interface.
func (i *importMap) Flatten() IImportMap {
	// todo: implement
	return nil
}

func (i *importMap) CombineSubPaths() IImportMap {
	// todo: implement
	return nil
}

func (i *importMap) Replace(_ url.URL, _ url.URL) IImportMap {
	// todo: implement
	return nil
}

func (i *importMap) Resolve(specifier string) (string, error) {
	return i.ResolveWithParent(specifier, i.mapUrl)
}

func (i *importMap) ResolveWithParent(specifier string, parentUrl *url.URL) (string, error) {
	parentUrlRaw, err := resolve(parentUrl.String(), i.mapUrl, i.rootUrl)

	if err != nil {
		return "", err
	}

	var specifierUrl *url.URL
	if !isPlain(specifier) {
		u, urlParseErr := url.Parse(specifier)
		if urlParseErr != nil {
			return "", urlParseErr
		}
		specifierUrl = parentUrl.ResolveReference(u)
		specifier = specifierUrl.String()
	}

	scopeMatches, err := getScopeMatches(parentUrlRaw, i.scopes, i.mapUrl, i.rootUrl)
	if err != nil {
		return "", err
	}

	for _, scopeMatch := range scopeMatches {
		mapMatch := getMapMatch(specifier, i.scopes[scopeMatch.First])
		if mapMatch == "" && specifierUrl != nil {
			specifier, err = rebase(specifier, i.mapUrl, i.rootUrl)
			if err != nil {
				return "", err
			}
			mapMatch = getMapMatch(specifier, i.scopes[scopeMatch.First])
			if mapMatch == "" && i.rootUrl != nil {
				specifier, err = rebase(specifier, i.mapUrl, nil)
				if err != nil {
					return "", err
				}
				mapMatch = getMapMatch(specifier, i.scopes[scopeMatch.First])
			}
		}
		if mapMatch != "" {
			target := i.scopes[scopeMatch.First][mapMatch]
			return resolve(target+specifier[len(mapMatch):], i.mapUrl, i.rootUrl)
		}
	}
	mapMatch := getMapMatch(specifier, i.imports)
	if mapMatch == "" && specifierUrl != nil {
		specifier, err = rebase(specifier, i.mapUrl, i.rootUrl)
		if err != nil {
			return "", err
		}
		mapMatch = getMapMatch(specifier, i.imports)
		if mapMatch == "" && i.rootUrl != nil {
			specifier, err = rebase(specifier, i.mapUrl, nil)
			if err != nil {
				return "", err
			}
			mapMatch = getMapMatch(specifier, i.imports)
		}
	}

	if mapMatch != "" {
		target := i.imports[mapMatch]
		return resolve(target+specifier[len(mapMatch):], i.mapUrl, i.rootUrl)
	}

	if specifierUrl != nil {
		return specifierUrl.String(), nil
	}
	return "", fmt.Errorf("unable to resolve %s in %s", specifier, parentUrl.String())
}

func (i *importMap) GetIntegrity() Integrity {
	return i.integrity
}

type scopeMatchTuple struct {
	First  string
	Second string
}

func getScopeMatches(parentUrl string, scopes Scopes, mapUrl *url.URL, rootUrl *url.URL) ([]scopeMatchTuple, error) {
	scopeCandidates := make([]scopeMatchTuple, 0, len(scopes))
	for scope := range scopes {
		scopeUrl, err := resolve(scope, mapUrl, rootUrl)
		if err != nil {
			return nil, err
		}
		scopeCandidates = append(scopeCandidates, scopeMatchTuple{
			First:  scope,
			Second: scopeUrl,
		})
	}

	sort.Slice(scopeCandidates, func(i, j int) bool {
		return len(scopeCandidates[i].Second) < len(scopeCandidates[j].Second)
	})

	var result []scopeMatchTuple
	for _, candidate := range scopeCandidates {
		scopeUrl := candidate.Second
		if scopeUrl == parentUrl || (strings.HasSuffix(scopeUrl, "/") && strings.HasPrefix(parentUrl, scopeUrl)) {
			result = append(result, candidate)
		}
	}

	return result, nil
}

func getMapMatch[T any](specifier string, inputMap map[string]T) string {
	if _, ok := inputMap[specifier]; ok {
		return specifier
	}
	var curMatch string
	for match := range inputMap {
		wildcard := strings.HasSuffix(match, "*")
		if !strings.HasSuffix(match, "/") && !wildcard {
			continue
		}
		if strings.HasPrefix(specifier, match[:len(match)-1]) {
			if curMatch == "" || len(match) > len(curMatch) {
				curMatch = match
			}
		}
	}
	return curMatch
}

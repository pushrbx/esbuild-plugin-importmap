package importmap

import (
	"encoding/json"
	"os"
)

// LoadFromFile  loads the contents of the import map file and returns an IImportMap instance
func LoadFromFile(path string) (IImportMap, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}

	fileContents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	data := Data{}
	err = json.Unmarshal(fileContents, &data)
	if err != nil {
		return nil, err
	}

	m := New(WithMap(data))

	return m, nil
}

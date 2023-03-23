package utils

import "embed"

func ExtractEmbeddedFsIntoFileMap(fs embed.FS, dir string) (map[string]string, error) {
	files := make(map[string]string)
	dirs, err := fs.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range dirs {
		if !entry.IsDir() {
			content, err := fs.ReadFile(dir + "/" + entry.Name())
			if err != nil {
				return nil, err
			}
			files[entry.Name()] = string(content)
		}
	}
	return files, err
}

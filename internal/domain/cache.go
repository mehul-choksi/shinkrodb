package domain

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

func filterASCII(s string) string {
	var result []rune
	for _, r := range s {
		if isAllowedCharacter(r) {
			result = append(result, r)
		}
	}
	return string(result)
}

func isAllowedCharacter(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return true
	}
	switch r {
	case '\n', '-', '_', '.', '~', ':', '/', '?', '#', '[', ']', '@', '!', '$', '&', '\'', '(', ')', '*', '+', ',', ';', '=', '%', '"':
		return true
	default:
		return false
	}
}

func validShowType(content string) bool {
	choices := []string{"tv"}

	valid := false
	for _, v := range choices {
		valid = valid || strings.Contains(content, "type="+v)
	}
	return valid
}

func matchPattern(searchTerm string, filteredContent string, startPoint string, endPoint string) string {

	index := strings.Index(filteredContent, searchTerm)

	for index != -1 {
		start := strings.LastIndex(filteredContent[:index], startPoint)
		if start == -1 {
			break
		}

		end := strings.Index(filteredContent[index:], endPoint)
		if end == -1 {
			break
		}
		end += index

		result := filteredContent[start+1 : end]
		return result

	}
	return "Not Found"
}

func shouldDelete(fileName string) bool {
	data, err := os.ReadFile(fileName)
	if err != nil {
		log.Fatalf("failed reading file: %s", err)
	}
	content := string(data)
	filteredContent := filterASCII(content)

	if !validShowType(filteredContent) {
		fmt.Println("Show type invalid, skipping further analysis")
		return false
	} else {
		fmt.Println("Show type valid, proceeding with anidb url search")
	}

	searchTerm := "canonical"
	animeUrl := matchPattern(searchTerm, filteredContent, "\"", "\n")

	animeUrl = strings.ReplaceAll(animeUrl, `canonical"href="`, "")
	animeUrl = strings.ReplaceAll(animeUrl, `"/`, "")

	animeId := strings.ReplaceAll(animeUrl, `https://myanimelist.net/anime/`, "")
	animeId = strings.SplitAfter(animeId, `/`)[0]
	animeId = strings.ReplaceAll(animeId, `/`, "")

	fmt.Println("Anime id ", animeId)
	animeIdNumber, _ := strconv.Atoi(animeId)

	if animeIdNumber < 50000 {
		return false
	}

	searchTerm = "anidb.net"
	aniDbUrl := matchPattern(searchTerm, filteredContent, "\"", "\"")

	result := !strings.HasPrefix(aniDbUrl, "https://")

	writeable := "Delete " + animeUrl + " ? "
	if result {
		writeable += " Yes\n"
		err := os.Remove(fileName)
		if err != nil {
			log.Printf("Error while deleting file %v %v", fileName, err)
		}
	} else {
		writeable += " No\n"
	}

	fmt.Println(writeable)

	return result
}

func CleanCache() {
	dirPath := "mal_cache"
	dir, err := os.Open(dirPath)
	if err != nil {
		log.Fatalf("Failed to open directory: %v", err)
	}
	defer dir.Close()

	folders, err := dir.Readdir(-1)
	if err != nil {
		log.Fatalf("Failed to read directory: %v", err)
	}

	var wg sync.WaitGroup

	for _, folder := range folders {
		if folder.IsDir() {
			folderPath := filepath.Join(dirPath, folder.Name())
			fmt.Printf("Processing folder: %s\n", folderPath)

			folderDir, err := os.Open(folderPath)
			if err != nil {
				log.Printf("Failed to open folder %s: %v", folder.Name(), err)
				continue
			}
			defer folderDir.Close()

			files, err := folderDir.Readdir(-1)
			if err != nil {
				log.Printf("Failed to list files in folder %s: %v", folder.Name(), err)
				continue
			}

			for _, file := range files {
				if !file.IsDir() {
					wg.Add(1)
					go func(filePath string) {
						defer wg.Done()
						fmt.Printf("Reading file: %s\n", filePath)
						fmt.Println("Should Delete file ", shouldDelete(filePath))
					}(filepath.Join(folderPath, file.Name()))
				}
			}
		}
	}

	wg.Wait()
}

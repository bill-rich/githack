package main

import (
	"compress/zlib"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	objects, err := Objects(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	out, err := json.Marshal(objects)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
}

type ObjectType string

const (
	ObjectTypeBlob   = "blob"
	ObjectTypeTree   = "tree"
	ObjectTypeCommit = "commit"
	ObjectTypeTag    = "tag"
)

type ObjectContent interface {
	String() string
}

type Object struct {
	Type    ObjectType
	Size    int64
	Content ObjectContent
	Sha1    string
}

type ObjectContentString string
type ObjectContentTree struct {
	Files []ObjectFile
}

func (s ObjectContentString) String() string {
	return string(s)
}

func (t ObjectContentTree) String() string {
	return fmt.Sprintf("%+d", len(t.Files))
}

type ObjectFile struct {
	Name string
	Mode string
	Hash string
}

func Objects(repoDir string) ([]Object, error) {
	var objects []Object
	folders, err := ioutil.ReadDir(fmt.Sprintf("%s/.git/objects", repoDir))
	if err != nil {
		return objects, err
	}
	re := regexp.MustCompile("^[0-9a-f]{2}$")
	for _, dirInfo := range folders {
		if dirInfo.IsDir() && re.MatchString(dirInfo.Name()) {
			files, err := ioutil.ReadDir(fmt.Sprintf("%s/.git/objects/%s", repoDir, dirInfo.Name()))
			if err != nil {
				return objects, err
			}
			for _, fileInfo := range files {
				object, err := ReadObject(fmt.Sprintf("%s/.git/objects/%s/%s", repoDir, dirInfo.Name(), fileInfo.Name()))
				if err != nil {
					return objects, err
				}
				objects = append(objects, object)
			}
		}
	}
	return objects, nil
}

func ReadObject(fileName string) (Object, error) {
	var object Object
	file, err := os.Open(fileName)
	if err != nil {
		return object, err
	}
	reader, err := zlib.NewReader(file)
	if err != nil {
		return object, err
	}
	raw, err := ioutil.ReadAll(reader)
	if err != nil {
		return object, nil
	}
	var chunks [][]byte
	var lastNull int
	for i, byte := range raw {
		if byte == 0 {
			chunks = append(chunks, raw[lastNull:i])
			lastNull = i + 1
		}
	}
	if len(raw) > lastNull {
		chunks = append(chunks, raw[lastNull:])
	}

	a := sha1.New()
	a.Write(raw)

	object.Sha1 = fmt.Sprintf("%x", a.Sum(nil))
	Header(chunks[0], &object)

	switch {
	case len(chunks) == 2:
		object.Content = ObjectContentString(chunks[1])
	default:
		object.Content = TreeContent(chunks[1:]...)
	}
	return object, nil
}
func TreeContent(chunks ...[]byte) ObjectContentTree {
	tree := ObjectContentTree{
		Files: []ObjectFile{},
	}
	headers := strings.Split(string(chunks[0]), " ")
	objFile := &ObjectFile{
		Name: headers[0],
		Mode: headers[1],
	}
	for i, chunk := range chunks {
		if i == 0 {
			continue
		}
		objFile.Hash = fmt.Sprintf("%x", chunk[:19])
		tree.Files = append(tree.Files, *objFile)
		if len(chunk) > 19 {
			headers := strings.Split(string(chunks[0]), " ")
			objFile = &ObjectFile{
				Name: headers[0],
				Mode: headers[1],
			}
		}
	}
	return tree
}

func Header(header []byte, object *Object) error {
	headerFields := strings.Split(string(header), " ")
	if len(headerFields) != 2 {
		return fmt.Errorf("expected 2 or 3 header fields, got %d", len(headerFields))
	}
	size, err := strconv.Atoi(headerFields[1])
	if err != nil {
		return nil
	}
	object.Size = int64(size)
	object.Type = ObjectType(string(headerFields[0]))
	return nil
}

func TreeHeader(fields []string) {

}

// Binary directory_indexer produces a set of Entry protos representing the
// files in the given directories.
//
// For instance, a file 'kythe/javatests/com/google/devtools/kythe/util/CAMPFIRE' would produce two entries:
//   {
//     "fact_name": "/kythe/node/kind",
//     "fact_value": "file",
//     "source": {
//       "signature": "c2b0d93b83c1b0e22fd564278be1b0373b1dcb67ff3bb77c2f29df7c393fe580",
//       "corpus": "kythe",
//       "root": "",
//       "path": "third_party/kythe/javatests/com/google/devtools/kythe/util/CAMPFIRE",
//       "language": ""
//     }
//   }
//   {
//     "fact_name": "/kythe/text",
//     "fact_value": "...",
//     "source": {
//       "signature": "c2b0d93b83c1b0e22fd564278be1b0373b1dcb67ff3bb77c2f29df7c393fe580",
//       "corpus": "kythe",
//       "root": "",
//       "path": "third_party/kythe/javatests/com/google/devtools/kythe/util/CAMPFIRE",
//       "language": ""
//     }
//   }
//
// Usage:
//   directory_indexer --corpus kythe --root kythe ~/repo/kythe/ \
//     --exclude '^buildtools,^campfire-out,^third_party,~$,#$,(^|/)\.'
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"third_party/kythe/go/platform/delimited"
	"third_party/kythe/go/storage/filevnames"

	spb "third_party/kythe/proto/storage_proto"

	"code.google.com/p/goprotobuf/proto"
)

var (
	vnamesConfigPath = flag.String("vnames", "", "Path to JSON VNames configuration")
	exclude          = flag.String("exclude", "", "Comma-separated list of exclude regexp patterns")
)

var (
	kindLabel = "/kythe/node/kind"
	textLabel = "/kythe/text"

	fileKind = []byte("file")
)

var w = delimited.NewWriter(os.Stdout)

func emitEntry(v *spb.VName, label string, value []byte) error {
	return w.PutProto(&spb.Entry{Source: v, FactName: &label, FactValue: value})
}

var (
	fileVNames *filevnames.Config
	excludes   []*regexp.Regexp
)

func emitPath(path string, info os.FileInfo, err error) error {
	if info.IsDir() {
		return nil
	}
	for _, re := range excludes {
		if re.MatchString(path) {
			return nil
		}
	}

	log.Printf("Reading/emitting %s", path)
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	vName := fileVNames.LookupVName(path)

	digest := sha256.Sum256(contents)
	vName.Signature = proto.String(hex.EncodeToString(digest[:]))

	if vName.Language == nil {
		vName.Language = sourceLanguage(path)
	}
	if vName.Path == nil {
		vName.Path = proto.String(path)
	}

	if err := emitEntry(vName, kindLabel, fileKind); err != nil {
		return err
	}
	return emitEntry(vName, textLabel, contents)
}

func main() {
	flag.Parse()

	if *exclude != "" {
		for _, pattern := range strings.Split(*exclude, ",") {
			excludes = append(excludes, regexp.MustCompile(pattern))
		}
	}

	config, err := filevnames.ParseFile(*vnamesConfigPath)
	if err != nil {
		log.Fatalf("Failed to read VNames config file %q: %v", *vnamesConfigPath, err)
	}
	fileVNames = config

	dirs := flag.Args()
	if len(dirs) == 0 {
		dirs = []string{"."}
	}

	for _, dir := range dirs {
		if err := filepath.Walk(dir, emitPath); err != nil {
			log.Fatalf("Error walking %s: %v", dir, err)
		}
	}
}

func sourceLanguage(path string) *string {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	switch ext {
	case "java", "go":
		return proto.String(ext)
	case "py":
		return proto.String("python")
	case "cc", "cpp", "h":
		return proto.String("c++")
	default:
		return nil
	}
}

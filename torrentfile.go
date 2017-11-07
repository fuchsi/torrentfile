/*
 * Copyright (c) 2017 Daniel Müller
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package torrentfile

import (
	"crypto/sha1"
	"io"
	"strings"
	"time"

	"github.com/fuchsi/bencode"
)

const PIECE_SIZE = 20

type TorrentFile struct {
	Name         string
	AnnounceUrl  string
	AnnounceList []string
	PieceLength  uint64
	Pieces       [][PIECE_SIZE]byte
	Files        []File
	Private      bool
	Comment      string
	CreatedBy    string
	CreationDate time.Time
	Encoding     string

	info map[string]interface{}
}

type File struct {
	Length uint64
	Path   string
}

func (t TorrentFile) TotalSize() uint64 {
	totalSize := uint64(0)

	for _, file := range t.Files {
		totalSize += file.Length
	}

	return totalSize
}

func (t TorrentFile) InfoHash() [PIECE_SIZE]byte {
	infoStr := bencode.Encode(t.info)
	return sha1.Sum(infoStr)
}

func (t TorrentFile) Encode() []byte {
	dict := make(map[string]interface{})
	info := make(map[string]interface{})

	// global dict
	dict["announce"] = t.AnnounceUrl
	if len(t.AnnounceList) > 0 {
		dict["announce-list"] = t.AnnounceList
	}
	if t.CreationDate.Unix() > 0 {
		dict["creation date"] = t.CreationDate.Unix()
	}
	if t.CreatedBy != "" {
		dict["created by"] = t.CreatedBy
	}
	if t.Comment != "" {
		dict["comment"] = t.Comment
	}
	if t.Encoding != "" {
		dict["encoding"] = t.Encoding
	}

	// info dict
	info["piece length"] = t.PieceLength
	var pieces string
	for _, v := range t.Pieces {
		pieces += string(v[:])
	}
	info["pieces"] = pieces
	if t.Private {
		info["private"] = 1
	}
	if t.Name != "" {
		info["name"] = t.Name
	}

	// files list
	singleFile := false
	if len(t.Files) == 1 { // single file mode
		if strings.Count(t.Files[0].Path, "/") == 0 { // really single file mode
			singleFile = true
			info["name"] = t.Files[0].Path
			info["length"] = t.Files[0].Length
		}
	}
	if !singleFile {
		files := make([]interface{}, len(t.Files))
		for i, v := range t.Files {
			file := make(map[string]interface{}, 2)
			file["length"] = v.Length
			file["path"] = partitionPath(v.Path)
			files[i] = file
		}
		info["files"] = files
	}

	dict["info"] = info

	return bencode.Encode(dict)
}

func DecodeTorrentFile(reader io.Reader) (TorrentFile, error) {
	dict, err := bencode.Decode(reader)
	if err != nil {
		return TorrentFile{}, err
	}

	info := dict["info"].(map[string]interface{})

	torrentfile := TorrentFile{
		AnnounceUrl: dict["announce"].(string),
		PieceLength: uint64(info["piece length"].(int64)),
		Pieces:      decodePieces(info["pieces"].(string)),
		info:        info,
	}

	if info["name"] != nil {
		torrentfile.Name = info["name"].(string)
	}
	if info["private"] != nil {
		torrentfile.Private = info["private"].(int64) == 1
	}
	if dict["comment"] != nil {
		torrentfile.Comment = dict["comment"].(string)
	}
	if dict["created by"] != nil {
		torrentfile.CreatedBy = dict["created by"].(string)
	}
	if dict["creation date"] != nil {
		torrentfile.CreationDate = time.Unix(dict["creation date"].(int64), 0)
	} else {
		torrentfile.CreationDate = time.Unix(0, 0)
	}
	if dict["encoding"] != nil {
		torrentfile.Encoding = dict["encoding"].(string)
	}

	if info["files"] != nil { // multiple file mode
		files := info["files"].([]interface{})
		torrentfile.Files = decodeFiles(&files)
	} else {
		filename := ""
		if info["name"] != nil {
			filename = info["name"].(string)
		}
		torrentfile.Files = []File{}
		torrentfile.Files = append(torrentfile.Files, File{Length: uint64(info["length"].(int64)), Path: filename})
	}

	if dict["announce-list"] != nil {
		l := info["announce-list"].([]interface{})
		al := make([]string, len(l))

		for _, v := range l {
			al = append(al, v.(string))
		}

		torrentfile.AnnounceList = al
	}

	return torrentfile, nil
}

func decodeFiles(fileList *[]interface{}) []File {
	files := make([]File, 0, len(*fileList))

	for _, v := range *fileList {
		file := v.(map[string]interface{})
		files = append(files, File{Length: uint64(file["length"].(int64)), Path: flattenPath(file["path"].([]interface{}))})
	}

	return files
}

func flattenPath(pathList []interface{}) string {
	var path string

	for _, p := range pathList {
		path += p.(string) + "/"
	}

	return strings.TrimRight(path, "/")
}

func partitionPath(path string) []interface{} {
	p := make([]interface{}, strings.Count(path, "/")+1)
	for i, v := range strings.Split(path, "/") {
		p[i] = v
	}

	return p
}

func decodePieces(pieceString string) [][PIECE_SIZE]byte {
	a := []byte(pieceString)
	pieces := make([][PIECE_SIZE]byte, 0, len(a)/PIECE_SIZE/2)
	var buf [PIECE_SIZE]byte

	for i, b := range a {
		buf[(i % PIECE_SIZE)] = b
		if (i+1)%PIECE_SIZE == 0 {
			pieces = append(pieces, buf)
			buf = [PIECE_SIZE]byte{}
		}
	}

	return pieces
}

func EncodeTorrentFile(t TorrentFile) []byte {
	return t.Encode()
}

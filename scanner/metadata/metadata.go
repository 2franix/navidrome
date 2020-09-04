package metadata

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/deluan/navidrome/log"
)

type Extractor interface {
	Extract(files ...string) (map[string]Metadata, error)
}

func Extract(files ...string) (map[string]Metadata, error) {
	e := &taglibExtractor{}
	return e.Extract(files...)
}

type Metadata interface {
	Title() string
	Album() string
	Artist() string
	AlbumArtist() string
	SortTitle() string
	SortAlbum() string
	SortArtist() string
	SortAlbumArtist() string
	Composer() string
	Genre() string
	Year() int
	TrackNumber() (int, int)
	DiscNumber() (int, int)
	DiscSubtitle() string
	HasPicture() bool
	Comment() string
	Compilation() bool
	Duration() float32
	BitRate() int
	ModificationTime() time.Time
	FilePath() string
	Suffix() string
	Size() int64
}

type baseMetadata struct {
	filePath string
	fileInfo os.FileInfo
	tags     map[string]string
}

func (m *baseMetadata) Title() string       { return m.getTag("title", "sort_name") }
func (m *baseMetadata) Album() string       { return m.getTag("album", "sort_album") }
func (m *baseMetadata) Artist() string      { return m.getTag("artist", "sort_artist") }
func (m *baseMetadata) AlbumArtist() string { return m.getTag("album_artist", "albumartist") }
func (m *baseMetadata) SortTitle() string   { return m.getSortTag("", "title", "name") }
func (m *baseMetadata) SortAlbum() string   { return m.getSortTag("", "album") }
func (m *baseMetadata) SortArtist() string  { return m.getSortTag("", "artist") }
func (m *baseMetadata) SortAlbumArtist() string {
	return m.getSortTag("tso2", "albumartist", "album_artist")
}
func (m *baseMetadata) Composer() string        { return m.getTag("composer", "tcm", "sort_composer") }
func (m *baseMetadata) Genre() string           { return m.getTag("genre") }
func (m *baseMetadata) Year() int               { return m.parseYear("date") }
func (m *baseMetadata) Comment() string         { return m.getTag("comment") }
func (m *baseMetadata) Compilation() bool       { return m.parseBool("compilation") }
func (m *baseMetadata) TrackNumber() (int, int) { return m.parseTuple("track", "tracknumber") }
func (m *baseMetadata) DiscNumber() (int, int)  { return m.parseTuple("disc", "discnumber") }
func (m *baseMetadata) DiscSubtitle() string {
	return m.getTag("tsst", "discsubtitle", "setsubtitle")
}

func (m *baseMetadata) ModificationTime() time.Time { return m.fileInfo.ModTime() }
func (m *baseMetadata) Size() int64                 { return m.fileInfo.Size() }
func (m *baseMetadata) FilePath() string            { return m.filePath }
func (m *baseMetadata) Suffix() string {
	return strings.ToLower(strings.TrimPrefix(path.Ext(m.FilePath()), "."))
}

func (m *baseMetadata) Duration() float32 { panic("not implemented") }
func (m *baseMetadata) BitRate() int      { panic("not implemented") }
func (m *baseMetadata) HasPicture() bool  { panic("not implemented") }

func (m *baseMetadata) parseInt(tagName string) int {
	if v, ok := m.tags[tagName]; ok {
		i, _ := strconv.Atoi(v)
		return i
	}
	return 0
}

var dateRegex = regexp.MustCompile(`^([12]\d\d\d)`)

func (m *baseMetadata) parseYear(tagName string) int {
	if v, ok := m.tags[tagName]; ok {
		match := dateRegex.FindStringSubmatch(v)
		if len(match) == 0 {
			log.Warn("Error parsing year from ffmpeg date field", "file", m.filePath, "date", v)
			return 0
		}
		year, _ := strconv.Atoi(match[1])
		return year
	}
	return 0
}

func (m *baseMetadata) getTag(tags ...string) string {
	for _, t := range tags {
		if v, ok := m.tags[t]; ok {
			return v
		}
	}
	return ""
}

func (m *baseMetadata) getSortTag(originalTag string, tags ...string) string {
	formats := []string{"sort%s", "sort_%s", "sort-%s", "%ssort", "%s_sort", "%s-sort"}
	all := []string{originalTag}
	for _, tag := range tags {
		for _, format := range formats {
			name := fmt.Sprintf(format, tag)
			all = append(all, name)
		}
	}
	return m.getTag(all...)
}

func (m *baseMetadata) parseTuple(tags ...string) (int, int) {
	for _, tagName := range tags {
		if v, ok := m.tags[tagName]; ok {
			tuple := strings.Split(v, "/")
			t1, t2 := 0, 0
			t1, _ = strconv.Atoi(tuple[0])
			if len(tuple) > 1 {
				t2, _ = strconv.Atoi(tuple[1])
			} else {
				t2, _ = strconv.Atoi(m.tags[tagName+"total"])
			}
			return t1, t2
		}
	}
	return 0, 0
}

func (m *baseMetadata) parseBool(tagName string) bool {
	if v, ok := m.tags[tagName]; ok {
		i, _ := strconv.Atoi(strings.TrimSpace(v))
		return i == 1
	}
	return false
}

var zeroTime = time.Date(0000, time.January, 1, 0, 0, 0, 0, time.UTC)

func (m *baseMetadata) parseDuration(tagName string) float32 {
	if v, ok := m.tags[tagName]; ok {
		d, err := time.Parse("15:04:05", v)
		if err != nil {
			return 0
		}
		return float32(d.Sub(zeroTime).Seconds())
	}
	return 0
}

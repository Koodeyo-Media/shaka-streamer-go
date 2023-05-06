// Contains the helper classes for HLS parsing and concatenation.
package streamer

import (
	"bufio"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	/*
		Common header tags to search for so we don't store them
			in `MediaPlaylist.content`. These tags must be defined only one time
		in a playlist, and written at the top of a media playlist file once.
	*/
	HEADER_TAGS = []string{"#EXTM3U", "#EXT-X-VERSION", "#EXT-X-PLAYLIST-TYPE"}
)

/*
A class representing a media playlist(any playlist that references
media files or media files' segments whether they were video/audio/text files).

The information collected from the master playlist about
a specific media playlist such as the bitrate, codec, etc...
is also stored here in `self.stream_info` dictionary.

Keep in mind that a stream variant playlist is also a MediaPlaylist.
*/
type MediaPlaylist struct {
	/*
		A number that is shared between all the MediaPlaylist objects to be used
		 to generate unique file names in the format 'stream_<current_stream_index>.m3u8'
	*/
	CurrentStreamIndex int

	// Common header tags to search for so we don't store them
	// in `MediaPlaylist.content`. These tags must be defined only one time
	// in a playlist, and written at the top of a media playlist file once.
	HeaderTags []string

	// A dictionary containing information about the stream, such as the bitrate,
	// codec, etc...
	StreamInfo map[string]string

	// The duration of the playlist.
	Duration float64

	// The target duration of the playlist.
	TargetDuration int

	// The content of the playlist.
	Content string

	language string

	codec Codec

	resolution string

	channelLayout string
}

/*
Given a `stream_info` and the `dir_name`, this method finds the media

	playlist file, parses it, and stores relevant parts of the playlist in `self.content`.

It also updates the segment paths to make it relative to `output_dir`

	A `streams_map` is used to match this media playlist to its OutputStream object.
*/
func NewMediaPlaylist(streamInfo map[string]string, dirName, outputDir string, streamsMap map[string]*OutputStream) *MediaPlaylist {
	// If there is a file to read, we MUST have a streams_map to match this
	// media playlist file with its OutputStream.
	if streamsMap == nil {
		panic("streamsMap is required")
	}

	mp := &MediaPlaylist{
		StreamInfo:     streamInfo,
		Duration:       0.0,
		TargetDuration: 0,
		Content:        "",
		HeaderTags:     []string{"#EXTM3U", "#EXT-X-VERSION", "#EXT-X-PLAYLIST-TYPE"},
	}

	if dirName == "" {
		// Do not read, The content will be added manually.
		return mp
	}

	periodDir, err := filepath.Rel(outputDir, dirName)

	if err != nil {
		panic(err)
	}

	mediaPlaylistFile := filepath.Join(dirName, unquote(streamInfo["URI"]))

	f, err := os.Open(mediaPlaylistFile)

	if err != nil {
		panic(err)
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#EXTINF") {
			// Add this segment duration to the total duration.
			// This will be used to re-calculate the average bitrate.
			dur, err := strconv.ParseFloat(strings.Split(line[len("#EXTINF:"):], ",")[0], 64)
			if err != nil {
				panic(err)
			}

			mp.Duration += dur
			mp.Content += line + "\n"
			scanner.Scan()
			line = scanner.Text()
			// If a byterange exists, add it to the content.
			if strings.HasPrefix(line, "#EXT-X-BYTERANGE") {
				mp.Content += line + "\n"
				scanner.Scan()
				line = scanner.Text()
			}
			// Update the segment's URI.
			mp.Content += filepath.Join(periodDir, line) + "\n"
		} else if strings.HasPrefix(line, "#EXT-X-MAP") {
			// An EXT-X-MAP must have a URI attribute and optionally
			// a BYTERANGE attribute.
			attribs := extractAttributes(line)
			mp.Content += "#EXT-X-MAP:URI=" + quote(filepath.Join(periodDir, unquote(attribs["URI"])))
			if byteRange, ok := attribs["BYTERANGE"]; ok {
				mp.Content += ",BYTERANGE=" + byteRange
			}

			mp.Content += "\n"
		} else if !strings.HasPrefix(line, "#EXT") {
			// Skip comments.
			mp.Content += line + "\n"
		} else if strings.HasPrefix(line, "#EXT-X-TARGETDURATION") {
			td, err := strconv.Atoi(line[len("#EXT-X-TARGETDURATION:"):])

			if err != nil {
				panic(err)
			}

			mp.TargetDuration = td

			mp.Content += line + "\n"
		} else if StartsWithAny(line, append(mp.HeaderTags, "#EXT-X-ENDLIST")) {
			// Skip header and end-list tags.
		} else {
			// Store lines that didn't match one of the above cases.
			// Like ENCRYPTIONKEYS, DISCONTINUITIES, COMMENTS, etc... .
			mp.Content += line + "\n"
		}
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	// Set the features we need to access easily while performing the concatenation.
	// Features like codec, channel_layout, resolution, etc... .
	mp.setFeatures(streamsMap)

	return mp
}

/*
Get the audio and video codecs and other relevant stream features

	from the matching OutputStream in the `streamsMap`, this will be used
	in the codec matching process in the concat_xxx() methods, but the codecs
	that will be written in the final concatenated master playlist will be
	the codecs from streamInfo dictionary(in HLS syntax).
*/
func (p *MediaPlaylist) setFeatures(streamsMap map[string]*OutputStream) {
	// We can't depend on the stream_info['CODECS'] to get the codecs from because
	// this is only present for STREAM-INF, this makes it harder to get codecs for
	// audio segments. Also if we try to get the audio codecs from one of the
	// #EXT-X-STREAM-INF tags we will have to match these codecs with each stream,
	// for example, a codec attribute might be CODECS="avc1,ac-3,mp4a,opus",
	// the video codec is put first then the audio codecs are put in
	// lexicographical order(by observation), which isn't necessary the same order of
	// #EXT-X-MEDIA in the master playlist, thus there is no solid baseground for
	// matching the codecs using the information in the master playlist.

	var outputStream MediaOutputStream = &OutputStream{}

	lines := strings.Split(p.Content, "\n")
	// We need to seek to the first line with the tag #EXTINF.  The line after
	// will have the URI we need, but if we encounter a byterange tag we need to
	// advance one more line.
	for i, line := range lines {
		// Don't use the URIs from any tag to try to extract codec information.
		// We should not rely on the exact structure of file names for this.
		// Use stream_maps instead.
		if strings.HasPrefix(line, "#EXTINF") {
			line = lines[i+1]
			if strings.HasPrefix(line, "#EXT-X-BYTERANGE") {
				line = lines[i+2]
			}
			fileName := path.Base(line)
			// Index the fileName and don't use map access directly.
			// There MUST be a match.
			outputStream = streamsMap[fileName]
			break
		}
	}

	if outputStream == nil {
		panic("No media file found in this media playlist")
	}

	p.codec = outputStream.GetCodec()

	switch outputStream.(type) {
	case *VideoOutputStream:
		// p.resolution = outputStream.(*VideoOutputStream).Resolution()
	case *AudioOutputStream:
		// p.channelLayout = outputStream.(*AudioOutputStream).Layout()
		// We will get the language from the stream_info because the stream information
		// is provided by Packager.  We might have mixed 3-letter and 2-letter format
		// from the Streamer, but Packager reduces them all to 2-letter language tag.
		if lang, ok := p.StreamInfo["LANGUAGE"]; ok {
			p.language = unquote(lang)
		} else {
			p.language = "und"
		}
	case *TextOutputStream:
		if lang, ok := p.StreamInfo["LANGUAGE"]; ok {
			p.language = unquote(lang)
		} else {
			p.language = "und"
		}

	default:
		panic("No stream found!")
	}
}

// Extracts attributes from an m3u8 #EXT-X tag to a python dictionary.
func extractAttributes(line string) map[string]string {
	attributes := make(map[string]string)
	line = strings.SplitN(line, ":", 2)[1]
	// For a tighter search, append ',' and search for it in the regex.
	line += ","
	// Search for all KEY=VALUE,
	regex := regexp.MustCompile(`([-A-Z]+)=("[^"]*"|[^",]*),`)
	matches := regex.FindAllStringSubmatch(line, -1)

	for _, match := range matches {
		key := match[1]
		value := match[2]
		attributes[key] = strings.Trim(value, "\"")
	}

	return attributes
}

// Puts a string in double quotes.
func quote(str string) string {
	return "\"" + str + "\""
}

// Removes the double quotes surrounding a string.
func unquote(str string) string {
	return str[1 : len(str)-1]
}

package subsonic

// envelope is the top-level "subsonic-response" wrapper every endpoint
// returns.
type envelope struct {
	Response response `json:"subsonic-response"`
}

type response struct {
	Status        string        `json:"status"`
	Version       string        `json:"version"`
	Error         *responseErr  `json:"error,omitempty"`
	Artists       *ArtistsIndex `json:"artists,omitempty"`
	Artist        *Artist       `json:"artist,omitempty"`
	Album         *Album        `json:"album,omitempty"`
	AlbumList2    *albumList2   `json:"albumList2,omitempty"`
	Genres        *genresList   `json:"genres,omitempty"`
	Song          *Song         `json:"song,omitempty"`
	SearchResult3 *SearchResult `json:"searchResult3,omitempty"`
	Playlists     *playlists    `json:"playlists,omitempty"`
	MusicFolders  *musicFolders `json:"musicFolders,omitempty"`
	Lyrics        *LyricsPlain  `json:"lyrics,omitempty"`
	LyricsList    *LyricsList   `json:"lyricsList,omitempty"`
	User          *User         `json:"user,omitempty"`
	License       *License      `json:"license,omitempty"`
}

type responseErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ArtistsIndex is the response of getArtists.view, artists grouped by
// starting letter.
type ArtistsIndex struct {
	Index []struct {
		Name    string   `json:"name"`
		Artists []Artist `json:"artist"`
	} `json:"index"`
}

// Artist is a single artist, optionally with its albums when returned from
// getArtist.view.
type Artist struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	AlbumCount int     `json:"albumCount"`
	CoverArt   string  `json:"coverArt,omitempty"`
	Album      []Album `json:"album,omitempty"`
}

// Album is a single album, optionally with its songs when returned from
// getAlbum.view.
type Album struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Artist      string `json:"artist"`
	ArtistID    string `json:"artistId"`
	CoverArt    string `json:"coverArt,omitempty"`
	SongCount   int    `json:"songCount"`
	Duration    int    `json:"duration"`
	Year        int    `json:"year,omitempty"`
	Genre       string `json:"genre,omitempty"`
	AccentColor string `json:"accentColor,omitempty"` // Sonora extension, SPECS §7
	Song        []Song `json:"song,omitempty"`
}

// Song is a single track.
type Song struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Album       string `json:"album,omitempty"`
	AlbumID     string `json:"albumId,omitempty"`
	Artist      string `json:"artist,omitempty"`
	ArtistID    string `json:"artistId,omitempty"`
	Track       int    `json:"track,omitempty"`
	Duration    int    `json:"duration,omitempty"`
	CoverArt    string `json:"coverArt,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	Suffix      string `json:"suffix,omitempty"`
}

type albumList2 struct {
	Album []Album `json:"album"`
}

type genresList struct {
	Genre []Genre `json:"genre"`
}

// Genre is a single genre with its item counts.
type Genre struct {
	Value      string `json:"value"`
	SongCount  int    `json:"songCount"`
	AlbumCount int    `json:"albumCount"`
}

// SearchResult is the response of search3.view.
type SearchResult struct {
	Artist []Artist `json:"artist,omitempty"`
	Album  []Album  `json:"album,omitempty"`
	Song   []Song   `json:"song,omitempty"`
}

type playlists struct {
	Playlist []Playlist `json:"playlist"`
}

// Playlist is a single server-side playlist (read-only in v1, SPECS §2).
type Playlist struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Owner     string `json:"owner,omitempty"`
	SongCount int    `json:"songCount"`
	Duration  int    `json:"duration"`
}

type musicFolders struct {
	MusicFolder []MusicFolder `json:"musicFolder"`
}

// MusicFolder is a top-level library root configured on the server.
type MusicFolder struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// LyricsPlain is the response of getLyrics.view — unsynced plain text.
type LyricsPlain struct {
	Artist string `json:"artist,omitempty"`
	Title  string `json:"title,omitempty"`
	Value  string `json:"value,omitempty"`
}

// LyricsList is the response of getLyricsBySongId.view — OpenSubsonic
// structured, millisecond-timestamped lyrics.
type LyricsList struct {
	StructuredLyrics []StructuredLyrics `json:"structuredLyrics"`
}

// StructuredLyrics is one synced-lyrics track for a song, possibly one of
// several languages/sources the server offers.
type StructuredLyrics struct {
	Lang          string      `json:"lang"`
	Synced        bool        `json:"synced"`
	Line          []LyricLine `json:"line"`
	DisplayArtist string      `json:"displayArtist,omitempty"`
	DisplayTitle  string      `json:"displayTitle,omitempty"`
}

// LyricLine is a single lyrics line, timestamped in milliseconds when Synced
// is true on the parent StructuredLyrics.
type LyricLine struct {
	Start int    `json:"start,omitempty"`
	Value string `json:"value"`
}

// User is the response of getUser.view.
type User struct {
	Username  string `json:"username"`
	AdminRole bool   `json:"adminRole,omitempty"`
}

// License is the response of getLicense.view.
type License struct {
	Valid bool `json:"valid"`
}

package subsonic

import (
	"context"
	"fmt"
	"net/url"
)

// Ping verifies connectivity and credentials against the server.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.get(ctx, "ping.view", nil)
	if err != nil {
		return fmt.Errorf("subsonic: ping: %w", err)
	}
	return nil
}

// GetArtists returns the full artist index, grouped by starting letter.
func (c *Client) GetArtists(ctx context.Context) (*ArtistsIndex, error) {
	res, err := c.get(ctx, "getArtists.view", nil)
	if err != nil {
		return nil, fmt.Errorf("subsonic: get artists: %w", err)
	}
	return res.Artists, nil
}

// GetArtist returns one artist with its albums.
func (c *Client) GetArtist(ctx context.Context, id string) (*Artist, error) {
	v := url.Values{}
	v.Set("id", id)
	res, err := c.get(ctx, "getArtist.view", v)
	if err != nil {
		return nil, fmt.Errorf("subsonic: get artist %s: %w", id, err)
	}
	return res.Artist, nil
}

// GetAlbum returns one album with its songs.
func (c *Client) GetAlbum(ctx context.Context, id string) (*Album, error) {
	v := url.Values{}
	v.Set("id", id)
	res, err := c.get(ctx, "getAlbum.view", v)
	if err != nil {
		return nil, fmt.Errorf("subsonic: get album %s: %w", id, err)
	}
	return res.Album, nil
}

// AlbumListType selects the ordering for GetAlbumList2.
type AlbumListType string

const (
	AlbumListRecent   AlbumListType = "recent"
	AlbumListNewest   AlbumListType = "newest"
	AlbumListFrequent AlbumListType = "frequent"
	AlbumListRandom   AlbumListType = "random"
)

// GetAlbumList2 returns a page of albums ordered by listType.
func (c *Client) GetAlbumList2(ctx context.Context, listType AlbumListType, size, offset int) ([]Album, error) {
	v := url.Values{}
	v.Set("type", string(listType))
	if size > 0 {
		v.Set("size", fmt.Sprintf("%d", size))
	}
	if offset > 0 {
		v.Set("offset", fmt.Sprintf("%d", offset))
	}
	res, err := c.get(ctx, "getAlbumList2.view", v)
	if err != nil {
		return nil, fmt.Errorf("subsonic: get album list (%s): %w", listType, err)
	}
	if res.AlbumList2 == nil {
		return nil, nil
	}
	return res.AlbumList2.Album, nil
}

// GetGenres returns every genre known to the server.
func (c *Client) GetGenres(ctx context.Context) ([]Genre, error) {
	res, err := c.get(ctx, "getGenres.view", nil)
	if err != nil {
		return nil, fmt.Errorf("subsonic: get genres: %w", err)
	}
	if res.Genres == nil {
		return nil, nil
	}
	return res.Genres.Genre, nil
}

// GetSong returns metadata for one track.
func (c *Client) GetSong(ctx context.Context, id string) (*Song, error) {
	v := url.Values{}
	v.Set("id", id)
	res, err := c.get(ctx, "getSong.view", v)
	if err != nil {
		return nil, fmt.Errorf("subsonic: get song %s: %w", id, err)
	}
	return res.Song, nil
}

// GetLyricsBySongID returns OpenSubsonic structured, millisecond-timestamped
// lyrics for id. An empty result (no error) means the server has none.
func (c *Client) GetLyricsBySongID(ctx context.Context, id string) (*LyricsList, error) {
	v := url.Values{}
	v.Set("id", id)
	res, err := c.get(ctx, "getLyricsBySongId.view", v)
	if err != nil {
		return nil, fmt.Errorf("subsonic: get lyrics by song id %s: %w", id, err)
	}
	return res.LyricsList, nil
}

// GetLyrics is the legacy plain-text, unsynced lyrics fallback, looked up by
// artist and title rather than song id.
func (c *Client) GetLyrics(ctx context.Context, artist, title string) (*LyricsPlain, error) {
	v := url.Values{}
	v.Set("artist", artist)
	v.Set("title", title)
	res, err := c.get(ctx, "getLyrics.view", v)
	if err != nil {
		return nil, fmt.Errorf("subsonic: get lyrics for %s - %s: %w", artist, title, err)
	}
	return res.Lyrics, nil
}

// Search3 performs a combined artist/album/song search.
func (c *Client) Search3(ctx context.Context, query string) (*SearchResult, error) {
	v := url.Values{}
	v.Set("query", query)
	res, err := c.get(ctx, "search3.view", v)
	if err != nil {
		return nil, fmt.Errorf("subsonic: search3 %q: %w", query, err)
	}
	return res.SearchResult3, nil
}

// Scrobble registers playback of id. submission=false marks now-playing;
// submission=true marks a completed play, per the server's scrobble
// threshold policy.
func (c *Client) Scrobble(ctx context.Context, id string, submission bool) error {
	v := url.Values{}
	v.Set("id", id)
	if submission {
		v.Set("submission", "true")
	} else {
		v.Set("submission", "false")
	}
	_, err := c.get(ctx, "scrobble.view", v)
	if err != nil {
		return fmt.Errorf("subsonic: scrobble %s: %w", id, err)
	}
	return nil
}

// GetPlaylists returns every server-side playlist visible to the
// authenticated user. Playlists are read-only in v1 (SPECS §2).
func (c *Client) GetPlaylists(ctx context.Context) ([]Playlist, error) {
	res, err := c.get(ctx, "getPlaylists.view", nil)
	if err != nil {
		return nil, fmt.Errorf("subsonic: get playlists: %w", err)
	}
	if res.Playlists == nil {
		return nil, nil
	}
	return res.Playlists.Playlist, nil
}

// GetMusicFolders returns the server's configured library roots.
func (c *Client) GetMusicFolders(ctx context.Context) ([]MusicFolder, error) {
	res, err := c.get(ctx, "getMusicFolders.view", nil)
	if err != nil {
		return nil, fmt.Errorf("subsonic: get music folders: %w", err)
	}
	if res.MusicFolders == nil {
		return nil, nil
	}
	return res.MusicFolders.MusicFolder, nil
}

// GetOpenSubsonicExtensions reports which OpenSubsonic extensions the server
// supports. The raw envelope is returned as no extension is consumed yet.
func (c *Client) GetOpenSubsonicExtensions(ctx context.Context) error {
	_, err := c.get(ctx, "getOpenSubsonicExtensions.view", nil)
	if err != nil {
		return fmt.Errorf("subsonic: get open subsonic extensions: %w", err)
	}
	return nil
}

// GetUser returns the authenticated user's account info.
func (c *Client) GetUser(ctx context.Context, username string) (*User, error) {
	v := url.Values{}
	v.Set("username", username)
	res, err := c.get(ctx, "getUser.view", v)
	if err != nil {
		return nil, fmt.Errorf("subsonic: get user %s: %w", username, err)
	}
	return res.User, nil
}

// GetLicense reports whether the server's license is valid.
func (c *Client) GetLicense(ctx context.Context) (*License, error) {
	res, err := c.get(ctx, "getLicense.view", nil)
	if err != nil {
		return nil, fmt.Errorf("subsonic: get license: %w", err)
	}
	return res.License, nil
}

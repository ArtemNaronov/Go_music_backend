package musicpath

import "testing"

func TestRelAlbumDirCollapsesDiscFolder(t *testing.T) {
	root := `D:\Music`
	path := `D:\Music\Queen\Greatest Hits\CD1\01.mp3`

	got := RelAlbumDir(root, path)
	want := RelAlbumDir(root, `D:\Music\Queen\Greatest Hits\01.mp3`)
	if got != want {
		t.Fatalf("RelAlbumDir = %q, want %q", got, want)
	}

	artist, album := FolderArtistAlbum(root, path)
	if artist != "Queen" || album != "Greatest Hits" {
		t.Fatalf("names = (%q, %q), want (Queen, Greatest Hits)", artist, album)
	}
}

func TestRelAlbumDirKeepsRegularNestedFolder(t *testing.T) {
	root := `D:\Music`
	path := `D:\Music\Linkin Park\Meteora\01.mp3`

	artist, album := FolderArtistAlbum(root, path)
	if artist != "Linkin Park" || album != "Meteora" {
		t.Fatalf("names = (%q, %q), want (Linkin Park, Meteora)", artist, album)
	}
}

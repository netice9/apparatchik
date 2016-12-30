package public

import "github.com/elazarl/go-bindata-assetfs"

//go:generate go-bindata-assetfs -pkg public -ignore .*\.go  ./

// AssetFS is public accessor to the assetFS() function
func AssetFS() *assetfs.AssetFS {
	afs := assetFS()
	afs.Prefix = "/"
	return afs
}

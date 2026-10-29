package main

import _ "embed"

// appIcon is the mark the app ships as its icon: Settings > About draws the same picture. It is
// embedded so that the binary carries it even when it runs outside its resource folder.
//
//go:embed resources/icon.png
var appIcon []byte

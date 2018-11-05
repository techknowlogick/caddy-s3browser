package main

import (
	"time"
	"fmt"
	"strings"
	"path"
)

type Directory struct {
	Path    string
	CanGoUp bool
	Folders []Folder
	Files   []File
}

type Folder struct {
	Name string
}

type File struct {
	Byetes uint64
	Name   string
	Date   time.Time
}

var x = `
58c182707ec85/config.json
58c177fccbabf/config.json
58c9abc15dd59/cover_Desert.jpg
58cc387f2d69b/images/001_capture.jpg
58cc504819e84/config.json
58cc4cb3ca766/config.json
58cc50ce48223/config.json
58c1720cb8465/audio/001_audio.mp3
58c174e6aaef5/audio/001_audio.mp3
58c176aa1718a/audio/001_audio.mp3
58c1720cb8465/caption/1_jellies.srt
58cc387f2d69b/videos/HD720p.mp4
58cc387f2d69b/videos/SD480p.mp4
58cc387f2d69b/videos/mobile.mp4
58cc3992a421a/audio/001_audio.mp3
58cc264f8720d/original/1_SampleVideo_1280x720_1mb.mp4
58c182707ec85/caption/1_jellies.srt
58c174e6aaef5/caption/1_jellies.srt
58cc55373fd58/cover_DesertDesertDesertDesertDesertDesertDesert.jpg
58c177fccbabf/audio/001_audio.mp3
58c1720cb8465/images/001_capture.jpg
58c1720cb8465/images/user_preview.jpg
58cc39187087b/images/001_capture.jpg
58cc3992a421a/caption/1_jellies.srt
58cc41f67bed7/caption/1_jellies.srt
58cc4aa59c517/caption/1_jellies.srt
58cc4934705c2/caption/1_jellies.srt
58c174e6aaef5/images/001_capture.jpg
58c174e6aaef5/images/user_preview.JPG
58dbf8dfd5e33/config.json
58cc47b110928/caption/1_jellies.srt
58c182707ec85/images/001_capture.jpg
58cc4686475f9/caption/1_jellies.srt
58cc4cb3ca766/caption/1_jellies.srt
58cc50ce48223/caption/1_jellies.srt
58c177fccbabf/caption/1_jellies.srt
58cc504819e84/audio/001_audio.mp3
58cc525a5c9b9/audio/001_audio.mp3
58cc3992a421a/images/001_capture.jpg
58cc387f2d69b/original/video/1_LATEST.mp4
58cc4686475f9/images/001_capture.jpg
58cc41f67bed7/images/001_capture.jpg
58cc4aa59c517/images/001_capture.jpg
58e671eb7f3b5/config.json
58cc4934705c2/images/001_capture.jpg
58cc47b110928/images/001_capture.jpg
58cc39187087b/videos/HD720p.mp4
58cc39187087b/videos/SD480p.mp4
58cc39187087b/videos/mobile.mp4
58cc504819e84/caption/1_jellies.srt
58cc4cb3ca766/images/001_capture.jpg
58cc50ce48223/images/001_capture.jpg
58cc50ce48223/images/user_preview.JPG
58c177fccbabf/images/001_capture.jpg
58c177fccbabf/images/user_preview.JPG
58c1720cb8465/videos/HD720p.mp4
58c1720cb8465/videos/SD480p.mp4
58c1720cb8465/videos/mobile.mp4
58c9abc15dd59/original/photos/1_Desert.jpg
58c9abc15dd59/original/photos/1_Hydrangeas.jpg
58c9abc15dd59/original/photos/1_Jellyfish.jpg
58c9abc15dd59/original/photos/1_Koala.jpg
58c176aa1718a/original/audio/1_520.mp3
58c174e6aaef5/videos/HD720p.mp4
`

func main() {

	fs := make(map[string]Directory)

	fs["/"] = Directory{
		Path: "/",
		CanGoUp: false,
	}

	temp := strings.Split(x, "\n") // TODO: from minio-go loop on objects instead

	for _, filePath := range temp {
		if len(filePath) < 1 || filePath == "" {
			continue
		}
		dir, file := path.Split(filePath)
		if len(dir) > 0 && dir[len(dir)-1:] == "/" {
			dir = dir[:len(dir)-1]
		}
		if len(dir) > 0 && dir[0] == "/" {
			dir = dir[1:]
		}
		if dir == "" {
			dir = "/"
		}

		// STEP ONE Check dir hirearchy
		// need to split on directory split
		if _, ok := fs[dir]; !ok {
			tempDir := strings.Split(dir, "/")
			built := ""
			// also loop through breadcrumb to check those as well
			for _, tempFolder := range tempDir {
				if len(tempFolder) < 1 {
					continue
				}
				if len(built) < 1 {
					built = tempFolder
				}else {
					built = built + "/" +tempFolder
				}
				if _, ok2 := fs[built]; !ok2 {
					fs[built] = Directory{
						Path: built,
						CanGoUp: true,
					}
					// also find parent and inject as a folder
					count := strings.Count(built, "/")
					if count > 0 {
						removeEnd := strings.SplitN(built, "/", count-1)
						if len(removeEnd) > 0 {
							removeEnd = removeEnd[:len(removeEnd)-1]
						}
						noEnd := strings.Join(removeEnd,"/")
						tempFs := fs[noEnd]
						tempFs.Folders = append(tempFs.Folders, Folder{Name: built})
						fs[noEnd] = tempFs
					} else {
						tempFs := fs["/"]
						tempFs.Folders = append(tempFs.Folders, Folder{Name: built})
						fs["/"] = tempFs
					}
				}
			}
		} // if hierachy exists?

		// STEP Two
		// add file to directory
		tempFile := File{Name: file}
		y := fs[dir]
		y.Files = append(y.Files, tempFile)
		fs[dir] = y
	}

	fmt.Print(fs)

}

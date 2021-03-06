package main

import (
	"archive/zip"
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"tawesoft.co.uk/go/dialog"
)

func getflags() (int, string, bool) {
	var interval = flag.Int("t", 30, "set interval time in minutes > 1 minute")

	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	var dir_path = flag.String("d", pwd, "set direcotry")

	var git_mode = flag.Bool("git", false, "set git mode")

	flag.Parse()

	return *interval * 60, *dir_path, *git_mode
}

func detect_os(dir_path string) string {
	var zip_path string

	os := runtime.GOOS
	if os == "windows" {
		zip_path = fmt.Sprintf("C:\\tmp\\%s.zip", filepath.Base(dir_path))
	} else {
		zip_path = fmt.Sprintf("/tmp/%s.zip", filepath.Base(dir_path))
	}

	return zip_path
}

func zipper(dir_path, zip_path string) error { //yes, i copied and pasted this.. zipping in Go is too complicated
	f, err := os.Create(zip_path)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := zip.NewWriter(f)
	defer writer.Close()

	return filepath.Walk(dir_path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Method = zip.Deflate

		header.Name, err = filepath.Rel(filepath.Dir(dir_path), path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			header.Name += "/"
		}

		headerWriter, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(headerWriter, f)
		return err
	})
}

func md5sum(path string) string {
	file, err := os.Open(path)

	if err != nil {
		panic(err)
	}

	defer file.Close()

	hash := md5.New()
	_, err = io.Copy(hash, file)

	if err != nil {
		panic(err)
	}

	return string(hash.Sum(nil))
}

func cod(interval int, prevhash, dir_path, zip_path string, git_mode bool) string {
	zipper(dir_path, zip_path)
	hash := md5sum(zip_path)

	if hash == prevhash {
		if git_mode {
			//delete repo content
			exec.Command("git", "rm .")
			exec.Command("git", "push")
		}
		// delete the whole repo
		os.RemoveAll(dir_path)
		sound()
		dialog.Alert("Your project is deleted, you should've been more productive")
	} else {
		return hash
	}

	return ""
}

func sound() {
	f, err := os.Open("beep.mp3")
	if err != nil {
		log.Fatal(err)
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	defer streamer.Close()

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	done := make(chan bool)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))

	<-done
}

func reminder(interval int) {
	sound()
	if float32(interval/4/60) < 1 {
		dialog.Alert("Code-or-die WARNING: %d SECONDS LEFT!", interval/4)
	} else {
		dialog.Alert("Code-or-die WARNING: %.2f MINUTES LEFT!", float32(interval/4))
	}
}

func main() {
	interval, dir_path, git_mode := getflags()

	zip_path := detect_os(dir_path)

	hash := cod(interval, "", dir_path, zip_path, git_mode)

	for {
		time.Sleep(time.Duration(float32(interval*3/4)) * time.Second)
		reminder(interval)
		time.Sleep(time.Duration(float32(interval/4)) * time.Second)

		hash = cod(interval, hash, dir_path, zip_path, git_mode)

	}
}

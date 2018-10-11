package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"libs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

var total = 0

func main() {
	outputDir := os.Args[1]
	fmt.Println(outputDir)

	for {

		file, err := os.Open("./follows.csv")
		if err != nil {
			panicOnError(err)
		}

		reader := bufio.NewReader(file)
		scanner := bufio.NewScanner(reader)

		for scanner.Scan() {
			cubeId := scanner.Text()
			if total > 5 {
				log.Println("There are 5 threads of downloading already")
				break
			}

			if isLocked(outputDir, cubeId) {
				log.Println("Already downloading")
				continue
			}

			download(outputDir, cubeId)
		}

		file.Close()
		time.Sleep(60 * time.Second)
	}
}

func createLockFile(dir, cubeId string) {
	lockFile := dir + "/" + cubeId + ".downloading"
	f, err := os.Create(lockFile)
	panicOnError(err)
	defer f.Close()
}

func isLocked(dir, cubeId string) bool {
	lockFile := dir + "/" + cubeId + ".downloading"
	if _, err := os.Stat(lockFile); !os.IsNotExist(err) {
		return true
	}
	return false
}

func removeLockFile(dir, cubeId string) {
	lockFile := dir + "/" + cubeId + ".downloading"
	err := os.Remove(lockFile)
	panicOnError(err)
}

func download(outputDir, cubeId string) (err error) {
	r, err := http.Get("https://www.cubetv.sg/studio/info?cube_id=" + cubeId)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	var userInfo libs.Map
	json.NewDecoder(r.Body).Decode(&userInfo)
	//fmt.Println(userInfo.GetMap("data"))

	gid := userInfo.GetMap("data").GetString("gid")
	nickName := userInfo.GetMap("data").GetString("nick_name")
	gameTitle := userInfo.GetMap("data").GetString("gameTitle")

	var gameInfo libs.Map
	r, _ = http.Get("https://www.cubetv.sg/studioApi/getStudioSrcBySid?videoType=1&https=1&sid=" + gid)
	json.NewDecoder(r.Body).Decode(&gameInfo)
	code := gameInfo.GetString("code")
	if code != "1" {
		fmt.Println(userInfo.GetMap("data"))
		return nil
	}

	log.Println("Started downloading " + cubeId)
	createLockFile(outputDir, cubeId)
	total++
	go func() error {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v", r)
			}
			total--
			removeLockFile(outputDir, cubeId)
			log.Println("Finished downloading " + cubeId)
		}()
		videoSrc := gameInfo.GetMap("data").GetString("video_src")

		output := fmt.Sprintf(
			"%s/%s-%s-%s-%s.mp4",
			outputDir,
			nickName, gameTitle, cubeId, time.Now().Format("2006-01-02_150405"))

		cmd := exec.Command("ffmpeg", "-i", videoSrc, "-c", "copy", "-bsf:a", "aac_adtstoasc", output)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stdout
		cmd.Stdin = os.Stdin

		time.Sleep(10 * time.Second)
		if err := cmd.Run(); err != nil {
			log.Println("retrying")
			time.Sleep(1 * time.Second)
			if err := cmd.Run(); err != nil {
				return err
			}
		}
		return nil
	}()

	return nil
}

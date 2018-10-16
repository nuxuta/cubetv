package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"libs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
		loop(outputDir)
	}
}

func loop(outputDir string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
			log.Println(err)
		}
	}()
	jsonFile, err := os.Open("config.json")
	panicOnError(err)
	byteValue, err := ioutil.ReadAll(jsonFile)
	panicOnError(err)
	var config libs.Map
	json.Unmarshal(byteValue, &config)
	jsonFile.Close()
	subFolder := config.GetString("subFolder")

	rootDir := filepath.Join(outputDir, subFolder)
	outputFolder := filepath.Join(rootDir, time.Now().Format("2006-01-02_150000"))
	os.MkdirAll(outputFolder, os.ModePerm)

	for _, cubeId := range config.GetArr("follows").ToArrStr() {
		if total >= config.GetInt("limit") {
			log.Printf("Limited %s\n", config.GetString("limit"))
			break
		}
		if isLocked(rootDir, cubeId) {
			log.Println("Already downloading")
			continue
		}
		download(rootDir, outputFolder, cubeId)
	}

	time.Sleep(time.Duration(config.GetInt("delay")) * time.Second)
	return err
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

func download(rootDir, outputDir, cubeId string) (err error) {
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
		log.Println(cubeId, gameInfo)
		return nil
	}

	log.Println("Started downloading " + cubeId)
	createLockFile(rootDir, cubeId)
	total++
	go func() error {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v", r)
			}
			total--
			removeLockFile(rootDir, cubeId)
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

		time.Sleep(5 * time.Second)
		if err := cmd.Run(); err != nil {
			return err
		}
		return nil
	}()

	return nil
}

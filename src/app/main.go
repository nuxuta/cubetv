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
	"syscall"
	"time"
)

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

var total = 0

type Process struct {
	startTime time.Time
	cmd       *exec.Cmd
}

type StreamInfo struct {
	CubeId    string
	VideoSrc  string
	GameTitle string
	NickName  string
}

var processesMap = make([]Process, 0, 10)

func main() {
	configFile := os.Args[1]
	outputDir := os.Args[2]
	for {
		err := loop(configFile, outputDir)
		if err != nil {
			log.Println(err)
		}
	}
}

func loop(configFile, outputDir string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			//panic(r)
			return
		}
	}()
	jsonFile, err := os.Open(configFile)
	panicOnError(err)
	byteValue, err := ioutil.ReadAll(jsonFile)
	panicOnError(err)
	var config libs.Map
	json.Unmarshal(byteValue, &config)
	jsonFile.Close()
	subFolder := config.GetString("subFolder")

	rootDir := filepath.Join(outputDir, subFolder)
	os.MkdirAll(rootDir, os.ModePerm)
	outputFolder := filepath.Join(rootDir, time.Now().Format("2006-01-02"))

	for _, cubeId := range config.GetArr("follows").ToArrStr() {
		if total >= config.GetInt("limit") {
			log.Printf("Limited %s\n", config.GetString("limit"))
			break
		}
		if isLocked(rootDir, cubeId) {
			log.Println("Already downloading")
			continue
		}
		streamInfo, err := getStreamInfo(cubeId)
		if err != nil {
			return err
		}
		if streamInfo == nil {
			continue
		}
		download(rootDir, outputFolder, streamInfo)
	}

	for _, cubeId := range config.GetArr("prefer").ToArrStr() {
		if isLocked(rootDir, cubeId) {
			log.Println("Already downloading")
			continue
		}
		streamInfo, err := getStreamInfo(cubeId)
		if err != nil {
			return err
		}
		if streamInfo == nil {
			continue
		}
		if total >= config.GetInt("limit") && len(processesMap) > 0 {
			process := processesMap[0]
			process.cmd.Process.Signal(syscall.SIGINT)
			processesMap = processesMap[1:]
		}
		download(rootDir, outputFolder, streamInfo)
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

func getStreamInfo(cubeId string) (streamInfo *StreamInfo, err error) {
	r, err := http.Get("https://www.cubetv.sg/studio/info?cube_id=" + cubeId)
	if err != nil {
		return nil, err
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
		return nil, nil
	}
	videoSrc := gameInfo.GetMap("data").GetString("video_src")

	return &StreamInfo{
		CubeId:    cubeId,
		GameTitle: gameTitle,
		NickName:  nickName,
		VideoSrc:  videoSrc,
	}, nil
}

func download(rootDir string, outputDir string, streamInfo *StreamInfo) (err error) {
	log.Println("Started downloading " + streamInfo.CubeId)
	createLockFile(rootDir, streamInfo.CubeId)
	os.MkdirAll(outputDir, os.ModePerm)
	videoSrc := streamInfo.VideoSrc

	output := fmt.Sprintf(
		"%s/%s-%s-%s-%s.mp4",
		outputDir,
		streamInfo.NickName,
		streamInfo.GameTitle,
		streamInfo.CubeId,
		time.Now().Format("2006-01-02_150405"),
	)

	cmd := exec.Command("ffmpeg", "-i", videoSrc, "-c", "copy", output)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	cmd.Stdin = os.Stdin
	processesMap = append(processesMap, Process{
		cmd:       cmd,
		startTime: time.Now(),
	})
	total++
	go func() error {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v", r)
			}
			total--
			removeLockFile(rootDir, streamInfo.CubeId)
			log.Println("Finished downloading " + streamInfo.CubeId)
		}()

		time.Sleep(5 * time.Second)
		go func() {
			time.Sleep(60 * 60 * time.Second)
			cmd.Process.Signal(syscall.SIGINT)
		}()

		if err := cmd.Run(); err != nil {
			return err
		}
		return nil
	}()

	return nil
}

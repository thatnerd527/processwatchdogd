package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"time"
	"net"
	"github.com/fatih/color"
)

func doesFileExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func panicIfNotNillErr(error error) {
	if error != nil {
		panic(error)
	}
}

const (
	ExitWithoutFailure = 0
	ExitUnexpectedly = 1
	ExitBeforeConnection = 2
	Restart = 3
)

func startProcess(commandname string, workingdir string, arguments []string) {
	udpserver, _ := net.ListenPacket("udp","127.0.0.1:0")
	arguments = append(arguments, fmt.Sprint(udpserver.LocalAddr().(*net.UDPAddr).Port));
	fmt.Printf("Starting command %v, at %v\n", commandname, time.Now().String())	
	cmd := exec.Command(commandname, arguments...)
	cmd.Dir = workingdir
	readerErr, err := cmd.StderrPipe()
	panicIfNotNillErr(err)
	readerOut, err := cmd.StdoutPipe()
	panicIfNotNillErr(err)
	readerIn, err := cmd.StdinPipe()
	panicIfNotNillErr(err)
	err = cmd.Start()
	panicIfNotNillErr(err)
	//status := ExitBeforeConnection;
	go func() {
		for {
			buffered := make([]byte, 1024)
			_ , err := readerErr.Read(buffered)
			if (err == io.ErrClosedPipe) {
				break;
			}
			os.Stdout.Write([]byte(color.New(color.BgRed,color.FgWhite).Sprint(string(buffered))));
		}
	}()
	go func() {
		for {
			buffered2 := make([]byte, 1024)
			_ , err := readerOut.Read(buffered2)
			if (err == io.ErrClosedPipe) {
				break;
			}
			fmt.Print(string(buffered2))
		}
	}()
	go func() {
		for {
			buffered2 := make([]byte,1024)
			os.Stdin.Read(buffered2)
			_, err := readerIn.Write(buffered2)
			if (err == io.ErrClosedPipe) {
				break;
			}
		}
	}()
	go func() {
		for {
			var message []byte = make([]byte, 1)
			for {
				buf := make([]byte, 1024)
				n, _, err := udpserver.ReadFrom(buf)
				if err != nil {
					break;
				}
				for _, v := range buf[:n] {
					message = append(message, v)
				}
				if (n <= len(buf)) {
					break;
				}
			}
			//fmt.Print(len(message))
			fmt.Print(string(message))
		}
	}()
	udpconn , _ := net.Dial("udp", udpserver.LocalAddr().String())
	udpconn.Write([]byte("12"))
	cmd.Wait()
}

func main() {
	fmt.Println("Process monitor daemon.")

	var result map[string]interface{}
	var filepath = "config.json"
	if !doesFileExist(filepath) {
		panicIfNotNillErr(os.WriteFile(filepath, []byte(`{
			"commandname": "monitored.exe",
			"workingdir": ".",
			"arguments": [""]
		}`), fs.FileMode(os.O_CREATE)))
	}
	read, err := os.ReadFile(filepath)
	panicIfNotNillErr(err)
	json.Unmarshal(read, &result)

	fmt.Printf("Starting to monitor: %v\n", result["commandname"].(string))
	fmt.Println("Console out, console in and console err are transparently passed.")

	// Run the executable
	var parsedargs []string = make([]string, 0)
	for _, v := range result["arguments"].([]interface{}) {
		parsedargs = append(parsedargs, v.(string))
	}
	startProcess(result["commandname"].(string), result["workingdir"].(string), parsedargs)



	//fmt.Println(result)
}

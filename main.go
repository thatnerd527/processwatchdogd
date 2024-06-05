package main

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"io"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"time"
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
	ExitWithoutFailure   = 0
	ExitUnexpectedly     = 1
	ExitBeforeConnection = 2
	Restart              = 3
)

func removeNulls(slice []byte) []byte {
	var result []byte
	for _, v := range slice {
		if v != 0 {
			result = append(result, v)
		}
	}
	return result
}

func startProcess(commandname string, workingdir string, logdir string, arguments []string) {
	for {
		udpserver, _ := net.ListenPacket("udp", "127.0.0.1:0")
		arguments = append(arguments, fmt.Sprint(udpserver.LocalAddr().(*net.UDPAddr).Port))
		fmt.Printf("Starting command %v, at %v\n", commandname, time.Now().String())
		cmd := exec.Command(commandname, arguments...)
		cmd.Dir = workingdir
		logfilenametemplate := fmt.Sprintf("%v/%v", logdir, time.Now().Format("2006-01-02-15-04-05"))

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
			logfile, err := os.Create(fmt.Sprintf("%v_err.log", logfilenametemplate))
			if err != nil {
				fmt.Println("Error creating error log file.")
				panic(err)
			}
			for {

				buffered := make([]byte, 1024)
				_, err = readerErr.Read(buffered)
				if err == io.ErrClosedPipe {
					logfile.Close()
					break
				}
				os.Stdout.Write([]byte(color.New(color.BgRed, color.FgWhite).Sprint(string(buffered))))
				logfile.Write(removeNulls(buffered))
				logfile.Sync()
			}
		}()
		go func() {
			logfile2, err := os.Create(fmt.Sprintf("%v_out.log", logfilenametemplate))
			if err != nil {
				fmt.Println("Error creating out log file.")
				panic(err)
			}
			for {

				buffered2 := make([]byte, 1024)
				_, err = readerOut.Read(buffered2)
				if err == io.ErrClosedPipe {
					logfile2.Close()

					break
				}
				fmt.Print(string(buffered2))
				logfile2.Write(removeNulls(buffered2))
				logfile2.Sync()

			}
		}()
		go func() {
			for {
				buffered2 := make([]byte, 1024)
				os.Stdin.Read(buffered2)
				_, err := readerIn.Write(buffered2)
				if err == io.ErrClosedPipe {
					break
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
						break
					}
					for _, v := range buf[:n] {
						message = append(message, v)
					}
					if n <= len(buf) {
						break
					}
				}
				//fmt.Print(len(message))
				fmt.Print(string(message))
			}
		}()
		//udpconn , _ := net.Dial("udp", udpserver.LocalAddr().String())
		//udpconn.Write([]byte("12"))
		cmd.Wait()
	}
	//startProcess(commandname,workingdir,arguments);
}

func main() {
	fmt.Println("Process monitor daemon.")

	var result map[string]interface{}
	var filepath = "config.json"
	if !doesFileExist(filepath) {
		panicIfNotNillErr(os.WriteFile(filepath, []byte(`{
			"commandname": "monitored.exe",
			"workingdir": ".",
			"arguments": [""],
			"logdir": "./logs"
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
	startProcess(result["commandname"].(string), result["workingdir"].(string), result["logdir"].(string), parsedargs)

	//fmt.Println(result)
}

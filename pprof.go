package pprof

import (
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"runtime/trace"
	"strings"
)

var path string = ""

func Pprof() {
	go func() {
		curPath, err := getCurrentPath()
		if err != nil {
			fmt.Printf("获取本地路径失败\n")
			return
		}
		//创建文件夹
		path = curPath + "/tmp"
		if !isExists(path) {
			//	创建
			if err := os.Mkdir(path, 0666); err != nil {
				fmt.Printf("创建文件夹tmp失败\n")
				return
			}
		}

		//默认是关闭GC的
		debug.SetGCPercent(100)
		http.HandleFunc("/start", start)
		http.HandleFunc("/stop", stop)
		http.HandleFunc("/gc", gc)
		http.ListenAndServe(":6060", nil)
	}()
}

func start(w http.ResponseWriter, r *http.Request) {
	if err := startTrace(); err != nil {
		fmt.Println("启动trace pprof失败")
		w.Write([]byte("启动trace pprof失败"))
	}
	if err := startCPU(); err != nil {
		fmt.Println("启动cpu pprof失败")
		w.Write([]byte("启动cpu pprof失败"))
	}

	if err := saveMem(); err != nil {
		fmt.Println("存储mem pprof失败")
		w.Write([]byte("存储mem pprof失败"))
	}
	if err := saveBlock(); err != nil {
		fmt.Println("存储groutine block pprof失败")
		w.Write([]byte("存储groutine pprof失败"))
	}
	fmt.Println("启动cpu pprof，存储mem pprof，存储groutine pprof成功")
	w.Write([]byte("启动trace pprof， cpu pprof，存储mem pprof，存储groutine pprof成功"))
}

func stop(w http.ResponseWriter, r *http.Request) {
	stopCpuProf()
	stopTrace()
	fmt.Println("停止trace pprof，cpu pprof成功")
	w.Write([]byte("停止trace pprof，cpu pprof成功"))
}

var fCPU *os.File = nil
var fTrace *os.File = nil

func startCPU() error {
	var err error
	fCPU, err = os.Create(path + "/cpu.prof")
	if err != nil {
		fmt.Println("创建cpu的pprof文件失败，错误:%v", err)
		fCPU.Close()
		return err
	}
	if err := pprof.StartCPUProfile(fCPU); err != nil {
		fmt.Println("写入cpuPprof文件存储失败，错误：%v", err)
		fCPU.Close()
	}
	return nil
}

func stopCpuProf() {
	pprof.StopCPUProfile()
	fCPU.Close()
}

func startTrace() error {
	var err error
	fTrace, err = os.Create(path + "/trace.out")
	if err != nil {
		fmt.Println("创建trace pprof文件失败，错误：", err)
		return err
	}

	if err = trace.Start(fTrace); err != nil {
		fmt.Println("启动trace pprof文件失败，错误：", err)
		return err
	}
	return nil
}

func stopTrace() {
	trace.Stop()
	fTrace.Close()
}

func gc(w http.ResponseWriter, r *http.Request) {
	runtime.GC()
	w.Write([]byte("启动GC"))
}

func saveMem() error {
	f, err := os.Create(path + "/mem.prof")
	if err != nil {
		fmt.Println("创建mem的pprof文件失败，错误:%v", err)
		return err
	}

	if err := pprof.WriteHeapProfile(f); err != nil {
		fmt.Println("写入memPprof文件存储失败，错误：%v", err)
	}
	f.Close()
	return nil
}

func saveBlock() error {
	f, err := os.Create(path + "/block.prof")
	if err != nil {
		fmt.Println("创建block文件存储失败，错误：%v", err)
		return err
	}

	if err := pprof.Lookup("block").WriteTo(f, 0); err != nil {
		fmt.Println("写入block的pprofPprof文件失败，错误：%v", err)
	}
	f.Close()
	return nil
}

func getCurrentPath() (string, error) {
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		return "", err
	}
	path, err := filepath.Abs(file)
	if err != nil {
		return "", err
	}
	i := strings.LastIndex(path, "/")
	if i < 0 {
		i = strings.LastIndex(path, "\\")
	}
	if i < 0 {
		return "", errors.New(`error: Can't find "/" or "\".`)
	}
	return string(path[0 : i+1]), nil
}

func isExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

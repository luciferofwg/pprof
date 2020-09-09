package pprof

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path"
	"runtime/debug"
	"runtime/pprof"
	"runtime/trace"
	"time"
)

type gcSwitch string

var (
	_gcSwitch   gcSwitch = gcOff
	srv         *http.Server
	dirFullPath string
	fCpu        *os.File = nil
	fMem        *os.File = nil
	fTrace      *os.File = nil
	fBlock      *os.File = nil
)

const (
	gcOff             = "gcOff"
	gcOn              = "gcOn"
	uriStartWithoutGC = "/startnogc"
	uriStartWithGC    = "/start"
	uriStop           = "/start"
)

type msgResp struct {
	Cpu   string `json:"cpu"`
	Mem   string `json:"mem"`
	Trace string `json:"trace"`
	Block string `json:"block,omitempty"`
}

func init() {
	curPath, err := os.Getwd()
	if err != nil {
		log.Panic(err)
	}

	dirFullPath = path.Join(curPath, "pprof")
	if !isExists(dirFullPath) {
		if err := os.Mkdir(dirFullPath, 0666); err != nil {
			log.Panic(err)
		}
	}
}

func Pprof(port int) {
	addrs := fmt.Sprintf("localhost:%d", port)
	log.Printf("pprof listen addr=[%v]", addrs)

	http.HandleFunc(uriStartWithoutGC, handleStartWithoutGC)
	http.HandleFunc(uriStartWithGC, handleStartWithGC)
	http.HandleFunc(uriStop, handleStop)

	srv = &http.Server{
		Addr:    addrs,
		Handler: nil,
	}
	go func(addr string) {
		if err := srv.ListenAndServe(); err != nil {
			log.Panic(err)
		}
	}(addrs)
}

func Shutdown() error {
	ctx, cancle := context.WithTimeout(context.Background(), time.Duration(1))
	defer cancle()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("stop srv failed,%v", err)
		return err
	}
	return nil
}

func generateFile(filepath string) (*os.File, error) {
	f, err := os.Create(filepath)
	if err != nil {
		return &os.File{}, nil
	}
	return f, nil
}

func handleStartWithoutGC(w http.ResponseWriter, r *http.Request) {
	_gcSwitch = gcOff
	debug.SetGCPercent(-1)
}
func handleStartWithGC(w http.ResponseWriter, r *http.Request) {
	_gcSwitch = gcOff
	debug.SetGCPercent(100)
	handleStart(w, r)
}

func handleStart(w http.ResponseWriter, r *http.Request) {
	log.Printf("recv profile start.")

	filepaths := []string{path.Join(dirFullPath, "cpu.pprof"), path.Join(dirFullPath, "mem.pprof"), path.Join(dirFullPath, "trace.pprof"), path.Join(dirFullPath, "block.pprof")}
	deleteFile(filepaths)
	var err error
	fCpu, err = generateFile(filepaths[0])
	if err != nil {
		log.Printf("generate cpu.pprof failed. %v")
		respErr(w, &msgResp{
			Cpu: "generate cpu.pprof failed.",
		})
		return
	}
	if err := pprof.StartCPUProfile(fCpu); err != nil {
		log.Printf("start cpu profile failed.%v", err)
		respErr(w, &msgResp{
			Cpu: "start cpu profile failed.",
		})
		return
	}

	fMem, err = generateFile(filepaths[1])
	if err != nil {
		log.Printf("generate mem.pprof failed. %v")
		respErr(w, &msgResp{
			Mem: "generate mem.pprof failed.",
		})
		return
	}
	if err := pprof.WriteHeapProfile(fMem); err != nil {
		log.Printf("start mem profile failed.%v", err)
		respErr(w, &msgResp{
			Mem: "start mem profile failed.",
		})
		return
	}

	fTrace, err = generateFile(filepaths[2])
	if err != nil {
		log.Printf("generate trace.pprof failed. %v")
		respErr(w, &msgResp{
			Trace: "generate mem.pprof failed.",
		})
		return
	}
	if err := trace.Start(fTrace); err != nil {
		log.Printf("start trace profile failed.%v", err)
		respErr(w, &msgResp{
			Trace: "start trace profile failed.",
		})
		return
	}

	fBlock, err = generateFile(filepaths[3])
	if err != nil {
		log.Printf("generate block.pprof failed. %v")
		respErr(w, &msgResp{
			Block: "generate block.pprof failed.",
		})
		return
	}
	if err := pprof.Lookup("block").WriteTo(fBlock, 0); err != nil {
		log.Printf("start block profile failed.%v", err)
		respErr(w, &msgResp{
			Block: "start block profile failed.",
		})
		return
	}

	respSucc(w, &msgResp{
		Cpu:   "cpu profile start success",
		Mem:   "mem profile start success",
		Trace: "trace profile start success",
		Block: "block profile start success",
	})
}
func handleStop(w http.ResponseWriter, r *http.Request) {
	log.Printf("recv profile stop.")
	if _gcSwitch == gcOn {
		_gcSwitch = gcOff
		debug.SetGCPercent(100)
	}

	if fCpu != nil {
		pprof.StopCPUProfile()
		fCpu.Close()
	}

	// 内存不需要停止
	fMem.Close()
	if fTrace != nil {
		trace.Stop()
		fTrace.Close()
	}
	// block直接关闭
	fBlock.Close()

	respSucc(w, &msgResp{
		Cpu:   "cpu profile stop success",
		Mem:   "mem profile stop success",
		Trace: "trace profile stop success",
		Block: "block profile stop success",
	})
}

func respErr(w http.ResponseWriter, msg *msgResp) {
	w.WriteHeader(http.StatusInternalServerError)
	valMsg, _ := json.Marshal(msg)
	if _, err := w.Write(valMsg); err != nil {
		log.Printf("write err resp msg failed,%v", err)
	}
}

func respSucc(w http.ResponseWriter, msg *msgResp) {
	w.WriteHeader(http.StatusOK)
	valMsg, _ := json.Marshal(msg)
	if _, err := w.Write(valMsg); err != nil {
		log.Printf("write success resp msg failed,%v", err)
	}
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

func deleteFile(filepaths []string) {
	for _, filepath := range filepaths {
		if isExists(filepath) {

			if err := os.Remove(filepath); err != nil {
				log.Printf("delete %v failed.%v", filepath, err)
			}
		}
	}
}
